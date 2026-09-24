package repository

import (
	"gorm.io/gorm"

	"github.com/gbadopt/gbadopt/internal/model"
)

// HandoverOfferRepository handles handover offer persistence.
type HandoverOfferRepository struct{ db *gorm.DB }

// NewHandoverOfferRepository creates the repository.
func NewHandoverOfferRepository(db *gorm.DB) *HandoverOfferRepository {
	return &HandoverOfferRepository{db: db}
}

// Create inserts an offer.
func (r *HandoverOfferRepository) Create(o *model.HandoverOffer) error {
	return translate(r.db.Create(o).Error)
}

// CreateTx inserts an offer within an outer transaction.
func (r *HandoverOfferRepository) CreateTx(tx *gorm.DB, o *model.HandoverOffer) error {
	return translate(tx.Create(o).Error)
}

// FindByID locates an offer by id.
func (r *HandoverOfferRepository) FindByID(id uint) (*model.HandoverOffer, error) {
	var o model.HandoverOffer
	if err := translate(r.db.First(&o, id).Error); err != nil {
		return nil, err
	}
	return &o, nil
}

// FindLatestByApplication returns the newest offer of an application.
func (r *HandoverOfferRepository) FindLatestByApplication(applicationID uint) (*model.HandoverOffer, error) {
	var o model.HandoverOffer
	if err := translate(r.db.Where("application_id = ?", applicationID).Order("id DESC").First(&o).Error); err != nil {
		return nil, err
	}
	return &o, nil
}

// ListByApplications returns all offers of the given applications, newest first.
func (r *HandoverOfferRepository) ListByApplications(applicationIDs []uint) ([]model.HandoverOffer, error) {
	var items []model.HandoverOffer
	if len(applicationIDs) == 0 {
		return items, nil
	}
	if err := r.db.Where("application_id IN ?", applicationIDs).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
