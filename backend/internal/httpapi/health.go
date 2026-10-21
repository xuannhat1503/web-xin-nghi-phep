package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Check is a named readiness probe. Its error is logged but never returned to the client.
type Check struct {
	Name string
	Fn   func(ctx context.Context) error
}

type Health struct {
	Checks []Check
	Logger *slog.Logger
}

// Live reports that the process is running.
func (h *Health) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Ready runs all checks with a short timeout and reports only pass/fail per check.
func (h *Health) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	status, code := "ok", http.StatusOK
	results := make(map[string]string, len(h.Checks))
	for _, chk := range h.Checks {
		if err := chk.Fn(ctx); err != nil {
			h.Logger.Warn("readiness check failed", slog.String("check", chk.Name), slog.String("error", err.Error()))
			results[chk.Name] = "fail"
			status, code = "unavailable", http.StatusServiceUnavailable
			continue
		}
		results[chk.Name] = "ok"
	}
	c.JSON(code, gin.H{"status": status, "checks": results})
}
