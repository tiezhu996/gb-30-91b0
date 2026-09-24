package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbadopt/gbadopt/internal/config"
	"github.com/gbadopt/gbadopt/internal/handler"
	"github.com/gbadopt/gbadopt/internal/middleware"
)

func registerHandoverRoutes(v1 *gin.RouterGroup, cfg *config.Config, h *handler.HandoverHandler, limiter *middleware.RateLimiter) {
	// Handover history and offers are nested under an application.
	apps := v1.Group("/applications", middleware.AuthRequired(cfg))
	apps.GET("/:id/handovers", h.ListByApplication)
	apps.POST("/:id/handovers", middleware.RequireRole("org"), limiter.Limit(), h.Offer)

	// Actions on a specific appointment live under /handovers.
	ho := v1.Group("/handovers", middleware.AuthRequired(cfg))
	ho.GET("/me", h.ListMine)
	ho.GET("/org", middleware.RequireRole("org"), h.ListOrg)
	ho.POST("/:hid/select", middleware.RequireRole("user"), h.Select)
	ho.POST("/:hid/confirm", middleware.RequireRole("org"), h.Confirm)
	ho.POST("/:hid/cancel", h.Cancel)
}
