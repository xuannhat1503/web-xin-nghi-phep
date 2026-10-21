package notification

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/auth"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

// Center is the per-user notification API. Every query is scoped by recipient_id = current user.
type Center struct {
	db  *sql.DB
	log *slog.Logger
}

func NewCenter(db *sql.DB, log *slog.Logger) *Center { return &Center{db: db, log: log} }

func (h *Center) Register(g gin.IRouter) {
	g.GET("/notifications", h.list)
	g.GET("/notifications/unread-count", h.unread)
	g.POST("/notifications/:id/read", h.read)
	g.POST("/notifications/read-all", h.readAll)
}

type Item struct {
	ID        uint64     `json:"id"`
	Type      string     `json:"type"`
	Params    string     `json:"params"` // JSON object with display params
	Link      string     `json:"link"`
	ReadAt    *time.Time `json:"read_at"`
	CreatedAt time.Time  `json:"created_at"`
}

const pageSize = 20

func (h *Center) list(c *gin.Context) {
	me := auth.CurrentPrincipal(c).EmployeeID
	p := httpx.PageParam(c, pageSize, pageSize)
	unreadOnly := c.Query("unread") == "1"
	where := "recipient_id = ?"
	if unreadOnly {
		where += " AND read_at IS NULL"
	}
	ctx := c.Request.Context()
	var total int64
	if err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications WHERE "+where, me).Scan(&total); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	rows, err := h.db.QueryContext(ctx,
		"SELECT id, type, body, link, read_at, created_at FROM notifications WHERE "+where+
			" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?", me, p.Limit(), p.Offset())
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		var it Item
		var read sql.NullTime
		if err := rows.Scan(&it.ID, &it.Type, &it.Params, &it.Link, &read, &it.CreatedAt); err != nil {
			httpx.Fail(c, h.log, err)
			return
		}
		if read.Valid {
			it.ReadAt = &read.Time
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, httpx.NewPaged(items, p, total))
}

// UnreadCount is also used by the layout on every page load.
func (h *Center) UnreadCount(ctx context.Context, me uint64) (int64, error) {
	var n int64
	err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notifications WHERE recipient_id = ? AND read_at IS NULL", me).Scan(&n)
	return n, err
}

func (h *Center) unread(c *gin.Context) {
	n, err := h.UnreadCount(c.Request.Context(), auth.CurrentPrincipal(c).EmployeeID)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"unread": n})
}

func (h *Center) read(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, h.log, httpx.ErrNotFound)
		return
	}
	if _, err := h.db.ExecContext(c.Request.Context(),
		"UPDATE notifications SET read_at = UTC_TIMESTAMP(3) WHERE id = ? AND recipient_id = ? AND read_at IS NULL",
		id, auth.CurrentPrincipal(c).EmployeeID); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Center) readAll(c *gin.Context) {
	if _, err := h.db.ExecContext(c.Request.Context(),
		"UPDATE notifications SET read_at = UTC_TIMESTAMP(3) WHERE recipient_id = ? AND read_at IS NULL",
		auth.CurrentPrincipal(c).EmployeeID); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.Status(http.StatusNoContent)
}
