package holiday

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

type Handlers struct {
	admin  *Admin
	seeder Seeder
	log    *slog.Logger
}

func NewHandlers(admin *Admin, seeder Seeder, log *slog.Logger) *Handlers {
	return &Handlers{admin: admin, seeder: seeder, log: log}
}

// RegisterAdmin mounts calendar administration (group must require an active ADMIN).
func (h *Handlers) RegisterAdmin(g gin.IRouter) {
	g.GET("/calendar/years/:year", h.year)
	g.POST("/calendar/years/:year/seed", h.seed)
	g.POST("/calendar/years/:year/confirm", h.confirmYear)
	g.POST("/calendar/days", h.createDay)
	g.PUT("/calendar/days/:id", h.updateDay)
	g.POST("/calendar/days/:id/confirm", h.confirmDay)
	g.POST("/calendar/days/:id/unconfirm", h.unconfirmDay)
	g.DELETE("/calendar/days/:id", h.deleteDay)
}

// RegisterActive mounts the read-only, CONFIRMED-only calendar for active employees.
func (h *Handlers) RegisterActive(g gin.IRouter) {
	g.GET("/calendar", h.confirmed)
}

func yearParam(c *gin.Context, raw string) (int, bool) {
	y, err := strconv.Atoi(raw)
	if err != nil || y < MinYear || y > MaxYear {
		httpx.Abort(c, http.StatusNotFound, "not_found", "Resource not found")
		return 0, false
	}
	return y, true
}

func (h *Handlers) year(c *gin.Context) {
	y, ok := yearParam(c, c.Param("year"))
	if !ok {
		return
	}
	v, err := h.admin.Year(c.Request.Context(), y)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (h *Handlers) seed(c *gin.Context) {
	y, ok := yearParam(c, c.Param("year"))
	if !ok {
		return
	}
	rep, err := h.seeder.SeedYear(c.Request.Context(), y)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	h.log.Info("calendar seeded", "year", y, "inserted", len(rep.Inserted), "actor", auth.CurrentPrincipal(c).EmployeeID)
	c.JSON(http.StatusOK, gin.H{"year": y, "year_confirmed": rep.YearConfirmed, "inserted": len(rep.Inserted), "skipped": rep.Skipped})
}

func (h *Handlers) confirmYear(c *gin.Context) {
	y, ok := yearParam(c, c.Param("year"))
	if !ok {
		return
	}
	var in VersionInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	n, err := h.admin.ConfirmYear(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, y, in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"year": y, "days_confirmed": n})
}

func (h *Handlers) createDay(c *gin.Context) {
	var in DayInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	res, err := h.admin.CreateDay(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusCreated, res)
}

func (h *Handlers) updateDay(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var in DayInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	res, err := h.admin.UpdateDay(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, id, in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handlers) setStatus(c *gin.Context, confirm bool) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	var in VersionInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	res, err := h.admin.SetDayStatus(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, id, confirm, in)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handlers) confirmDay(c *gin.Context)   { h.setStatus(c, true) }
func (h *Handlers) unconfirmDay(c *gin.Context) { h.setStatus(c, false) }

func (h *Handlers) deleteDay(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	v, err := strconv.ParseUint(c.Query("version"), 10, 32)
	if err != nil {
		httpx.Abort(c, http.StatusBadRequest, "invalid_request", "version query parameter is required")
		return
	}
	res, err := h.admin.DeleteDay(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID, id, VersionInput{Version: uint32(v)})
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handlers) confirmed(c *gin.Context) {
	y, ok := yearParam(c, c.Query("year"))
	if !ok {
		return
	}
	yearConfirmed, days, err := h.admin.Confirmed(c.Request.Context(), y)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"year": y, "year_confirmed": yearConfirmed, "days": days})
}
