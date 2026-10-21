package leave

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/settings"
)

type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers { return &Handlers{svc: svc, log: log} }

// Register mounts the owner routes; the group must require an active account.
func (h *Handlers) Register(g gin.IRouter) {
	g.GET("/leave-context", h.context)
	g.POST("/leave-requests", h.create)
	g.GET("/leave-requests/drafts", h.drafts)
	g.POST("/leave-requests/preview", h.preview)
	g.GET("/leave-requests/:id", h.get)
	g.PATCH("/leave-requests/:id", h.save)
	g.DELETE("/leave-requests/:id", h.delete)
	g.GET("/leave-requests/:id/review", h.review)
	g.POST("/leave-requests/:id/submit", h.submit)
	g.POST("/leave-requests/:id/cancel", h.cancel)
	g.GET("/leave-requests/:id/history", h.history)
	g.GET("/leave-requests", h.list)
	g.GET("/leave-summary", h.summary)
}

func me(c *gin.Context) uint64 { return auth.CurrentPrincipal(c).EmployeeID }

func (h *Handlers) fail(c *gin.Context, err error) {
	var se *SubmitError
	if errors.As(err, &se) {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, gin.H{"error": gin.H{
			"code": "submit_blocked", "message": "Please fix the problems below before submitting.",
			"issues": se.Issues, "request_id": c.GetString("request_id"),
		}})
		return
	}
	httpx.Fail(c, h.log, err)
}

func (h *Handlers) context(c *gin.Context) {
	ctx, err := h.svc.Context(c.Request.Context(), me(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	pj := ctx.Policy.JSON()
	c.JSON(http.StatusOK, gin.H{
		"employee": ctx.Employee, "manager": ctx.Manager, "reasons": nonNil(ctx.Reasons),
		"policy": gin.H{
			"work_start": pj.WorkStart, "work_end": pj.WorkEnd, "lunch_start": pj.LunchStart, "lunch_end": pj.LunchEnd,
			"workdays": pj.Workdays, "minutes_per_day": pj.MinutesPerDay, "min_leave_minutes": pj.MinLeaveMinutes,
			"leave_unit_minutes": pj.LeaveUnitMinutes, "leave_types": pj.LeaveTypes, "timezone": pj.Timezone,
		},
	})
}

func nonNil(r []settings.Reason) []settings.Reason {
	if r == nil {
		return []settings.Reason{}
	}
	return r
}

func (h *Handlers) create(c *gin.Context) {
	r, err := h.svc.CreateDraft(c.Request.Context(), me(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusCreated, r)
}

func (h *Handlers) drafts(c *gin.Context) {
	items, err := h.svc.ListDrafts(c.Request.Context(), me(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (h *Handlers) get(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	r, err := h.svc.Get(c.Request.Context(), me(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handlers) save(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var in SaveInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	r, err := h.svc.SaveDraft(c.Request.Context(), me(c), id, in)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handlers) delete(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	v, err := strconv.ParseUint(c.Query("version"), 10, 32)
	if err != nil {
		httpx.Abort(c, http.StatusBadRequest, "invalid_request", "version query parameter is required")
		return
	}
	if err := h.svc.DeleteDraft(c.Request.Context(), me(c), id, uint32(v)); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

type previewInput struct {
	RequestID uint64 `json:"request_id"`
	Fields
}

func (h *Handlers) preview(c *gin.Context) {
	var in previewInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	ev, err := h.svc.Preview(c.Request.Context(), me(c), in.RequestID, in.Fields)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, ev)
}

func (h *Handlers) review(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	r, err := h.svc.Review(c.Request.Context(), me(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handlers) submit(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var in struct {
		Version uint32 `json:"version"`
	}
	if !httpx.BindJSON(c, &in) {
		return
	}
	r, err := h.svc.Submit(c.Request.Context(), me(c), id, in.Version, c.GetHeader("Idempotency-Key"))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handlers) cancel(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var in struct {
		Version uint32 `json:"version"`
	}
	if !httpx.BindJSON(c, &in) {
		return
	}
	r, err := h.svc.Cancel(c.Request.Context(), me(c), id, in.Version)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handlers) history(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	events, versions, err := h.svc.OwnHistory(c.Request.Context(), me(c), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"events": events, "versions": versions})
}

func (h *Handlers) list(c *gin.Context) {
	res, err := h.svc.List(c.Request.Context(), me(c), ParseListFilter(c), httpx.PageParam(c, ListPageSize, ListPageSize))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handlers) summary(c *gin.Context) {
	res, err := h.svc.Summary(c.Request.Context(), me(c))
	if err != nil {
		h.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
