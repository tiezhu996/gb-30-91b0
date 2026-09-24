package model

import "time"

// HandoverOffer is an org's handover plan for an approved application:
// location, confirmation deadline and the bookable time slots.
// Every re-schedule creates a new row so old offers are kept as history.
type HandoverOffer struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	ApplicationID uint      `gorm:"index;not null" json:"application_id"`
	OrgID         uint      `gorm:"index;not null" json:"org_id"`
	Location      string    `gorm:"size:255;not null" json:"location"`
	Deadline      time.Time `gorm:"not null" json:"deadline"`
	Slots         string    `gorm:"type:json;not null" json:"slots"` // [{"start_at","end_at"}]
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
