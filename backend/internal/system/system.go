// Package system reports safe operational status for administrators: no secrets, raw SQL errors or stack traces.
package system

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/notification"
)

type Handlers struct {
	db  *sql.DB
	log *slog.Logger
}

func NewHandlers(db *sql.DB, log *slog.Logger) *Handlers { return &Handlers{db: db, log: log} }

func (h *Handlers) RegisterAdmin(g gin.IRouter) { g.GET("/system", h.status) }

type Worker struct {
	Name          string     `json:"name"`
	LastRunAt     *time.Time `json:"last_run_at"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	LastError     *string    `json:"last_error"`
}

type OpError struct {
	Component string    `json:"component"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func (h *Handlers) status(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()
	out := gin.H{"server_time": time.Now().UTC(), "database": "ok"}
	if err := h.db.PingContext(ctx); err != nil {
		h.log.Warn("system status: db ping failed", slog.String("error", err.Error()))
		out["database"] = "fail"
		c.JSON(http.StatusOK, out)
		return
	}
	v, dirty, err := database.SchemaVersion(ctx, h.db)
	latest, _ := database.LatestMigration()
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	out["migration"] = gin.H{"version": v, "latest": latest, "dirty": dirty, "up_to_date": !dirty && v == latest}

	rows, err := h.db.QueryContext(ctx, "SELECT name, last_run_at, last_success_at, last_error FROM worker_status ORDER BY name")
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	workers := []Worker{}
	for rows.Next() {
		var w Worker
		var run, ok sql.NullTime
		var e sql.NullString
		if err := rows.Scan(&w.Name, &run, &ok, &e); err != nil {
			rows.Close()
			httpx.Fail(c, h.log, err)
			return
		}
		if run.Valid {
			w.LastRunAt = &run.Time
		}
		if ok.Valid {
			w.LastSuccessAt = &ok.Time
		}
		if e.Valid {
			w.LastError = &e.String
		}
		workers = append(workers, w)
	}
	rows.Close()
	out["workers"] = workers

	counts, err := notification.OutboxCounts(c, h.db)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	out["outbox"] = counts

	rows, err = h.db.QueryContext(ctx, "SELECT component, code, message, created_at FROM operational_errors ORDER BY id DESC LIMIT 20")
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	defer rows.Close()
	errs := []OpError{}
	for rows.Next() {
		var e OpError
		if err := rows.Scan(&e.Component, &e.Code, &e.Message, &e.CreatedAt); err != nil {
			httpx.Fail(c, h.log, err)
			return
		}
		errs = append(errs, e)
	}
	out["recent_errors"] = errs
	if m, err := notification.LoadEmailMode(ctx, h.db); err == nil {
		out["email_mode"] = m.Name
	} else {
		out["email_mode"] = "OFF"
	}
	c.JSON(http.StatusOK, out)
}
