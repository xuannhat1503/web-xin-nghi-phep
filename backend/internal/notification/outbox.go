package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/database"
)

const (
	maxAttempts  = 5
	batchSize    = 20
	leaseSeconds = 300
)

// Outbox renders queued emails. Every message is stored in mock_emails. Only when an Admin selects
// TEST_EMAIL (and SMTP is configured in .env) is it also sent - and then exclusively to the test inbox.
type Outbox struct {
	DB     *sql.DB
	Origin string // public app origin for links in emails
	SMTP   config.SMTP
	Live   bool          // EMAIL_LIVE_ENABLED: LIVE mode may send to the real recipients
	Access config.Access // ALLOWED_EMAILS / ALLOWED_EMAIL_DOMAIN: every LIVE recipient must pass
}

type task struct {
	id        uint64
	eventType string
	payload   struct {
		RequestID   uint64 `json:"request_id"`
		RecipientID uint64 `json:"recipient_id"`
	}
	attempts int
}

// dbtx is satisfied by *sql.DB and *sql.Tx.
type dbtx interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Process handles one batch: claim rows in a short transaction (SKIP LOCKED), then render and send
// outside any transaction, then record the outcome. A crashed run's claim expires after the lease.
func (o Outbox) Process(ctx context.Context) error {
	var tasks []task
	err := database.InTx(ctx, o.DB, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx,
			`SELECT id, event_type, payload, attempts FROM notification_outbox
			 WHERE attempts < ? AND (
			   (status IN ('PENDING', 'FAILED') AND next_retry_at <= UTC_TIMESTAMP(3))
			   OR (status = 'PROCESSING' AND locked_until < UTC_TIMESTAMP(3)))
			 ORDER BY id LIMIT ? FOR UPDATE SKIP LOCKED`, maxAttempts, batchSize)
		if err != nil {
			return err
		}
		for rows.Next() {
			var t task
			var raw []byte
			if err := rows.Scan(&t.id, &t.eventType, &raw, &t.attempts); err != nil {
				rows.Close()
				return err
			}
			_ = json.Unmarshal(raw, &t.payload)
			tasks = append(tasks, t)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		for _, t := range tasks {
			if _, err := tx.ExecContext(ctx,
				`UPDATE notification_outbox SET status = 'PROCESSING', locked_until = UTC_TIMESTAMP(3) + INTERVAL ? SECOND WHERE id = ?`,
				leaseSeconds, t.id); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil || len(tasks) == 0 {
		return err
	}
	mode, err := LoadEmailMode(ctx, o.DB)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if err := o.deliver(ctx, o.DB, t, mode); err != nil {
			attempts := t.attempts + 1
			backoff := time.Duration(1<<attempts) * time.Minute
			if _, uerr := o.DB.ExecContext(ctx,
				`UPDATE notification_outbox SET status = 'FAILED', attempts = ?, last_error = ?, locked_until = NULL,
				 next_retry_at = UTC_TIMESTAMP(3) + INTERVAL ? SECOND WHERE id = ?`,
				attempts, safeError(err), int(backoff.Seconds()), t.id); uerr != nil {
				return uerr
			}
			continue
		}
		if _, err := o.DB.ExecContext(ctx,
			`UPDATE notification_outbox SET status = 'SENT', attempts = attempts + 1, last_error = NULL, locked_until = NULL,
			 processed_at = UTC_TIMESTAMP(3) WHERE id = ?`, t.id); err != nil {
			return err
		}
	}
	return nil
}

// EmailMode is the Admin-selected delivery mode and test recipients.
type EmailMode struct {
	Name string // MOCK | TEST_EMAIL | LIVE
	To   string // test inbox (TEST_EMAIL only)
	CC   []string
}

func (m EmailMode) Test() bool { return m.Name == "TEST_EMAIL" }
func (m EmailMode) Live() bool { return m.Name == "LIVE" }

func splitList(s string) []string {
	var out []string
	for _, a := range strings.Split(s, ",") {
		if a = strings.TrimSpace(a); a != "" {
			out = append(out, a)
		}
	}
	return out
}

func LoadEmailMode(ctx context.Context, db dbtx) (EmailMode, error) {
	var mode, to, cc string
	if err := db.QueryRowContext(ctx, "SELECT email_mode, test_email_to, test_email_cc FROM company_settings WHERE id = 1").
		Scan(&mode, &to, &cc); err != nil {
		return EmailMode{}, err
	}
	return EmailMode{Name: mode, To: to, CC: splitList(cc)}, nil
}

// errRender errors carry only safe, generic text (stored and shown on the admin pages).
type errRender string

func (e errRender) Error() string { return string(e) }

func safeError(err error) string {
	var r errRender
	if errors.As(err, &r) {
		return string(r)
	}
	return "Rendering failed. See server logs."
}

func (o Outbox) deliver(ctx context.Context, db dbtx, t task, mode EmailMode) error {
	var (
		leaveType, startTime, endTime, reasonText          string
		reasonLabel, rejection, changes, reviewer, reasonC sql.NullString
		startDate, endDate                                 time.Time
		total                                              int
		snapshot                                           []byte
		employeeName                                       string
	)
	err := db.QueryRowContext(ctx,
		`SELECT COALESCE(lt.name_en, lr.leave_type), lr.start_date, lr.end_date, lr.start_time, lr.end_time,
		        lr.total_minutes, lr.reason_code, lr.reason_text, lr.rejection_reason, lr.change_request_reason,
		        COALESCE(lr.calculation_snapshot, JSON_OBJECT()), e.full_name, rv.full_name, COALESCE(NULLIF(rs.label_en, ''), rs.label)
		 FROM leave_requests lr
		 JOIN employees e ON e.id = lr.employee_id
		 LEFT JOIN employees rv ON rv.id = lr.reviewer_id
		 LEFT JOIN leave_reasons rs ON rs.code = lr.reason_code
		 LEFT JOIN leave_types lt ON lt.code = lr.leave_type
		 WHERE lr.id = ?`, t.payload.RequestID).
		Scan(&leaveType, &startDate, &endDate, &startTime, &endTime, &total, &reasonC, &reasonText,
			&rejection, &changes, &snapshot, &employeeName, &reviewer, &reasonLabel)
	if errors.Is(err, sql.ErrNoRows) {
		return errRender("Leave request no longer exists.")
	}
	if err != nil {
		return err
	}
	var recipientName, recipientEmail string
	if err := db.QueryRowContext(ctx, "SELECT full_name, email FROM employees WHERE id = ?", t.payload.RecipientID).
		Scan(&recipientName, &recipientEmail); err != nil {
		return errRender("Recipient not found.")
	}
	var snap struct {
		MinutesPerDay int `json:"minutes_per_day"`
	}
	_ = json.Unmarshal(snapshot, &snap)
	st, _ := civil.ParseClock(startTime)
	et, _ := civil.ParseClock(endTime)

	reason := strings.TrimSpace(reasonText)
	if reasonLabel.Valid && reasonLabel.String != "" {
		reason = strings.Trim(reasonLabel.String+" - "+reason, " -")
	}
	link := fmt.Sprintf("%s/leave/%d", o.Origin, t.payload.RequestID)
	if t.eventType == EmailApprovalRequired || t.eventType == EmailResubmitted || t.eventType == EmailCancelled || t.eventType == EmailReassigned {
		link = fmt.Sprintf("%s/leader/requests/%d", o.Origin, t.payload.RequestID)
	}
	var cc []string
	// Finance is copied only on approval, so it sees each leave once, with the final approved details.
	if t.eventType == EmailApproved {
		var list string
		_ = db.QueryRowContext(ctx, "SELECT finance_cc_emails FROM company_settings WHERE id = 1").Scan(&list)
		cc = splitList(list)
	}
	e := LeaveEmail{
		EmployeeName: employeeName, ManagerName: recipientName, ManagerEmail: recipientEmail, CC: cc,
		LeaveType: leaveType, StartDate: civil.DateOf(startDate), EndDate: civil.DateOf(endDate),
		StartTime: st, EndTime: et, TotalMinutes: total, MinutesPerDay: snap.MinutesPerDay, Reason: reason, ReviewLink: link,
	}
	var r Rendered
	switch t.eventType {
	case EmailApprovalRequired:
		r = ApprovalRequired(e)
	case EmailRejected:
		r = StatusEmail(t.eventType, recipientName, e, rejection.String)
	case EmailChanges:
		r = StatusEmail(t.eventType, recipientName, e, changes.String)
	case EmailApproved, EmailResubmitted, EmailCancelled, EmailReassigned:
		r = StatusEmail(t.eventType, recipientName, e, "")
	default:
		return errRender("Unknown email type.")
	}
	transport := "MOCK"
	if mode.Test() || mode.Live() {
		transport = mode.Name
	}
	// Stored first; UNIQUE(outbox_id) keeps retries idempotent.
	if _, err := db.ExecContext(ctx,
		`INSERT IGNORE INTO mock_emails (outbox_id, email_type, to_address, cc_addresses, subject, body_text, transport)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.id, t.eventType, r.To, strings.Join(r.CC, ","), r.Subject, r.Body, transport); err != nil {
		return err
	}
	switch {
	case mode.Test():
		return o.sendTest(ctx, db, t.id, r, mode)
	case mode.Live():
		return o.sendLive(ctx, db, t.id, r)
	}
	return nil
}

// sendLive delivers to the real recipients (manager / employee, Finance CC on approval). It needs
// EMAIL_LIVE_ENABLED=true on the server, and every address must pass ALLOWED_EMAILS / ALLOWED_EMAIL_DOMAIN.
// A message already delivered is never sent again.
func (o Outbox) sendLive(ctx context.Context, db dbtx, outboxID uint64, r Rendered) error {
	var delivered sql.NullTime
	if err := db.QueryRowContext(ctx, "SELECT delivered_at FROM mock_emails WHERE outbox_id = ?", outboxID).Scan(&delivered); err != nil {
		return err
	}
	if delivered.Valid {
		return nil
	}
	if !o.Live {
		return errRender("LIVE is selected but EMAIL_LIVE_ENABLED is not true in .env.")
	}
	if !o.SMTP.Configured() {
		return errRender("LIVE is selected but SMTP is not configured in .env.")
	}
	if strings.TrimSpace(r.To) == "" {
		return errRender("The recipient has no email address.")
	}
	for _, a := range append([]string{r.To}, r.CC...) {
		if !o.Access.Allows(a) {
			return errRender("A recipient is not allowed by ALLOWED_EMAILS / ALLOWED_EMAIL_DOMAIN: " + a)
		}
	}
	if err := smtpSend(ctx, o.SMTP, r.To, r.CC, r.Subject, r.Body); err != nil {
		slog.Warn("live email send failed", slog.Uint64("outbox_id", outboxID), slog.String("error", err.Error()))
		return errRender("SMTP send failed. Check the SMTP settings in .env and the server log.")
	}
	_, err := db.ExecContext(ctx, "UPDATE mock_emails SET delivered_to = ?, delivered_at = UTC_TIMESTAMP(3) WHERE outbox_id = ?",
		strings.Join(append([]string{r.To}, r.CC...), ","), outboxID)
	return err
}

// sendTest delivers a stored message to the test inbox only (never to the real recipients, who are listed in
// a footer). A message already delivered is never sent again.
func (o Outbox) sendTest(ctx context.Context, db dbtx, outboxID uint64, r Rendered, mode EmailMode) error {
	var delivered sql.NullTime
	if err := db.QueryRowContext(ctx, "SELECT delivered_at FROM mock_emails WHERE outbox_id = ?", outboxID).Scan(&delivered); err != nil {
		return err
	}
	if delivered.Valid {
		return nil
	}
	if !o.SMTP.Configured() {
		return errRender("TEST_EMAIL is selected but SMTP is not configured in .env.")
	}
	if err := CheckTestAllowlist(mode.To, mode.CC, nil, mode.To, mode.CC); err != nil {
		return errRender("Test recipients are not allowed (empty TO or company domain).")
	}
	// Same subject and body as LIVE; only a short footer says who the message was really addressed to.
	intended := "Originally addressed to: " + r.To
	if len(r.CC) > 0 {
		intended += " | CC: " + strings.Join(r.CC, ", ")
	}
	body := r.Body + "\n---\n" + intended + "\n"
	if err := smtpSend(ctx, o.SMTP, mode.To, mode.CC, r.Subject, body); err != nil {
		slog.Warn("test email send failed", slog.Uint64("outbox_id", outboxID), slog.String("error", err.Error()))
		return errRender("SMTP send failed. Check the SMTP settings in .env and the server log.")
	}
	_, err := db.ExecContext(ctx, "UPDATE mock_emails SET delivered_to = ?, delivered_at = UTC_TIMESTAMP(3) WHERE outbox_id = ?",
		strings.Join(append([]string{mode.To}, mode.CC...), ","), outboxID)
	return err
}

// Cleanup removes rendered emails older than 30 days (they contain leave reasons).
func (o Outbox) Cleanup(ctx context.Context) error {
	_, err := o.DB.ExecContext(ctx, "DELETE FROM mock_emails WHERE created_at < UTC_TIMESTAMP(3) - INTERVAL 30 DAY")
	return err
}
