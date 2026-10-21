package notification

import (
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

// Admin exposes the outbox and rendered Mock Emails (group must require an active ADMIN).
type Admin struct {
	db  *sql.DB
	log *slog.Logger
}

func NewAdmin(db *sql.DB, log *slog.Logger) *Admin { return &Admin{db: db, log: log} }

func (h *Admin) Register(g gin.IRouter) {
	g.GET("/outbox", h.outbox)
	g.POST("/outbox/:id/retry", h.retry)
	g.GET("/mock-emails", h.emails)
	g.GET("/mock-emails/:id", h.email)
}

type OutboxItem struct {
	ID          uint64     `json:"id"`
	EventType   string     `json:"event_type"`
	Status      string     `json:"status"`
	Attempts    int        `json:"attempts"`
	NextRetryAt time.Time  `json:"next_retry_at"`
	LastError   *string    `json:"last_error"`
	RequestCode *string    `json:"request_code"`
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at"`
}

var outboxStatuses = map[string]bool{"PENDING": true, "PROCESSING": true, "SENT": true, "FAILED": true}

func (h *Admin) outbox(c *gin.Context) {
	ctx := c.Request.Context()
	p := httpx.PageParam(c, 20, 20)
	where, args := "1=1", []any{}
	if s := c.Query("status"); outboxStatuses[s] {
		where, args = "o.status = ?", append(args, s)
	}
	var total int64
	if err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM notification_outbox o WHERE "+where, args...).Scan(&total); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	rows, err := h.db.QueryContext(ctx,
		`SELECT o.id, o.event_type, o.status, o.attempts, o.next_retry_at, o.last_error, lr.request_code, o.created_at, o.processed_at
		 FROM notification_outbox o LEFT JOIN leave_requests lr ON lr.id = o.aggregate_id
		 WHERE `+where+` ORDER BY o.id DESC LIMIT ? OFFSET ?`, append(args, p.Limit(), p.Offset())...)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	defer rows.Close()
	var items []OutboxItem
	for rows.Next() {
		var it OutboxItem
		var lastErr, code sql.NullString
		var processed sql.NullTime
		if err := rows.Scan(&it.ID, &it.EventType, &it.Status, &it.Attempts, &it.NextRetryAt, &lastErr, &code, &it.CreatedAt, &processed); err != nil {
			httpx.Fail(c, h.log, err)
			return
		}
		if lastErr.Valid {
			it.LastError = &lastErr.String
		}
		if code.Valid {
			it.RequestCode = &code.String
		}
		if processed.Valid {
			it.ProcessedAt = &processed.Time
		}
		items = append(items, it)
	}
	counts, err := OutboxCounts(c, h.db)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"page": httpx.NewPaged(items, p, total), "counts": counts})
}

// OutboxCounts returns the number of outbox rows per status (also used by the system status page).
func OutboxCounts(c *gin.Context, db *sql.DB) (map[string]int64, error) {
	out := map[string]int64{"PENDING": 0, "PROCESSING": 0, "SENT": 0, "FAILED": 0}
	rows, err := db.QueryContext(c.Request.Context(), "SELECT status, COUNT(*) FROM notification_outbox GROUP BY status")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, rows.Err()
}

// retry re-queues a FAILED task (the worker picks it up on its next run).
func (h *Admin) retry(c *gin.Context) {
	id, ok := httpx.IDParam(c, "id")
	if !ok {
		return
	}
	res, err := h.db.ExecContext(c.Request.Context(),
		"UPDATE notification_outbox SET status = 'PENDING', attempts = 0, next_retry_at = UTC_TIMESTAMP(3), last_error = NULL WHERE id = ? AND status = 'FAILED'", id)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	if n, _ := res.RowsAffected(); n != 1 {
		httpx.Fail(c, h.log, httpx.ErrConflict)
		return
	}
	c.Status(http.StatusNoContent)
}

type EmailSummary struct {
	ID        uint64    `json:"id"`
	EmailType string    `json:"email_type"`
	To        string    `json:"to"`
	Subject   string    `json:"subject"`
	CreatedAt time.Time `json:"created_at"`
}

// emails lists metadata only; the body (which contains the leave reason) is shown one email at a time.
func (h *Admin) emails(c *gin.Context) {
	ctx := c.Request.Context()
	p := httpx.PageParam(c, 20, 20)
	var total int64
	if err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM mock_emails").Scan(&total); err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	rows, err := h.db.QueryContext(ctx,
		"SELECT id, email_type, to_address, subject, created_at FROM mock_emails ORDER BY id DESC LIMIT ? OFFSET ?", p.Limit(), p.Offset())
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	defer rows.Close()
	var items []EmailSummary
	for rows.Next() {
		var it EmailSummary
		if err := rows.Scan(&it.ID, &it.EmailType, &it.To, &it.Subject, &it.CreatedAt); err != nil {
			httpx.Fail(c, h.log, err)
			return
		}
		items = append(items, it)
	}
	c.JSON(http.StatusOK, httpx.NewPaged(items, p, total))
}

func (h *Admin) email(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		httpx.Fail(c, h.log, httpx.ErrNotFound)
		return
	}
	var it EmailSummary
	var cc, body, transport string
	err = h.db.QueryRowContext(c.Request.Context(),
		"SELECT id, email_type, to_address, cc_addresses, subject, body_text, transport, created_at FROM mock_emails WHERE id = ?", id).
		Scan(&it.ID, &it.EmailType, &it.To, &cc, &it.Subject, &body, &transport, &it.CreatedAt)
	if err == sql.ErrNoRows {
		httpx.Fail(c, h.log, httpx.ErrNotFound)
		return
	}
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	ccList := []string{}
	for _, a := range strings.Split(cc, ",") {
		if a = strings.TrimSpace(a); a != "" {
			ccList = append(ccList, a)
		}
	}
	c.JSON(http.StatusOK, gin.H{"email": it, "cc": ccList, "body": body, "transport": transport})
}
