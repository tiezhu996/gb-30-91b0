package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/model"
)

// HandoverAppointmentRepository handles handover appointment persistence.
type HandoverAppointmentRepository struct{ db *gorm.DB }

// NewHandoverAppointmentRepository creates the repository.
func NewHandoverAppointmentRepository(db *gorm.DB) *HandoverAppointmentRepository {
	return &HandoverAppointmentRepository{db: db}
}

// Create inserts a handover appointment.
func (r *HandoverAppointmentRepository) Create(h *model.HandoverAppointment) error {
	return translate(r.db.Create(h).Error)
}

// FindByID locates a handover appointment by id.
func (r *HandoverAppointmentRepository) FindByID(id uint) (*model.HandoverAppointment, error) {
	var h model.HandoverAppointment
	if err := translate(r.db.First(&h, id).Error); err != nil {
		return nil, err
	}
	return &h, nil
}

// Update persists a handover appointment.
func (r *HandoverAppointmentRepository) Update(h *model.HandoverAppointment) error {
	return translate(r.db.Save(h).Error)
}

// UpdateStatusIf persists only when the current status still matches expect,
// returning whether a row was changed (used to serialize slot selection).
func (r *HandoverAppointmentRepository) UpdateStatusIf(h *model.HandoverAppointment, expect string) (bool, error) {
	res := r.db.Model(&model.HandoverAppointment{}).
		Where("id = ? AND status = ?", h.ID, expect).
		Updates(map[string]interface{}{
			"status":        h.Status,
			"selected_slot": h.SelectedSlot,
			"updated_at":    h.UpdatedAt,
		})
	if res.Error != nil {
		return false, translate(res.Error)
	}
	return res.RowsAffected == 1, nil
}

// ListByApplication returns all handover appointments of an application, newest first.
func (r *HandoverAppointmentRepository) ListByApplication(applicationID uint) ([]model.HandoverAppointment, error) {
	var items []model.HandoverAppointment
	if err := r.db.Where("application_id = ?", applicationID).
		Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListByUser returns all handover appointments of an adopter, newest first.
func (r *HandoverAppointmentRepository) ListByUser(userID uint) ([]model.HandoverAppointment, error) {
	var items []model.HandoverAppointment
	if err := r.db.Where("user_id = ?", userID).
		Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListByOrg returns all handover appointments targeting an org, newest first.
func (r *HandoverAppointmentRepository) ListByOrg(orgID uint) ([]model.HandoverAppointment, error) {
	var items []model.HandoverAppointment
	if err := r.db.Where("org_id = ?", orgID).
		Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindActiveByApplication returns the non-terminal handover appointment of an
// application, if any.
func (r *HandoverAppointmentRepository) FindActiveByApplication(applicationID uint) (*model.HandoverAppointment, error) {
	var h model.HandoverAppointment
	err := translate(r.db.
		Where("application_id = ? AND status IN ?", applicationID, []string{"offered", "pending"}).
		Order("id DESC").First(&h).Error)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// FindActiveBySlot reports whether the org's slot at t is occupied by another
// application (pending/confirmed appointments only).
func (r *HandoverAppointmentRepository) FindActiveBySlot(orgID uint, t time.Time, excludeID uint) (*model.HandoverAppointment, error) {
	var h model.HandoverAppointment
	err := translate(r.db.
		Where("org_id = ? AND selected_slot = ? AND id <> ? AND status IN ?",
			orgID, t, excludeID, []string{"pending", "confirmed"}).
		Order("id DESC").First(&h).Error)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// ExpireDue flips offered/pending appointments past their confirmation deadline
// to expired, releasing any selected slot.
func (r *HandoverAppointmentRepository) ExpireDue(now time.Time) (int64, error) {
	res := r.db.Model(&model.HandoverAppointment{}).
		Where("status IN ? AND confirm_before < ?", []string{"offered", "pending"}, now).
		Updates(map[string]interface{}{
			"status":        "expired",
			"selected_slot": nil,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}
