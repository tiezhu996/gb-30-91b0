package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/gbadopt/gbadopt/internal/constants"
	"github.com/gbadopt/gbadopt/internal/dto"
	"github.com/gbadopt/gbadopt/internal/middleware"
	"github.com/gbadopt/gbadopt/internal/service"
	"github.com/gbadopt/gbadopt/internal/util"
)

// HandoverHandler exposes handover scheduling endpoints.
type HandoverHandler struct {
	svc    *service.HandoverService
	logger *slog.Logger
}

// NewHandoverHandler creates a HandoverHandler.
func NewHandoverHandler(svc *service.HandoverService, logger *slog.Logger) *HandoverHandler {
	return &HandoverHandler{svc: svc, logger: logger}
}

// CreateOffer handles POST /handovers/offers (org).
func (h *HandoverHandler) CreateOffer(c *gin.Context) {
	var req dto.HandoverOfferCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	offer, err := h.svc.CreateOffer(middleware.GetUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(offer))
}

// GetOffer handles GET /applications/:id/handover.
func (h *HandoverHandler) GetOffer(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid application id"))
		return
	}
	offer, err := h.svc.GetOffer(middleware.GetUserID(c), middleware.GetUserRole(c), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(offer))
}

// LockSlot handles POST /handovers/appointments (adopter).
func (h *HandoverHandler) LockSlot(c *gin.Context) {
	var req struct {
		OfferID uint             `json:"offer_id" binding:"required"`
		Slot    dto.HandoverSlot `json:"slot" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	a, err := h.svc.LockSlot(middleware.GetUserID(c), req.OfferID, req.Slot)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(a))
}

// Cancel handles PUT /handovers/appointments/:id/cancel.
func (h *HandoverHandler) Cancel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid appointment id"))
		return
	}
	a, err := h.svc.Cancel(middleware.GetUserID(c), middleware.GetUserRole(c), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(a))
}

// Confirm handles PUT /handovers/appointments/:id/confirm (org).
func (h *HandoverHandler) Confirm(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid appointment id"))
		return
	}
	a, err := h.svc.Confirm(middleware.GetUserID(c), uint(id))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(a))
}
