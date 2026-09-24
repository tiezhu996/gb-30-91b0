package dto

import "time"

// ApplicationSubmitRequest submits an adoption application.
type ApplicationSubmitRequest struct {
	PetID         uint   `json:"pet_id" binding:"required"`
	Questionnaire string `json:"questionnaire"`
}

// ApplicationStatusRequest changes application status.
type ApplicationStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// ApplicationView is an application enriched with handover scheduling info.
type ApplicationView struct {
	ID            uint                      `json:"id"`
	UserID        uint                      `json:"user_id"`
	PetID         uint                      `json:"pet_id"`
	OrgID         uint                      `json:"org_id"`
	Questionnaire string                    `json:"questionnaire"`
	Status        string                    `json:"status"`
	CreatedAt     time.Time                 `json:"created_at"`
	UpdatedAt     time.Time                 `json:"updated_at"`
	Handover      *HandoverOfferView        `json:"handover,omitempty"`
	History       []HandoverAppointmentView `json:"history,omitempty"`
}
