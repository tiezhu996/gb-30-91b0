package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/dto"
	"github.com/gbadopt/gbadopt/internal/model"
	"github.com/gbadopt/gbadopt/internal/repository"
	"github.com/gbadopt/gbadopt/internal/util"
)

// HandoverService implements handover scheduling after an application is approved.
type HandoverService struct {
	db        *gorm.DB
	offerRepo *repository.HandoverOfferRepository
	apptRepo  *repository.HandoverAppointmentRepository
	appRepo   *repository.AdoptionApplicationRepository
	orgRepo   *repository.OrganizationRepository
	logger    *slog.Logger
}

// NewHandoverService creates a HandoverService.
func NewHandoverService(db *gorm.DB, offerRepo *repository.HandoverOfferRepository, apptRepo *repository.HandoverAppointmentRepository, appRepo *repository.AdoptionApplicationRepository, orgRepo *repository.OrganizationRepository, logger *slog.Logger) *HandoverService {
	return &HandoverService{db: db, offerRepo: offerRepo, apptRepo: apptRepo, appRepo: appRepo, orgRepo: orgRepo, logger: logger}
}

// expireDue flips all pending appointments past their deadline to expired.
func (s *HandoverService) expireDue() {
	if n, err := s.apptRepo.MarkExpired(time.Now()); err != nil {
		s.logger.Warn("handover expire sweep failed", "error", err)
	} else if n > 0 {
		s.logger.Info(fmt.Sprintf(constants.LogHandoverExpired, n), "count", n)
	}
}

// CreateOffer publishes an org's handover plan for an approved application.
func (s *HandoverService) CreateOffer(orgUserID uint, req dto.HandoverOfferCreateRequest) (*dto.HandoverOfferView, error) {
	now := time.Now()
	if !req.Deadline.After(now) {
		return nil, util.NewAppError(422, constants.CodeValidationError,
			fmt.Sprintf("HandoverOffer[application_id=%d] deadline must be in the future", req.ApplicationID))
	}
	if req.Location == "" {
		return nil, util.NewAppError(422, constants.CodeValidationError, "HandoverOffer location is required")
	}
	seen := make(map[string]bool, len(req.Slots))
	for i, slot := range req.Slots {
		if !slot.EndAt.After(slot.StartAt) {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("HandoverOffer slots[%d] end_at must be after start_at", i))
		}
		if !slot.StartAt.After(now) {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("HandoverOffer slots[%d] start_at must be in the future", i))
		}
		key := slot.StartAt.Format(time.RFC3339) + "/" + slot.EndAt.Format(time.RFC3339)
		if seen[key] {
			return nil, util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("HandoverOffer slots[%d] duplicated", i))
		}
		seen[key] = true
	}

	app, err := s.appRepo.FindByID(req.ApplicationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("AdoptionApplication[id=%d] not found", req.ApplicationID))
		}
		return nil, fmt.Errorf("handover offer app find: %w", err)
	}
	org, err := s.orgRepo.FindByUserID(orgUserID)
	if err != nil || org.ID != app.OrgID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("HandoverOffer[application_id=%d] create failed: user_id=%d not org owner", req.ApplicationID, orgUserID))
	}
	if app.Status != constants.AppStatusApproved {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("HandoverOffer[application_id=%d] create failed: application not approved (status=%s)", req.ApplicationID, app.Status))
	}

	s.expireDue()
	if active, err := s.apptRepo.FindActiveByApplication(app.ID); err == nil && active != nil {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("HandoverOffer[application_id=%d] create failed: active appointment id=%d status=%s", app.ID, active.ID, active.Status))
	} else if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("handover offer active find: %w", err)
	}

	slotsJSON, err := json.Marshal(req.Slots)
	if err != nil {
		return nil, fmt.Errorf("handover offer slots marshal: %w", err)
	}
	offer := &model.HandoverOffer{
		ApplicationID: app.ID, OrgID: org.ID,
		Location: req.Location, Deadline: req.Deadline, Slots: string(slotsJSON),
	}
	if err := s.offerRepo.Create(offer); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogHandoverOfferCreateFailed, app.ID), "error", err)
		return nil, fmt.Errorf("handover offer create: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogHandoverOfferCreated, offer.ID, app.ID), "id", offer.ID)
	return s.buildOfferView(offer, nil), nil
}

// GetOffer returns the latest offer of an application with slot occupation and the active appointment.
func (s *HandoverService) GetOffer(userID uint, role string, applicationID uint) (*dto.HandoverOfferView, error) {
	app, err := s.appRepo.FindByID(applicationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("AdoptionApplication[id=%d] not found", applicationID))
		}
		return nil, fmt.Errorf("handover offer get app: %w", err)
	}
	if err := s.requireParticipant(userID, role, app); err != nil {
		return nil, err
	}
	s.expireDue()
	return s.latestOfferForApp(app)
}

// LockSlot occupies a slot of an offer for the adopter.
func (s *HandoverService) LockSlot(userID uint, offerID uint, slot dto.HandoverSlot) (*dto.HandoverAppointmentView, error) {
	s.expireDue()

	var created *model.HandoverAppointment
	err := s.db.Transaction(func(tx *gorm.DB) error {
		offer, err := s.offerRepo.FindByID(offerID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("HandoverOffer[id=%d] not found", offerID))
			}
			return fmt.Errorf("handover lock offer find: %w", err)
		}
		app, err := s.appRepo.FindByID(offer.ApplicationID)
		if err != nil {
			return fmt.Errorf("handover lock app find: %w", err)
		}
		if roleErr := s.requireAdopter(userID, app); roleErr != nil {
			return roleErr
		}
		if !offer.Deadline.After(time.Now()) {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("HandoverOffer[id=%d] lock failed: confirmation deadline passed", offerID))
		}
		if !s.offerHasSlot(offer, slot.StartAt, slot.EndAt) {
			return util.NewAppError(422, constants.CodeValidationError,
				fmt.Sprintf("HandoverOffer[id=%d] lock failed: slot not offered", offerID))
		}
		if active, e := s.apptRepo.FindActiveBySlot(tx, offer.OrgID, slot.StartAt, slot.EndAt); e == nil && active != nil {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("HandoverOffer[id=%d] lock failed: slot occupied by appointment id=%d", offerID, active.ID))
		} else if e != nil && !errors.Is(e, repository.ErrNotFound) {
			return fmt.Errorf("handover lock slot find: %w", e)
		}
		if active, e := s.apptRepo.FindActiveByApplication(app.ID); e == nil && active != nil {
			return util.NewAppError(409, constants.CodeConflict,
				fmt.Sprintf("HandoverOffer[id=%d] lock failed: application already holds appointment id=%d", offerID, active.ID))
		} else if e != nil && !errors.Is(e, repository.ErrNotFound) {
			return fmt.Errorf("handover lock appt find: %w", e)
		}
		created = &model.HandoverAppointment{
			OfferID: offer.ID, ApplicationID: app.ID, OrgID: offer.OrgID, UserID: userID,
			Location: offer.Location, Deadline: offer.Deadline,
			StartAt: slot.StartAt, EndAt: slot.EndAt, Status: constants.HandoverApptPending,
		}
		if e := s.apptRepo.CreateTx(tx, created); e != nil {
			if errors.Is(e, repository.ErrDuplicate) {
				return util.NewAppError(409, constants.CodeConflict,
					fmt.Sprintf("HandoverOffer[id=%d] lock failed: slot just taken", offerID))
			}
			return fmt.Errorf("handover lock create: %w", e)
		}
		return nil
	})
	if err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogHandoverLockFailed, offerID), "error", err)
		return nil, err
	}
	s.logger.Info(fmt.Sprintf(constants.LogHandoverLocked, created.ID, offerID), "id", created.ID)
	return appointmentView(created), nil
}

// Cancel releases a pending appointment. Both the org and the adopter may cancel before confirmation.
func (s *HandoverService) Cancel(userID uint, role string, id uint) (*dto.HandoverAppointmentView, error) {
	s.expireDue()
	a, err := s.loadAppointmentForParticipant(userID, role, id)
	if err != nil {
		return nil, err
	}
	if a.Status != constants.HandoverApptPending {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("HandoverAppointment[id=%d] cancel failed: status=%s is not cancellable", id, a.Status))
	}
	a.Status = constants.HandoverApptCancelled
	a.CancelledBy = userID
	if err := s.apptRepo.Update(a); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogHandoverCancelFailed, id), "error", err)
		return nil, fmt.Errorf("handover cancel update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogHandoverCancelled, id, userID), "id", id)
	return appointmentView(a), nil
}

// Confirm marks a pending appointment as confirmed (org side).
func (s *HandoverService) Confirm(orgUserID, id uint) (*dto.HandoverAppointmentView, error) {
	s.expireDue()
	a, err := s.loadAppointmentForOrg(orgUserID, id)
	if err != nil {
		return nil, err
	}
	if a.Status != constants.HandoverApptPending {
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("HandoverAppointment[id=%d] confirm failed: status=%s is not pending", id, a.Status))
	}
	if !a.Deadline.After(time.Now()) {
		a.Status = constants.HandoverApptExpired
		_ = s.apptRepo.Update(a)
		return nil, util.NewAppError(409, constants.CodeConflict,
			fmt.Sprintf("HandoverAppointment[id=%d] confirm failed: deadline passed, appointment expired", id))
	}
	now := time.Now()
	a.Status = constants.HandoverApptConfirmed
	a.ConfirmedAt = &now
	if err := s.apptRepo.Update(a); err != nil {
		s.logger.Error(fmt.Sprintf(constants.LogHandoverConfirmFailed, id), "error", err)
		return nil, fmt.Errorf("handover confirm update: %w", err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogHandoverConfirmed, id), "id", id)
	return appointmentView(a), nil
}

// EnrichApplications attaches the latest offer (with slot states) and appointment history.
func (s *HandoverService) EnrichApplications(items []model.AdoptionApplication) []dto.ApplicationView {
	views := make([]dto.ApplicationView, 0, len(items))
	if len(items) == 0 {
		return views
	}
	s.expireDue()

	ids := make([]uint, 0, len(items))
	for _, a := range items {
		ids = append(ids, a.ID)
	}
	offers, err := s.offerRepo.ListByApplications(ids)
	if err != nil {
		s.logger.Warn("handover enrich offers failed", "error", err)
	}
	appts, err := s.apptRepo.ListByApplications(ids)
	if err != nil {
		s.logger.Warn("handover enrich appointments failed", "error", err)
	}

	latestOffer := make(map[uint]model.HandoverOffer)
	for _, o := range offers { // ordered id DESC: first wins = newest
		if _, ok := latestOffer[o.ApplicationID]; !ok {
			latestOffer[o.ApplicationID] = o
		}
	}
	history := make(map[uint][]model.HandoverAppointment)
	active := make(map[uint]model.HandoverAppointment)
	for _, a := range appts { // ordered id DESC
		history[a.ApplicationID] = append(history[a.ApplicationID], a)
		if constants.IsActiveHandoverAppointmentStatus(a.Status) {
			if _, ok := active[a.ApplicationID]; !ok {
				active[a.ApplicationID] = a
			}
		}
	}

	for _, a := range items {
		v := dto.ApplicationView{
			ID: a.ID, UserID: a.UserID, PetID: a.PetID, OrgID: a.OrgID,
			Questionnaire: a.Questionnaire, Status: a.Status,
			CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		}
		if o, ok := latestOffer[a.ID]; ok {
			offerCopy := o
			var apptPtr *model.HandoverAppointment
			if ap, ok2 := active[a.ID]; ok2 {
				apCopy := ap
				apptPtr = &apCopy
			}
			v.Handover = s.buildOfferView(&offerCopy, apptPtr)
		}
		if hs := history[a.ID]; len(hs) > 0 {
			v.History = make([]dto.HandoverAppointmentView, 0, len(hs))
			for _, h := range hs {
				v.History = append(v.History, *appointmentView(&h))
			}
		}
		views = append(views, v)
	}
	return views
}

// --- helpers ---

func (s *HandoverService) requireParticipant(userID uint, role string, app *model.AdoptionApplication) error {
	if role == "admin" {
		return nil
	}
	if role == "org" {
		org, err := s.orgRepo.FindByUserID(userID)
		if err != nil || org.ID != app.OrgID {
			return util.NewAppError(403, constants.CodeForbidden,
				fmt.Sprintf("AdoptionApplication[id=%d] access failed: user_id=%d not org owner", app.ID, userID))
		}
		return nil
	}
	if app.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("AdoptionApplication[id=%d] access failed: not owner", app.ID))
	}
	return nil
}

func (s *HandoverService) requireAdopter(userID uint, app *model.AdoptionApplication) error {
	if app.UserID != userID {
		return util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("AdoptionApplication[id=%d] handover lock failed: not applicant", app.ID))
	}
	return nil
}

func (s *HandoverService) loadAppointmentForParticipant(userID uint, role string, id uint) (*model.HandoverAppointment, error) {
	a, err := s.apptRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("HandoverAppointment[id=%d] not found", id))
		}
		return nil, fmt.Errorf("handover appt find: %w", err)
	}
	app, err := s.appRepo.FindByID(a.ApplicationID)
	if err != nil {
		return nil, fmt.Errorf("handover appt app find: %w", err)
	}
	if err := s.requireParticipant(userID, role, app); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *HandoverService) loadAppointmentForOrg(orgUserID, id uint) (*model.HandoverAppointment, error) {
	a, err := s.apptRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(404, constants.CodeNotFound, fmt.Sprintf("HandoverAppointment[id=%d] not found", id))
		}
		return nil, fmt.Errorf("handover confirm find: %w", err)
	}
	org, err := s.orgRepo.FindByUserID(orgUserID)
	if err != nil || org.ID != a.OrgID {
		return nil, util.NewAppError(403, constants.CodeForbidden,
			fmt.Sprintf("HandoverAppointment[id=%d] confirm failed: user_id=%d not org owner", id, orgUserID))
	}
	return a, nil
}

func (s *HandoverService) latestOfferForApp(app *model.AdoptionApplication) (*dto.HandoverOfferView, error) {
	offer, err := s.offerRepo.FindLatestByApplication(app.ID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil // no handover scheduled yet
		}
		return nil, fmt.Errorf("handover offer find latest: %w", err)
	}
	var apptPtr *model.HandoverAppointment
	if active, e := s.apptRepo.FindActiveByApplication(app.ID); e == nil && active != nil {
		apptPtr = active
	} else if e != nil && !errors.Is(e, repository.ErrNotFound) {
		return nil, fmt.Errorf("handover offer active find: %w", e)
	}
	return s.buildOfferView(offer, apptPtr), nil
}

func (s *HandoverService) offerHasSlot(offer *model.HandoverOffer, start, end time.Time) bool {
	var slots []dto.HandoverSlot
	if err := json.Unmarshal([]byte(offer.Slots), &slots); err != nil {
		s.logger.Warn("handover offer slots unmarshal failed", "offer_id", offer.ID, "error", err)
		return false
	}
	for _, slot := range slots {
		if slot.StartAt.Equal(start) && slot.EndAt.Equal(end) {
			return true
		}
	}
	return false
}

// buildOfferView renders an offer, annotating each slot with free/locked_by_me/occupied.
func (s *HandoverService) buildOfferView(offer *model.HandoverOffer, active *model.HandoverAppointment) *dto.HandoverOfferView {
	var slots []dto.HandoverSlot
	if err := json.Unmarshal([]byte(offer.Slots), &slots); err != nil {
		s.logger.Warn("handover offer slots unmarshal failed", "offer_id", offer.ID, "error", err)
		slots = nil
	}
	views := make([]dto.HandoverSlotView, 0, len(slots))
	for _, slot := range slots {
		sv := dto.HandoverSlotView{StartAt: slot.StartAt, EndAt: slot.EndAt, Status: "free"}
		if active != nil && active.StartAt.Equal(slot.StartAt) && active.EndAt.Equal(slot.EndAt) {
			sv.Status = "locked_by_me"
			sv.ApplicationID = active.ApplicationID
		} else {
			if other, err := s.apptRepo.FindActiveBySlot(nil, offer.OrgID, slot.StartAt, slot.EndAt); err == nil && other != nil {
				sv.Status = "occupied"
				sv.ApplicationID = other.ApplicationID
			}
		}
		views = append(views, sv)
	}
	v := &dto.HandoverOfferView{
		ID: offer.ID, ApplicationID: offer.ApplicationID, OrgID: offer.OrgID,
		Location: offer.Location, Deadline: offer.Deadline, Slots: views,
		CreatedAt: offer.CreatedAt,
	}
	if active != nil {
		v.Appointment = appointmentView(active)
	}
	return v
}

func appointmentView(a *model.HandoverAppointment) *dto.HandoverAppointmentView {
	return &dto.HandoverAppointmentView{
		ID: a.ID, OfferID: a.OfferID, ApplicationID: a.ApplicationID, OrgID: a.OrgID,
		UserID: a.UserID, Location: a.Location, Deadline: a.Deadline,
		StartAt: a.StartAt, EndAt: a.EndAt, Status: a.Status,
		CancelledBy: a.CancelledBy, ConfirmedAt: a.ConfirmedAt,
		CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
	}
}
