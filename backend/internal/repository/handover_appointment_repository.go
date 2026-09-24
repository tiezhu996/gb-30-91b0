package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/model"
)

// HandoverAppointmentRepository handles handover appointment persistence.
type HandoverAppointmentRepository struct{ db *gorm.DB }

// NewHandoverAppointmentRepository creates the repository.
func NewHandoverAppointmentRepository(db *gorm.DB) *HandoverAppointmentRepository {
	return &HandoverAppointmentRepository{db: db}
}

// Create inserts an appointment.
func (r *HandoverAppointmentRepository) Create(a *model.HandoverAppointment) error {
	return translate(r.db.Create(a).Error)
}

// CreateTx inserts an appointment within an outer transaction.
func (r *HandoverAppointmentRepository) CreateTx(tx *gorm.DB, a *model.HandoverAppointment) error {
	return translate(tx.Create(a).Error)
}

// FindByID locates an appointment by id.
func (r *HandoverAppointmentRepository) FindByID(id uint) (*model.HandoverAppointment, error) {
	var a model.HandoverAppointment
	if err := translate(r.db.First(&a, id).Error); err != nil {
		return nil, err
	}
	return &a, nil
}

// FindByIDTx locates an appointment by id within an outer transaction.
func (r *HandoverAppointmentRepository) FindByIDTx(tx *gorm.DB, id uint) (*model.HandoverAppointment, error) {
	var a model.HandoverAppointment
	if err := translate(tx.First(&a, id).Error); err != nil {
		return nil, err
	}
	return &a, nil
}

// Update persists an appointment.
func (r *HandoverAppointmentRepository) Update(a *model.HandoverAppointment) error {
	return translate(r.db.Save(a).Error)
}

// UpdateTx persists an appointment within an outer transaction.
func (r *HandoverAppointmentRepository) UpdateTx(tx *gorm.DB, a *model.HandoverAppointment) error {
	return translate(tx.Save(a).Error)
}

// FindActiveBySlot reports an active (slot-occupying) appointment for an org
// whose time window exactly matches the given slot.
func (r *HandoverAppointmentRepository) FindActiveBySlot(tx *gorm.DB, orgID uint, start, end time.Time) (*model.HandoverAppointment, error) {
	var a model.HandoverAppointment
	q := r.db
	if tx != nil {
		q = tx
	}
	err := translate(q.
		Where("org_id = ? AND start_at = ? AND end_at = ? AND status IN ?",
			orgID, start, end, constants.ActiveHandoverAppointmentStatuses()).
		Order("id DESC").
		First(&a).Error)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// FindActiveByApplication returns the active appointment of an application.
func (r *HandoverAppointmentRepository) FindActiveByApplication(applicationID uint) (*model.HandoverAppointment, error) {
	var a model.HandoverAppointment
	err := translate(r.db.
		Where("application_id = ? AND status IN ?", applicationID, constants.ActiveHandoverAppointmentStatuses()).
		Order("id DESC").
		First(&a).Error)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// ListByApplications returns all appointments of the given applications, newest first.
func (r *HandoverAppointmentRepository) ListByApplications(applicationIDs []uint) ([]model.HandoverAppointment, error) {
	var items []model.HandoverAppointment
	if len(applicationIDs) == 0 {
		return items, nil
	}
	if err := r.db.Where("application_id IN ?", applicationIDs).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// MarkExpired flips pending appointments past their confirmation deadline to expired.
func (r *HandoverAppointmentRepository) MarkExpired(now time.Time) (int64, error) {
	res := r.db.Model(&model.HandoverAppointment{}).
		Where("status = ? AND deadline < ?", constants.HandoverApptPending, now).
		Update("status", constants.HandoverApptExpired)
	return res.RowsAffected, res.Error
}
