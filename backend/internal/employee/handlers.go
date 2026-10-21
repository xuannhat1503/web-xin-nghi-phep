package employee

import (
	"database/sql"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers { return &Handlers{svc: svc, log: log} }

// Register mounts the admin routes; the group must already require an active ADMIN.
func (h *Handlers) Register(admin gin.IRouter) {
	admin.GET("/employees", h.list)
	admin.GET("/employees/:id", h.get)
	admin.PATCH("/employees/:id", h.updateAccess)
	admin.PUT("/employees/:id/manager", h.assignManager)
	admin.GET("/employees/:id/manager-candidates", h.candidates)
}

func (h *Handlers) list(c *gin.Context) {
	res, err := h.svc.List(c.Request.Context(), ListFilter{
		Search: c.Query("q"), Status: c.Query("status"), NoManager: c.Query("no_manager") == "1",
	}, httpx.PageParam(c, PageSize, PageSize))
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
	d, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, d)
}

func (h *Handlers) updateAccess(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var body AccessChange
	if !httpx.BindJSON(c, &body) {
		return
	}
	if err := h.svc.UpdateAccess(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, id, body); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	h.get(c)
}

func (h *Handlers) assignManager(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var body ManagerChange
	if !httpx.BindJSON(c, &body) {
		return
	}
	if err := h.svc.AssignManager(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, id, body); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	h.get(c)
}

func (h *Handlers) candidates(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	items, err := h.svc.ManagerCandidates(c.Request.Context(), id, c.Query("q"))
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func nullStr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func actor(s sql.NullString) string {
	if !s.Valid {
		return "System"
	}
	return s.String
}

func sortHistory(h []HistoryEntry) {
	slices.SortStableFunc(h, func(a, b HistoryEntry) int { return b.CreatedAt.Compare(a.CreatedAt) })
}
