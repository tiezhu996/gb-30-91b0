package dto

import "time"

// HandoverOfferRequest is sent by the org to arrange a handover for an
// approved application.
type HandoverOfferRequest struct {
	ApplicationID uint     `json:"application_id" binding:"required"`
	Location      string   `json:"location" binding:"required"`
	Slots         []string `json:"slots" binding:"required,min=1"`
	ConfirmBefore string   `json:"confirm_before" binding:"required"`
}

// HandoverSelectRequest is sent by the adopter to occupy a offered slot.
type HandoverSelectRequest struct {
	Slot string `json:"slot" binding:"required"`
}

// HandoverSlotView describes one bookable slot and its current lock state.
type HandoverSlotView struct {
	Time                string `json:"time"`
	Locked              bool   `json:"locked"`
	LockedApplicationID uint   `json:"locked_application_id,omitempty"`
}

// HandoverAppointmentResponse is the handover view returned to clients.
type HandoverAppointmentResponse struct {
	ID                uint                `json:"id"`
	ApplicationID     uint                `json:"application_id"`
	OrgID             uint                `json:"org_id"`
	UserID            uint                `json:"user_id"`
	Location          string              `json:"location"`
	Slots             []HandoverSlotView  `json:"slots"`
	ConfirmBefore     time.Time           `json:"confirm_before"`
	SelectedSlot      *time.Time          `json:"selected_slot"`
	Status            string              `json:"status"`
	CancelledBy       string              `json:"cancelled_by"`
	ConfirmedAt       *time.Time          `json:"confirmed_at"`
	CancelledAt       *time.Time          `json:"cancelled_at"`
	CreatedAt         time.Time           `json:"created_at"`
	UpdatedAt         time.Time           `json:"updated_at"`
}
