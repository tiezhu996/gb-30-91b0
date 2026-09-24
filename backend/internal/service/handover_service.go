package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/dto"
	"github.com/gbadopt/gbadopt/internal/model"
	"github.com/gbadopt/gbadopt/internal/repository"
	"github.com/gbadopt/gbadopt/internal/util"
)

// handoverSlotLayout accepts RFC3339 timestamps from clients.
const handoverSlotLayout = time.RFC3339

// HandoverService implements the handover appointment lifecycle.
type HandoverService struct {
	repo    *repository.HandoverAppointmentRepository
	appRepo *repository.AdoptionApplicationRepository
	orgRepo *repository.OrganizationRepository
	logger  *slog.Logger
}

// NewHandoverService creates a HandoverService.
func NewHandoverService(repo *repository.HandoverAppointmentRepository, appRepo *repository.AdoptionApplicationRepository, orgRepo *repository.OrganizationRepository, logger *slog.Logger) *HandoverService {
	return &HandoverService{repo: repo, appRepo: appRepo, orgRepo: orgRepo, logger: logger}
}

// Offer creates a handover arrangement for an approved application (org side).
func (s *HandoverService) Offer(userID uint, req dto.HandoverOfferRequest) (*model.HandoverAppointment, error) {
	app, err := s.loadOwnedApplication(userID, req.ApplicationID, "org")
	if err != nil {
		return nil, err
	}
	if app.Status != constants.AppStatusApproved {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("Handover[application_id=%d] offer failed: application not approved (status=%s)", app.ID, app.Status))
	}
	if active, err := s.repo.FindActiveByApplication(app.ID); err == nil && active != nil {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("Handover[application_id=%d] offer failed: active appointment id=%d exists", app.ID, active.ID))
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("handover offer active lookup: %w", err)
	}

	deadline, err := time.ParseInLocation(handoverSlotLayout, req.ConfirmBefore, time.Local)
	if err != nil {
		return nil, util.NewAppError(422, constants.CodeValidationError, "confirm_before must be an RFC3339 timestamp")
	}
	deadline = deadline.UTC()
	if !deadline.After(time.Now()) {
		return nil, util.NewAppError(422, constants.CodeValidationError, "confirm_before must be in the future")
	}
	slots, err := parseHandoverSlots(req.Slots)
	if err != nil {
		return nil, err
	}
	for _, slot := range slots {
		if slot.Before(deadline) {
			return nil, util.NewAppError(422, constants.CodeValidationError, "every slot must be no earlier than confirm_before")
		}
	}

	raw, _ := json.Marshal(slots)
	h := &model.HandoverAppointment{
		ApplicationID: app.ID, OrgID: app.OrgID, UserID: app.UserID,
		Location: req.Location, Slots: string(raw), ConfirmBefore: deadline,
		Status: constants.HandoverStatusOffered,
	}
	if err := s.repo.Create(h); err != nil {
		return nil, fmt.Errorf("handover offer create: %w", err)
	}
	s.logger.Info("handover appointment offered", "id", h.ID, "application_id", app.ID)
	return h, nil
}

// Select occupies one offered slot for the adopter.
func (s *HandoverService) Select(userID, id uint, raw string) (*model.HandoverAppointment, error) {
	h, _, err := s.loadOwnedHandover(userID, id, "user")
	if err != nil {
		return nil, err
	}
	if err := s.ensureAlive(h); err != nil {
		return nil, err
	}
	if h.Status != constants.HandoverStatusOffered {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("Handover[id=%d] select failed: status=%s is not offered", id, h.Status))
	}
	slot, err := time.ParseInLocation(handoverSlotLayout, raw, time.Local)
	if err != nil {
		return nil, util.NewAppError(422, constants.CodeValidationError, "slot must be an RFC3339 timestamp")
	}
	slot = slot.UTC()
	if !containsHandoverSlot(h.Slots, slot) {
		return nil, util.NewAppError(422, constants.CodeValidationError, "slot is not among the offered slots")
	}
	if holder, err := s.repo.FindActiveBySlot(h.OrgID, slot, h.ID); err == nil && holder != nil {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("Handover[id=%d] select failed: slot already locked by appointment id=%d", id, holder.ID))
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("handover select slot lookup: %w", err)
	}

	now := time.Now()
	h.Status = constants.HandoverStatusPending
	h.SelectedSlot = &slot
	h.UpdatedAt = now
	// Conditional update plus the partial unique index guard against two
	// adopters grabbing the same org slot concurrently.
	ok, err := s.repo.UpdateStatusIf(h, constants.HandoverStatusOffered)
	if err != nil {
		if errors.Is(err, repository.ErrDuplicate) {
			return nil, util.NewAppError(409, constants.CodeConflict, "该时段已被其他申请占用")
		}
		return nil, fmt.Errorf("handover select update: %w", err)
	}
	if !ok {
		return nil, util.NewAppError(409, constants.CodeConflict, "预约状态已变化，请刷新后重试")
	}
	s.logger.Info("handover slot selected", "id", h.ID, "slot", slot)
	return h, nil
}

// Confirm locks the arrangement from the org side.
func (s *HandoverService) Confirm(userID, id uint) (*model.HandoverAppointment, error) {
	h, _, err := s.loadOwnedHandover(userID, id, "org")
	if err != nil {
		return nil, err
	}
	if err := s.ensureAlive(h); err != nil {
		return nil, err
	}
	if h.Status != constants.HandoverStatusPending {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("Handover[id=%d] confirm failed: status=%s is not pending", id, h.Status))
	}
	now := time.Now()
	h.Status = constants.HandoverStatusConfirmed
	h.ConfirmedAt = &now
	if err := s.repo.Update(h); err != nil {
		return nil, fmt.Errorf("handover confirm update: %w", err)
	}
	s.logger.Info("handover appointment confirmed", "id", h.ID)
	return h, nil
}

// Cancel releases a slot before confirmation; both parties may cancel.
func (s *HandoverService) Cancel(userID, id uint, role string) (*model.HandoverAppointment, error) {
	h, _, err := s.loadOwnedHandover(userID, id, role)
	if err != nil {
		return nil, err
	}
	if err := s.ensureAlive(h); err != nil {
		return nil, err
	}
	if constants.IsHandoverTerminal(h.Status) {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("Handover[id=%d] cancel failed: status=%s is terminal", id, h.Status))
	}
	now := time.Now()
	by := constants.HandoverCancelByUser
	if role == "org" {
		by = constants.HandoverCancelByOrg
	}
	h.Status = constants.HandoverStatusCancelled
	h.CancelledBy = by
	h.CancelledAt = &now
	h.SelectedSlot = nil
	if err := s.repo.Update(h); err != nil {
		return nil, fmt.Errorf("handover cancel update: %w", err)
	}
	s.logger.Info("handover appointment cancelled", "id", h.ID, "by", by)
	return h, nil
}

// ListByApplication returns the full handover history of an application.
func (s *HandoverService) ListByApplication(userID, applicationID uint, role string) ([]model.HandoverAppointment, error) {
	if _, err := s.loadOwnedApplication(userID, applicationID, role); err != nil {
		return nil, err
	}
	s.expireDue()
	items, err := s.repo.ListByApplication(applicationID)
	if err != nil {
		return nil, fmt.Errorf("handover list by application: %w", err)
	}
	return items, nil
}

// ListByUser returns the adopter's handover appointments.
func (s *HandoverService) ListByUser(userID uint) ([]model.HandoverAppointment, error) {
	s.expireDue()
	items, err := s.repo.ListByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("handover list by user: %w", err)
	}
	return items, nil
}

// ListByOrg returns the org's handover appointments.
func (s *HandoverService) ListByOrg(userID uint) ([]model.HandoverAppointment, error) {
	org, err := s.orgRepo.FindByUserID(userID)
	if err != nil {
		return nil, util.NewAppError(403, constants.CodeForbidden, "org profile not found")
	}
	s.expireDue()
	items, err := s.repo.ListByOrg(org.ID)
	if err != nil {
		return nil, fmt.Errorf("handover list by org: %w", err)
	}
	return items, nil
}

// ToResponse enriches an appointment with per-slot lock state for the client.
func (s *HandoverService) ToResponse(h *model.HandoverAppointment) dto.HandoverAppointmentResponse {
	rawSlots := decodeHandoverSlots(h.Slots)
	views := make([]dto.HandoverSlotView, 0, len(rawSlots))
	for _, t := range rawSlots {
		view := dto.HandoverSlotView{Time: t.Format(handoverSlotLayout)}
		if h.SelectedSlot != nil && h.SelectedSlot.Equal(t) {
			view.Locked = true
			view.LockedApplicationID = h.ApplicationID
		} else if holder, err := s.repo.FindActiveBySlot(h.OrgID, t, h.ID); err == nil && holder != nil {
			view.Locked = true
			view.LockedApplicationID = holder.ApplicationID
		}
		views = append(views, view)
	}
	return dto.HandoverAppointmentResponse{
		ID: h.ID, ApplicationID: h.ApplicationID, OrgID: h.OrgID, UserID: h.UserID,
		Location: h.Location, Slots: views, ConfirmBefore: h.ConfirmBefore,
		SelectedSlot: h.SelectedSlot, Status: h.Status, CancelledBy: h.CancelledBy,
		ConfirmedAt: h.ConfirmedAt, CancelledAt: h.CancelledAt,
		CreatedAt: h.CreatedAt, UpdatedAt: h.UpdatedAt,
	}
}

// ToResponses maps a slice to response views.
func (s *HandoverService) ToResponses(items []model.HandoverAppointment) []dto.HandoverAppointmentResponse {
	out := make([]dto.HandoverAppointmentResponse, 0, len(items))
	for i := range items {
		out = append(out, s.ToResponse(&items[i]))
	}
	return out
}

// --- helpers ---

// ensureAlive lazily expires an appointment past its confirmation deadline.
func (s *HandoverService) ensureAlive(h *model.HandoverAppointment) error {
	if (h.Status == constants.HandoverStatusOffered || h.Status == constants.HandoverStatusPending) &&
		time.Now().After(h.ConfirmBefore) {
		now := time.Now()
		h.Status = constants.HandoverStatusExpired
		h.SelectedSlot = nil
		h.UpdatedAt = now
		if err := s.repo.Update(h); err != nil {
			return fmt.Errorf("handover expire update: %w", err)
		}
		s.logger.Info("handover appointment expired at deadline", "id", h.ID)
		return util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("Handover[id=%d] already expired past confirm deadline", h.ID))
	}
	return nil
}

func (s *HandoverService) expireDue() {
	n, err := s.repo.ExpireDue(time.Now())
	if err != nil {
		s.logger.Warn("handover expire due failed", "error", err)
		return
	}
	if n > 0 {
		s.logger.Info("handover appointments expired", "count", n)
	}
}

// loadOwnedApplication loads the application and verifies the caller owns it
// (adopter) or owns its org (org member).
func (s *HandoverService) loadOwnedApplication(userID, applicationID uint, role string) (*model.AdoptionApplication, error) {
	app, err := s.appRepo.FindByID(applicationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("AdoptionApplication[id=%d] not found", applicationID))
		}
		return nil, fmt.Errorf("handover application find: %w", err)
	}
	if err := s.checkParty(userID, role, app); err != nil {
		return nil, err
	}
	return app, nil
}

// loadOwnedHandover loads the appointment, its application and verifies access.
func (s *HandoverService) loadOwnedHandover(userID, id uint, role string) (*model.HandoverAppointment, *model.AdoptionApplication, error) {
	h, err := s.repo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil, util.NewAppError(404, constants.CodeNotFound,
				fmt.Sprintf("HandoverAppointment[id=%d] not found", id))
		}
		return nil, nil, fmt.Errorf("handover find: %w", err)
	}
	app, err := s.appRepo.FindByID(h.ApplicationID)
	if err != nil {
		return nil, nil, fmt.Errorf("handover application find: %w", err)
	}
	if err := s.checkParty(userID, role, app); err != nil {
		return nil, nil, err
	}
	return h, app, nil
}

func (s *HandoverService) checkParty(userID uint, role string, app *model.AdoptionApplication) error {
	switch role {
	case "org":
		org, err := s.orgRepo.FindByUserID(userID)
		if err != nil || org.ID != app.OrgID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("AdoptionApplication[id=%d] access denied: not the org owner", app.ID))
		}
	case "user":
		if app.UserID != userID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("AdoptionApplication[id=%d] access denied: not the applicant", app.ID))
		}
	default:
		return util.NewAppError(403, constants.CodeForbidden, "only the adopter or org may access handover appointments")
	}
	return nil
}

func parseHandoverSlots(raw []string) ([]time.Time, error) {
	if len(raw) == 0 {
		return nil, util.NewAppError(422, constants.CodeValidationError, "slots must not be empty")
	}
	out := make([]time.Time, 0, len(raw))
	seen := make(map[int64]bool, len(raw))
	for _, v := range raw {
		t, err := time.ParseInLocation(handoverSlotLayout, v, time.Local)
		if err != nil {
			return nil, util.NewAppError(422, constants.CodeValidationError, "slots must be RFC3339 timestamps")
		}
		t = t.UTC()
		if seen[t.UnixNano()] {
			continue
		}
		seen[t.UnixNano()] = true
		out = append(out, t)
	}
	return out, nil
}

func decodeHandoverSlots(raw string) []time.Time {
	if raw == "" {
		return nil
	}
	var strs []string
	if err := json.Unmarshal([]byte(raw), &strs); err != nil {
		return nil
	}
	out := make([]time.Time, 0, len(strs))
	for _, v := range strs {
		if t, err := time.Parse(handoverSlotLayout, v); err == nil {
			out = append(out, t)
		}
	}
	return out
}

func containsHandoverSlot(raw string, want time.Time) bool {
	for _, t := range decodeHandoverSlots(raw) {
		if t.Equal(want) {
			return true
		}
	}
	return false
}
