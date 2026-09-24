package dto

import "time"

// HandoverSlot is one bookable time window offered by an org.
type HandoverSlot struct {
	StartAt time.Time `json:"start_at" binding:"required"`
	EndAt   time.Time `json:"end_at" binding:"required"`
}

// HandoverOfferCreateRequest is the org's handover plan for an application.
type HandoverOfferCreateRequest struct {
	ApplicationID uint           `json:"application_id" binding:"required"`
	Location      string         `json:"location" binding:"required"`
	Deadline      time.Time      `json:"deadline" binding:"required"`
	Slots         []HandoverSlot `json:"slots" binding:"required,min=1,dive"`
}

// HandoverSlotView is a slot enriched with its current occupation state.
type HandoverSlotView struct {
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	Status        string    `json:"status"` // free / locked_by_me / occupied
	ApplicationID uint      `json:"application_id,omitempty"`
}

// HandoverAppointmentView is an appointment record.
type HandoverAppointmentView struct {
	ID            uint       `json:"id"`
	OfferID       uint       `json:"offer_id"`
	ApplicationID uint       `json:"application_id"`
	OrgID         uint       `json:"org_id"`
	UserID        uint       `json:"user_id"`
	Location      string     `json:"location"`
	Deadline      time.Time  `json:"deadline"`
	StartAt       time.Time  `json:"start_at"`
	EndAt         time.Time  `json:"end_at"`
	Status        string     `json:"status"`
	CancelledBy   uint       `json:"cancelled_by,omitempty"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// HandoverOfferView is an offer with slots and its latest appointment.
type HandoverOfferView struct {
	ID            uint                     `json:"id"`
	ApplicationID uint                     `json:"application_id"`
	OrgID         uint                     `json:"org_id"`
	Location      string                   `json:"location"`
	Deadline      time.Time                `json:"deadline"`
	Slots         []HandoverSlotView       `json:"slots"`
	Appointment   *HandoverAppointmentView `json:"appointment,omitempty"`
	CreatedAt     time.Time                `json:"created_at"`
}
