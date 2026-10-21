// Package review implements the reviewer (leader) side: assigned-request lists and decisions.
// Every query is scoped to the current reviewer.
package review

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/dbtypes"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/leave"
	"github.com/g-asiapacific/elm/backend/internal/notification"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

const maxComment = 1000

var (
	ErrNotPending      = httpx.E(http.StatusConflict, "not_pending", "This request is no longer pending.")
	ErrCommentRequired = httpx.E(http.StatusUnprocessableEntity, "comment_required", "Please enter a reason.")
	ErrCommentTooLong  = httpx.E(http.StatusUnprocessableEntity, "comment_too_long", "The reason must be at most 1000 characters.")
	ErrBadReviewer     = httpx.E(http.StatusUnprocessableEntity, "invalid_reviewer", "Choose an active employee other than the requester and the current reviewer.")
)

type Service struct {
	db  *sql.DB
	q   *store.Queries
	now func() time.Time
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db, q: store.New(db), now: func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }}
}

// reviewerScope: requests currently assigned to me, never my own, never drafts.
const reviewerScope = "lr.reviewer_id = ? AND lr.employee_id <> ? AND lr.status <> 'DRAFT'"

func (s *Service) List(ctx context.Context, me uint64, f leave.ListFilter, p httpx.Page) (httpx.Paged[leave.ListItem], error) {
	return leave.ListRequests(ctx, s.db, reviewerScope, []any{me, me}, f, p)
}

func (s *Service) Summary(ctx context.Context, me uint64, overdueAfter time.Duration) (map[string]any, error) {
	counts, err := leave.StatusCounts(ctx, s.db, reviewerScope, me, me)
	if err != nil {
		return nil, err
	}
	var overdue int64
	if err := s.db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM leave_requests lr WHERE "+reviewerScope+" AND lr.status = 'PENDING' AND lr.submitted_at < ?",
		me, me, s.now().Add(-overdueAfter)).Scan(&overdue); err != nil {
		return nil, err
	}
	pending, err := s.List(ctx, me, leave.ListFilter{Status: "PENDING", Sort: "oldest"}, httpx.Page{Number: 1, Size: 5})
	if err != nil {
		return nil, err
	}
	changes, err := s.List(ctx, me, leave.ListFilter{Status: "NEEDS_CHANGES", Sort: "newest"}, httpx.Page{Number: 1, Size: 5})
	if err != nil {
		return nil, err
	}
	return map[string]any{"counts": counts, "overdue": overdue, "pending": pending.Items, "needs_changes": changes.Items}, nil
}

type Detail struct {
	ID                  uint64                `json:"id"`
	Code                *string               `json:"request_code"`
	Status              string                `json:"status"`
	Employee            leave.Person          `json:"employee"`
	LeaveType           *string               `json:"leave_type"`
	StartDate           *string               `json:"start_date"`
	EndDate             *string               `json:"end_date"`
	StartTime           *string               `json:"start_time"`
	EndTime             *string               `json:"end_time"`
	TotalMinutes        uint32                `json:"total_minutes"`
	ReasonCode          *string               `json:"reason_code"`
	ReasonLabel         *string               `json:"reason_label"`
	ReasonText          string                `json:"reason_text"`
	RejectionReason     *string               `json:"rejection_reason"`
	ChangeRequestReason *string               `json:"change_request_reason"`
	Calculation         dbtypes.JSON          `json:"calculation"`
	CalendarChanged     bool                  `json:"calendar_changed"`
	SubmitCount         uint16                `json:"submit_count"`
	Version             uint32                `json:"version"`
	SubmittedAt         *time.Time            `json:"submitted_at"`
	ReviewedAt          *time.Time            `json:"reviewed_at"`
	CreatedAt           time.Time             `json:"created_at"`
	Events              []leave.TimelineEvent `json:"events"`
	Versions            []leave.Version       `json:"versions"`
}

func str(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func tm(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func dateStr(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	v := t.Time.Format(time.DateOnly)
	return &v
}

func clockStr(s sql.NullString) *string {
	if !s.Valid || len(s.String) < 5 {
		return nil
	}
	v := s.String[:5]
	return &v
}

// Get returns the request only while it is assigned to the viewer (a transferred-away reviewer gets 404).
func (s *Service) Get(ctx context.Context, me, id uint64) (Detail, error) {
	r, err := s.q.GetReviewRequest(ctx, store.GetReviewRequestParams{ID: id, ReviewerID: sql.NullInt64{Int64: int64(me), Valid: true}, ViewerID: me})
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, httpx.ErrNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	d := Detail{
		ID: r.ID, Code: str(r.RequestCode), Status: string(r.Status),
		Employee:  leave.Person{ID: r.EmployeeID, Name: r.EmployeeName, Email: r.EmployeeEmail},
		StartDate: dateStr(r.StartDate), EndDate: dateStr(r.EndDate), StartTime: clockStr(r.StartTime), EndTime: clockStr(r.EndTime),
		TotalMinutes: r.TotalMinutes, ReasonCode: str(r.ReasonCode), ReasonLabel: str(r.ReasonLabel), ReasonText: r.ReasonText,
		RejectionReason: str(r.RejectionReason), ChangeRequestReason: str(r.ChangeRequestReason),
		Calculation: r.CalculationSnapshot, CalendarChanged: r.CalendarChanged, SubmitCount: r.SubmitCount, Version: r.Version,
		SubmittedAt: tm(r.SubmittedAt), ReviewedAt: tm(r.ReviewedAt), CreatedAt: r.CreatedAt,
	}
	if r.LeaveType.Valid {
		t := r.LeaveType.String
		d.LeaveType = &t
	}
	if d.Events, d.Versions, err = leave.History(ctx, s.q, id); err != nil {
		return Detail{}, err
	}
	return d, nil
}

type Decision struct {
	Version uint32 `json:"version"`
	Comment string `json:"comment"`
}

// Decide applies approve / reject / request_changes. One decision per version: the conditional update
// (status, version, reviewer) lets exactly one concurrent decision succeed; the others get 409.
func (s *Service) Decide(ctx context.Context, me, id uint64, action string, in Decision) error {
	comment := strings.TrimSpace(in.Comment)
	var to store.LeaveRequestsStatus
	var event string
	switch action {
	case "approve":
		to, event = store.LeaveRequestsStatusAPPROVED, "APPROVED"
	case "reject":
		to, event = store.LeaveRequestsStatusREJECTED, "REJECTED"
	case "request_changes":
		to, event = store.LeaveRequestsStatusNEEDSCHANGES, "CHANGES_REQUESTED"
	default:
		return httpx.ErrNotFound
	}
	if to != store.LeaveRequestsStatusAPPROVED && comment == "" {
		return ErrCommentRequired
	}
	if utf8.RuneCountInString(comment) > maxComment {
		return ErrCommentTooLong
	}
	return database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.LockLeaveRequest(ctx, id)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && (!cur.ReviewerID.Valid || uint64(cur.ReviewerID.Int64) != me || cur.EmployeeID == me)) {
			return httpx.ErrNotFound // not (or no longer) assigned to me; never reveal more
		}
		if err != nil {
			return err
		}
		if cur.Status != store.LeaveRequestsStatusPENDING {
			return ErrNotPending
		}
		if cur.Version != in.Version {
			return httpx.ErrConflict
		}
		now := sql.NullTime{Time: s.now(), Valid: true}
		p := store.DecideLeaveRequestParams{Status: to, ID: id, Version: in.Version,
			ReviewerID: sql.NullInt64{Int64: int64(me), Valid: true}, NotEmployeeID: me}
		switch to {
		case store.LeaveRequestsStatusREJECTED:
			p.RejectionReason, p.ReviewedAt = sql.NullString{String: comment, Valid: true}, now
		case store.LeaveRequestsStatusNEEDSCHANGES:
			p.ChangeRequestReason = sql.NullString{String: comment, Valid: true}
		default:
			p.ReviewedAt = now
		}
		n, err := q.DecideLeaveRequest(ctx, p)
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		ev := store.InsertLeaveEventParams{
			LeaveRequestID: id, Action: event, ActorID: sql.NullInt64{Int64: int64(me), Valid: true},
			FromStatus: sql.NullString{String: "PENDING", Valid: true}, ToStatus: sql.NullString{String: string(to), Valid: true},
			RequestVersion: in.Version + 1, SubmitNo: sql.NullInt16{Int16: int16(cur.SubmitCount), Valid: true},
		}
		if comment != "" {
			ev.Comment = sql.NullString{String: comment, Valid: true}
		}
		if err := q.InsertLeaveEvent(ctx, ev); err != nil {
			return err
		}
		ref, err := leave.RefFor(ctx, q, id, in.Version+1)
		if err != nil {
			return err
		}
		return notification.Decided(ctx, tx, ref, string(to))
	})
}
