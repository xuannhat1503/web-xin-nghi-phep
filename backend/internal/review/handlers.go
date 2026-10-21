package review

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/leave"
	"github.com/g-asiapacific/elm/backend/internal/settings"
)

type Handlers struct {
	svc      *Service
	settings *settings.Service
	log      *slog.Logger
}

func NewHandlers(svc *Service, st *settings.Service, log *slog.Logger) *Handlers {
	return &Handlers{svc: svc, settings: st, log: log}
}

// RegisterActive mounts reviewer routes (any active account; the data is scoped to the caller).
func (h *Handlers) RegisterActive(g gin.IRouter) {
	g.GET("/reviews", h.list)
	g.GET("/reviews/summary", h.summary)
	g.GET("/reviews/:id", h.get)
	g.POST("/reviews/:id/approve", h.decide("approve"))
	g.POST("/reviews/:id/reject", h.decide("reject"))
	g.POST("/reviews/:id/request-changes", h.decide("request_changes"))
}

func me(c *gin.Context) uint64 { return auth.CurrentPrincipal(c).EmployeeID }

func (h *Handlers) list(c *gin.Context) {
	res, err := h.svc.List(c.Request.Context(), me(c), leave.ParseListFilter(c), httpx.PageParam(c, leave.ListPageSize, leave.ListPageSize))
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handlers) overdueAfter(ctx context.Context) time.Duration {
	if p, err := h.settings.Current(ctx); err == nil && p.ReminderIntervalMinutes > 0 {
		return time.Duration(p.ReminderIntervalMinutes) * time.Minute
	}
	return 8 * time.Hour
}

func (h *Handlers) summary(c *gin.Context) {
	ctx := c.Request.Context()
	res, err := h.svc.Summary(ctx, me(c), h.overdueAfter(ctx))
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handlers) get(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	d, err := h.svc.Get(c.Request.Context(), me(c), id)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *Handlers) decide(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.IDParam(c, "id")
		if !ok {
			return
		}
		var in Decision
		if !httpx.BindJSON(c, &in) {
			return
		}
		if err := h.svc.Decide(c.Request.Context(), me(c), id, action, in); err != nil {
			httpx.Fail(c, h.log, err)
			return
		}
		h.get(c)
	}
}
