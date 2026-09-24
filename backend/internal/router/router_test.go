package router

import (
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/gbadopt/gbadopt/internal/config"
	"github.com/gbadopt/gbadopt/internal/util"
)

// TestSetupRegistersRoutes ensures the full route tree (static /me, /org mixed
// with parameterised /:id/... paths, including handover routes) builds without
// panicking. Repositories are constructed lazily, so a nil DB is sufficient for
// registration; no request is actually served.
func TestSetupRegistersRoutes(t *testing.T) {
	logger := util.NewLogger()
	cfg := config.Load()
	r := Setup(cfg, nil, nil, nil, logger)

	want := []struct {
		method string
		path   string
	}{
		{"POST", "/api/v1/applications"},
		{"GET", "/api/v1/applications/me"},
		{"GET", "/api/v1/applications/org"},
		{"PUT", "/api/v1/applications/:id/status"},
		{"POST", "/api/v1/applications/:id/handovers"},
		{"GET", "/api/v1/applications/:id/handovers"},
		{"GET", "/api/v1/handovers/me"},
		{"GET", "/api/v1/handovers/org"},
		{"POST", "/api/v1/handovers/:hid/select"},
		{"POST", "/api/v1/handovers/:hid/confirm"},
		{"POST", "/api/v1/handovers/:hid/cancel"},
	}
	for _, w := range want {
		if !routeRegistered(r, w.method, w.path) {
			t.Errorf("route not registered: %s %s", w.method, w.path)
		}
	}
}

func routeRegistered(r interface{ Routes() gin.RoutesInfo }, method, path string) bool {
	for _, ri := range r.Routes() {
		if ri.Method == method && ri.Path == path {
			return true
		}
	}
	return false
}
