package model

import "time"

// HandoverAppointment is an adopter's slot lock against a handover offer.
// Only one pending/confirmed appointment may occupy the same org + slot window,
// enforced by a partial unique index (see migrations).
type HandoverAppointment struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	OfferID       uint       `gorm:"index;not null" json:"offer_id"`
	ApplicationID uint       `gorm:"index;not null" json:"application_id"`
	OrgID         uint       `gorm:"index;not null" json:"org_id"`
	UserID        uint       `gorm:"index;not null" json:"user_id"`
	Location      string     `gorm:"size:255;not null" json:"location"`
	Deadline      time.Time  `gorm:"not null" json:"deadline"`
	StartAt       time.Time  `gorm:"index;not null" json:"start_at"`
	EndAt         time.Time  `gorm:"not null" json:"end_at"`
	Status        string     `gorm:"size:16;default:pending;index" json:"status"`
	CancelledBy   uint       `json:"cancelled_by"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
