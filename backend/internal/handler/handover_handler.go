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

// HandoverHandler exposes handover appointment endpoints.
type HandoverHandler struct {
	svc    *service.HandoverService
	logger *slog.Logger
}

// NewHandoverHandler creates a HandoverHandler.
func NewHandoverHandler(svc *service.HandoverService, logger *slog.Logger) *HandoverHandler {
	return &HandoverHandler{svc: svc, logger: logger}
}

// Offer handles POST /applications/:id/handovers (org).
func (h *HandoverHandler) Offer(c *gin.Context) {
	appID, ok := parseApplicationID(c)
	if !ok {
		return
	}
	var req dto.HandoverOfferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam+": "+err.Error()))
		return
	}
	req.ApplicationID = appID
	created, err := h.svc.Offer(middleware.GetUserID(c), req)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, dto.OK(h.svc.ToResponse(created)))
}

// ListByApplication handles GET /applications/:id/handovers.
func (h *HandoverHandler) ListByApplication(c *gin.Context) {
	appID, ok := parseApplicationID(c)
	if !ok {
		return
	}
	items, err := h.svc.ListByApplication(middleware.GetUserID(c), appID, middleware.GetUserRole(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(h.svc.ToResponses(items)))
}

// ListMine handles GET /handovers/me (adopter).
func (h *HandoverHandler) ListMine(c *gin.Context) {
	items, err := h.svc.ListByUser(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(h.svc.ToResponses(items)))
}

// ListOrg handles GET /handovers/org.
func (h *HandoverHandler) ListOrg(c *gin.Context) {
	items, err := h.svc.ListByOrg(middleware.GetUserID(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(h.svc.ToResponses(items)))
}

// Select handles POST /handovers/:id/select (adopter).
func (h *HandoverHandler) Select(c *gin.Context) {
	id, ok := parseHandoverID(c)
	if !ok {
		return
	}
	var req dto.HandoverSelectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, constants.MsgInvalidParam))
		return
	}
	updated, err := h.svc.Select(middleware.GetUserID(c), id, req.Slot)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(h.svc.ToResponse(updated)))
}

// Confirm handles POST /handovers/:id/confirm (org).
func (h *HandoverHandler) Confirm(c *gin.Context) {
	id, ok := parseHandoverID(c)
	if !ok {
		return
	}
	updated, err := h.svc.Confirm(middleware.GetUserID(c), id)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(h.svc.ToResponse(updated)))
}

// Cancel handles POST /handovers/:id/cancel (both parties, before confirm).
func (h *HandoverHandler) Cancel(c *gin.Context) {
	id, ok := parseHandoverID(c)
	if !ok {
		return
	}
	updated, err := h.svc.Cancel(middleware.GetUserID(c), id, middleware.GetUserRole(c))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, dto.OK(h.svc.ToResponse(updated)))
}

func parseApplicationID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid application id"))
		return 0, false
	}
	return uint(id), true
}

func parseHandoverID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("hid"), 10, 64)
	if err != nil {
		c.Error(util.NewAppError(http.StatusBadRequest, constants.CodeBadRequest, "invalid handover id"))
		return 0, false
	}
	return uint(id), true
}
