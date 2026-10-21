package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

// Execer is satisfied by *sql.DB and *sql.Tx; helpers run inside the caller's business transaction.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// In-app notification types. The UI translates them with "notif.<type>" and the params.
const (
	TypeNewRequest      = "NEW_REQUEST"
	TypeResubmitted     = "RESUBMITTED"
	TypeApproved        = "APPROVED"
	TypeRejected        = "REJECTED"
	TypeChanges         = "CHANGES_REQUESTED"
	TypeCancelled       = "CANCELLED"
	TypeAssignedToYou   = "REASSIGNED_TO_YOU"
	TypeAssignedAway    = "REASSIGNED_AWAY"
	TypeReviewerChanged = "REVIEWER_CHANGED"
	TypeReminder        = "REMINDER"
)

// Mock email types (outbox event_type).
const (
	EmailApprovalRequired = "APPROVAL_REQUIRED"
	EmailApproved         = "APPROVED"
	EmailRejected         = "REJECTED"
	EmailChanges          = "CHANGES_REQUESTED"
	EmailResubmitted      = "RESUBMITTED"
	EmailCancelled        = "CANCELLED"
	EmailReassigned       = "REVIEWER_REASSIGNED"
)

// Ref identifies the request and the people involved in an event.
type Ref struct {
	RequestID    uint64
	Code         string
	Version      uint32 // request version after the transition (makes dedup keys unique per event)
	SubmitNo     uint16
	EmployeeID   uint64
	EmployeeName string
	ReviewerID   uint64
	ReviewerName string
}

func reviewerLink(id uint64) string { return fmt.Sprintf("/leader/requests/%d", id) }
func ownerLink(id uint64) string    { return fmt.Sprintf("/leave/%d", id) }

// notify inserts one in-app notification; the dedup key makes retries harmless (INSERT IGNORE).
// body holds only the display params (request code and a person's name) as JSON.
func notify(ctx context.Context, x Execer, recipient uint64, typ string, r Ref, link, dedup string, params map[string]string) error {
	if recipient == 0 {
		return nil
	}
	if params == nil {
		params = map[string]string{}
	}
	params["code"] = r.Code
	body, _ := json.Marshal(params)
	_, err := x.ExecContext(ctx,
		`INSERT IGNORE INTO notifications (recipient_id, type, reference_id, title, body, link, dedup_key)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		recipient, typ, r.RequestID, typ+" "+r.Code, string(body), link, dedup)
	return err
}

// enqueue adds one Mock Email task to the outbox (INSERT IGNORE on the idempotency key).
func enqueue(ctx context.Context, x Execer, emailType string, r Ref, recipient uint64, key string) error {
	if recipient == 0 {
		return nil
	}
	payload, _ := json.Marshal(map[string]any{"request_id": r.RequestID, "recipient_id": recipient, "submit_no": r.SubmitNo})
	_, err := x.ExecContext(ctx,
		`INSERT IGNORE INTO notification_outbox (event_type, aggregate_id, payload, idempotency_key) VALUES (?, ?, ?, ?)`,
		emailType, r.RequestID, string(payload), key)
	return err
}

func key(kind string, r Ref, recipient uint64) string {
	return fmt.Sprintf("%s:%d:%d:%d", kind, r.RequestID, r.Version, recipient)
}

// Submitted notifies the reviewer of a new or resubmitted request.
func Submitted(ctx context.Context, x Execer, r Ref, resubmit bool) error {
	typ, email := TypeNewRequest, EmailApprovalRequired
	if resubmit {
		typ, email = TypeResubmitted, EmailResubmitted
	}
	if err := notify(ctx, x, r.ReviewerID, typ, r, reviewerLink(r.RequestID), key(typ, r, r.ReviewerID), map[string]string{"name": r.EmployeeName}); err != nil {
		return err
	}
	return enqueue(ctx, x, email, r, r.ReviewerID, key("mail:"+email, r, r.ReviewerID))
}

// Decided notifies the employee of an approval, rejection or change request.
func Decided(ctx context.Context, x Execer, r Ref, status string) error {
	typ, email := TypeApproved, EmailApproved
	switch status {
	case "REJECTED":
		typ, email = TypeRejected, EmailRejected
	case "NEEDS_CHANGES":
		typ, email = TypeChanges, EmailChanges
	}
	if err := notify(ctx, x, r.EmployeeID, typ, r, ownerLink(r.RequestID), key(typ, r, r.EmployeeID), map[string]string{"name": r.ReviewerName}); err != nil {
		return err
	}
	return enqueue(ctx, x, email, r, r.EmployeeID, key("mail:"+email, r, r.EmployeeID))
}

// Cancelled tells the current reviewer the employee withdrew the request.
func Cancelled(ctx context.Context, x Execer, r Ref) error {
	if err := notify(ctx, x, r.ReviewerID, TypeCancelled, r, reviewerLink(r.RequestID), key(TypeCancelled, r, r.ReviewerID), map[string]string{"name": r.EmployeeName}); err != nil {
		return err
	}
	return enqueue(ctx, x, EmailCancelled, r, r.ReviewerID, key("mail:"+EmailCancelled, r, r.ReviewerID))
}

// Reassigned informs the new reviewer (with email), the employee, and the previous reviewer (history only:
// the link goes to their notification list because they no longer have access to the request).
func Reassigned(ctx context.Context, x Execer, r Ref, oldReviewerID uint64) error {
	if err := notify(ctx, x, r.ReviewerID, TypeAssignedToYou, r, reviewerLink(r.RequestID), key(TypeAssignedToYou, r, r.ReviewerID), map[string]string{"name": r.EmployeeName}); err != nil {
		return err
	}
	if err := enqueue(ctx, x, EmailReassigned, r, r.ReviewerID, key("mail:"+EmailReassigned, r, r.ReviewerID)); err != nil {
		return err
	}
	if err := notify(ctx, x, r.EmployeeID, TypeReviewerChanged, r, ownerLink(r.RequestID), key(TypeReviewerChanged, r, r.EmployeeID), map[string]string{"name": r.ReviewerName}); err != nil {
		return err
	}
	return notify(ctx, x, oldReviewerID, TypeAssignedAway, r, "/notifications", key(TypeAssignedAway, r, oldReviewerID), map[string]string{"name": r.EmployeeName})
}

// Reminder reminds the current reviewer; level is the reminder number for this submission and reviewer.
func Reminder(ctx context.Context, x Execer, r Ref, level int) error {
	dedup := fmt.Sprintf("%s:%d:%d:%d:%d", TypeReminder, r.RequestID, r.SubmitNo, r.ReviewerID, level)
	return notify(ctx, x, r.ReviewerID, TypeReminder, r, reviewerLink(r.RequestID), dedup,
		map[string]string{"name": r.EmployeeName, "n": fmt.Sprint(level)})
}
