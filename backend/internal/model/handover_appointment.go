package model

import "time"

// HandoverAppointment arranges the in-person pet handover for an approved
// adoption application. An org offers a location, a confirmation deadline and
// bookable slots; the adopter picks one slot (occupying it for the whole org),
// then the org confirms. Both parties may cancel before confirmation, which
// releases the slot; missing the deadline without confirmation expires it.
// Confirmed/expired appointments are immutable, and a fresh record is created
// when arranging again, so old records are preserved.
type HandoverAppointment struct {
	ID            uint       `gorm:"primaryKey" json:"id"`
	ApplicationID uint       `gorm:"index;not null" json:"application_id"`
	OrgID         uint       `gorm:"index;not null" json:"org_id"`
	UserID        uint       `gorm:"index;not null" json:"user_id"`
	Location      string     `gorm:"size:255;not null" json:"location"`
	Slots         string     `gorm:"type:json;not null" json:"slots"`
	ConfirmBefore time.Time  `gorm:"not null" json:"confirm_before"`
	SelectedSlot  *time.Time `gorm:"index" json:"selected_slot"`
	Status        string     `gorm:"size:16;default:offered;index" json:"status"`
	CancelledBy   string     `gorm:"size:8" json:"cancelled_by"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
	CancelledAt   *time.Time `json:"cancelled_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}
