package router

import (
	"github.com/gin-gonic/gin"

	"github.com/gbadopt/gbadopt/internal/config"
	"github.com/gbadopt/gbadopt/internal/handler"
	"github.com/gbadopt/gbadopt/internal/middleware"
)

func registerHandoverRoutes(v1 *gin.RouterGroup, cfg *config.Config, hh *handler.HandoverHandler, limiter *middleware.RateLimiter) {
	handovers := v1.Group("/handovers", middleware.AuthRequired(cfg))
	handovers.POST("/offers", middleware.RequireRole("org"), limiter.Limit(), hh.CreateOffer)
	handovers.POST("/appointments", middleware.RequireRole("user"), limiter.Limit(), hh.LockSlot)
	handovers.PUT("/appointments/:id/cancel", hh.Cancel)
	handovers.PUT("/appointments/:id/confirm", middleware.RequireRole("org"), hh.Confirm)
}
