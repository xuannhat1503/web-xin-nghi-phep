package settings

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers { return &Handlers{svc: svc, log: log} }

// RegisterAdmin mounts admin routes (group must require an active ADMIN).
func (h *Handlers) RegisterAdmin(admin gin.IRouter) {
	admin.GET("/settings", h.overview)
	admin.PUT("/settings/policy", h.updatePolicy)
	admin.PUT("/settings/finance-cc", h.updateFinance)
	admin.PUT("/settings/email", h.updateEmail)
	admin.POST("/settings/reasons", h.createReason)
	admin.PATCH("/settings/reasons/:id", h.updateReason)
	admin.DELETE("/settings/reasons/:id", h.deleteReason)
	admin.POST("/settings/leave-types", h.createLeaveType)
	admin.PATCH("/settings/leave-types/:code", h.updateLeaveType)
	admin.DELETE("/settings/leave-types/:code", h.deleteLeaveType)
}

// RegisterActive mounts read-only routes for any active employee (leave form needs reasons and rules).
func (h *Handlers) RegisterActive(api gin.IRouter) {
	api.GET("/leave-policy", h.publicPolicy)
	api.GET("/catalog", h.catalog)
}

func (h *Handlers) overview(c *gin.Context) {
	o, err := h.svc.Overview(c.Request.Context())
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *Handlers) updatePolicy(c *gin.Context) {
	var in PolicyInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	if _, err := h.svc.UpdatePolicy(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, in); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	h.overview(c)
}

func (h *Handlers) updateFinance(c *gin.Context) {
	var in FinanceInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	if err := h.svc.UpdateFinanceCC(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, in); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	h.overview(c)
}

func (h *Handlers) createReason(c *gin.Context) {
	var in ReasonInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	r, err := h.svc.CreateReason(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, r)
}

func (h *Handlers) updateReason(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var in ReasonInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	r, err := h.svc.UpdateReason(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, id, in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

// publicPolicy exposes only what the leave form needs: working hours, units, per-type rules and active reasons.
func (h *Handlers) publicPolicy(c *gin.Context) {
	ctx := c.Request.Context()
	p, err := h.svc.Current(ctx)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	reasons, err := h.svc.Reasons(ctx)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	active := make([]Reason, 0, len(reasons))
	for _, r := range reasons {
		if r.Active {
			active = append(active, r)
		}
	}
	pj := p.JSON()
	c.JSON(http.StatusOK, gin.H{
		"timezone": pj.Timezone, "work_start": pj.WorkStart, "work_end": pj.WorkEnd,
		"lunch_start": pj.LunchStart, "lunch_end": pj.LunchEnd, "workdays": pj.Workdays,
		"minutes_per_day": pj.MinutesPerDay, "min_leave_minutes": pj.MinLeaveMinutes,
		"leave_unit_minutes": pj.LeaveUnitMinutes, "leave_types": pj.LeaveTypes, "reasons": active,
	})
}

func (h *Handlers) updateEmail(c *gin.Context) {
	var in EmailInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	if err := h.svc.UpdateEmail(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, in); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	h.overview(c)
}

func (h *Handlers) deleteReason(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	deleted, err := h.svc.DeleteReason(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, id)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

func (h *Handlers) createLeaveType(c *gin.Context) {
	var in LeaveTypeInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	t, err := h.svc.CreateLeaveType(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, t)
}

func (h *Handlers) updateLeaveType(c *gin.Context) {
	var in LeaveTypeInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	t, err := h.svc.UpdateLeaveType(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, c.Param("code"), in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, t)
}

// deleteLeaveType hard-deletes an unused type; a type already used by requests is deactivated (deleted=false).
func (h *Handlers) deleteLeaveType(c *gin.Context) {
	deleted, err := h.svc.DeleteLeaveType(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, c.Param("code"))
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}

// catalog returns every leave type and reason (active or not) so pages can label old requests too.
func (h *Handlers) catalog(c *gin.Context) {
	cat, err := h.svc.Catalog(c.Request.Context())
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, cat)
}
