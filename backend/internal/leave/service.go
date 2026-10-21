package leave

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/dbtypes"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/notification"
	"github.com/g-asiapacific/elm/backend/internal/settings"
	"github.com/g-asiapacific/elm/backend/internal/store"
	"github.com/g-asiapacific/elm/backend/internal/workcal"
)

const (
	maxOpenDrafts     = 20
	draftEventEvery   = 10 * time.Minute
	idempotencyScope  = "leave.submit"
	idempotencyTTL    = 24 * time.Hour
	statusDraft       = "DRAFT"
	statusNeedChanges = "NEEDS_CHANGES"
)

var (
	ErrTooManyDrafts = httpx.E(http.StatusConflict, "too_many_drafts", fmt.Sprintf("You already have %d drafts. Submit or delete some first.", maxOpenDrafts))
	ErrNotEditable   = httpx.E(http.StatusConflict, "not_editable", "This request can no longer be edited.")
	ErrIdempotency   = httpx.E(http.StatusUnprocessableEntity, "idempotency_mismatch", "This submission key was already used for a different request.")
)

type Service struct {
	db       *sql.DB
	q        *store.Queries
	settings *settings.Service
	origin   string
	now      func() time.Time
}

func NewService(db *sql.DB, st *settings.Service, appOrigin string) *Service {
	return &Service{db: db, q: store.New(db), settings: st, origin: appOrigin,
		now: func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) }}
}

type Person struct {
	ID     uint64 `json:"id,omitempty"`
	Name   string `json:"full_name"`
	Email  string `json:"email"`
	Active bool   `json:"active,omitempty"`
}

// Request is the owner's view of a leave request.
type Request struct {
	ID                  uint64       `json:"id"`
	Code                *string      `json:"request_code"`
	Status              string       `json:"status"`
	LeaveType           *string      `json:"leave_type"`
	StartDate           *string      `json:"start_date"`
	EndDate             *string      `json:"end_date"`
	StartTime           *string      `json:"start_time"`
	EndTime             *string      `json:"end_time"`
	TotalMinutes        uint32       `json:"total_minutes"`
	ReasonCode          *string      `json:"reason_code"`
	ReasonLabel         *string      `json:"reason_label"`
	ReasonText          string       `json:"reason_text"`
	Reviewer            *Person      `json:"reviewer"`
	RejectionReason     *string      `json:"rejection_reason"`
	ChangeRequestReason *string      `json:"change_request_reason"`
	Calculation         dbtypes.JSON `json:"calculation"`
	CalendarChanged     bool         `json:"calendar_changed"`
	SubmitCount         uint16       `json:"submit_count"`
	Version             uint32       `json:"version"`
	Editable            bool         `json:"editable"`
	CreatedAt           time.Time    `json:"created_at"`
	FirstSubmittedAt    *time.Time   `json:"first_submitted_at"`
	SubmittedAt         *time.Time   `json:"submitted_at"`
	ReviewedAt          *time.Time   `json:"reviewed_at"`
	UpdatedAt           time.Time    `json:"updated_at"`
}

func toRequest(r store.GetOwnLeaveRequestRow) Request {
	out := Request{
		ID: r.ID, Code: nstr(r.RequestCode), Status: string(r.Status), TotalMinutes: r.TotalMinutes,
		ReasonCode: nstr(r.ReasonCode), ReasonLabel: nstr(r.ReasonLabel), ReasonText: r.ReasonText,
		RejectionReason: nstr(r.RejectionReason), ChangeRequestReason: nstr(r.ChangeRequestReason),
		Calculation: r.CalculationSnapshot, CalendarChanged: r.CalendarChanged, SubmitCount: r.SubmitCount,
		Version: r.Version, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		FirstSubmittedAt: ntime(r.FirstSubmittedAt), SubmittedAt: ntime(r.SubmittedAt), ReviewedAt: ntime(r.ReviewedAt),
		StartDate: ndate(r.StartDate), EndDate: ndate(r.EndDate), StartTime: nclock(r.StartTime), EndTime: nclock(r.EndTime),
		Editable: r.Status == store.LeaveRequestsStatusDRAFT || r.Status == store.LeaveRequestsStatusNEEDSCHANGES,
	}
	if r.LeaveType.Valid {
		t := r.LeaveType.String
		out.LeaveType = &t
	}
	if r.ManagerNameSnapshot.Valid {
		out.Reviewer = &Person{Name: r.ManagerNameSnapshot.String, Email: r.ManagerEmailSnapshot.String}
	}
	return out
}

func (r Request) fields() Fields {
	return Fields{LeaveType: r.LeaveType, StartDate: r.StartDate, EndDate: r.EndDate, StartTime: r.StartTime,
		EndTime: r.EndTime, ReasonCode: r.ReasonCode, ReasonText: r.ReasonText}
}

// Context is what the leave form needs besides the request itself.
type Context struct {
	Employee Person            `json:"employee"`
	Manager  *Person           `json:"manager"`
	Policy   settings.Policy   `json:"-"`
	Reasons  []settings.Reason `json:"reasons"`
}

func (s *Service) Context(ctx context.Context, employeeID uint64) (Context, error) {
	emp, err := s.q.GetEmployeeByID(ctx, employeeID)
	if err != nil {
		return Context{}, err
	}
	out := Context{Employee: Person{ID: emp.ID, Name: emp.FullName, Email: emp.Email}}
	if emp.ManagerID.Valid {
		m, err := s.q.GetManagerInfo(ctx, uint64(emp.ManagerID.Int64))
		if err != nil {
			return Context{}, err
		}
		out.Manager = &Person{ID: m.ID, Name: m.FullName, Email: m.Email, Active: m.Active}
	}
	if out.Policy, err = s.settings.Current(ctx); err != nil {
		return Context{}, err
	}
	all, err := s.settings.Reasons(ctx)
	if err != nil {
		return Context{}, err
	}
	for _, r := range all {
		if r.Active {
			out.Reasons = append(out.Reasons, r)
		}
	}
	return out, nil
}

func (s *Service) CreateDraft(ctx context.Context, employeeID uint64) (Request, error) {
	var id int64
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		if _, err := q.LockEmployeeForSubmit(ctx, employeeID); err != nil {
			return err
		}
		n, err := q.CountOpenDrafts(ctx, employeeID)
		if err != nil {
			return err
		}
		if n >= maxOpenDrafts {
			return ErrTooManyDrafts
		}
		if id, err = q.CreateLeaveDraft(ctx, employeeID); err != nil {
			return err
		}
		return q.InsertLeaveEvent(ctx, store.InsertLeaveEventParams{
			LeaveRequestID: uint64(id), Action: "CREATED", ActorID: actor(employeeID),
			ToStatus: nullStr(statusDraft), RequestVersion: 1,
		})
	})
	if err != nil {
		return Request{}, err
	}
	return s.Get(ctx, employeeID, uint64(id))
}

type DraftSummary struct {
	ID           uint64    `json:"id"`
	LeaveType    *string   `json:"leave_type"`
	StartDate    *string   `json:"start_date"`
	EndDate      *string   `json:"end_date"`
	TotalMinutes uint32    `json:"total_minutes"`
	Version      uint32    `json:"version"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (s *Service) ListDrafts(ctx context.Context, employeeID uint64) ([]DraftSummary, error) {
	rows, err := s.q.ListLeaveDrafts(ctx, employeeID)
	if err != nil {
		return nil, err
	}
	out := make([]DraftSummary, 0, len(rows))
	for _, r := range rows {
		d := DraftSummary{ID: r.ID, StartDate: ndate(r.StartDate), EndDate: ndate(r.EndDate), TotalMinutes: r.TotalMinutes, Version: r.Version, UpdatedAt: r.UpdatedAt}
		if r.LeaveType.Valid {
			t := r.LeaveType.String
			d.LeaveType = &t
		}
		out = append(out, d)
	}
	return out, nil
}

// Get returns a request only to its owner; any other id is "not found" (no existence leak).
func (s *Service) Get(ctx context.Context, employeeID, id uint64) (Request, error) {
	r, err := s.q.GetOwnLeaveRequest(ctx, store.GetOwnLeaveRequestParams{ID: id, EmployeeID: employeeID})
	if errors.Is(err, sql.ErrNoRows) {
		return Request{}, httpx.ErrNotFound
	}
	if err != nil {
		return Request{}, err
	}
	return toRequest(r), nil
}

type SaveInput struct {
	Version uint32 `json:"version"`
	Fields
}

// SaveDraft is the auto-save endpoint. It never changes the status and never submits.
func (s *Service) SaveDraft(ctx context.Context, employeeID, id uint64, in SaveInput) (Request, error) {
	pol, err := s.settings.Current(ctx)
	if err != nil {
		return Request{}, err
	}
	if issues := lenientCheck(in.Fields, pol.LeaveTypes); len(issues) > 0 {
		return Request{}, validation(issues)
	}
	// Store an estimate so the drafts list can show it; the official value is recomputed at submit.
	var estimate uint32
	if ev, err := s.evaluateFields(ctx, s.q, employeeID, id, in.Fields, false); err == nil && ev.Calculation != nil {
		estimate = uint32(ev.Calculation.TotalMinutes)
	}
	err = database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.LockOwnLeaveRequest(ctx, store.LockOwnLeaveRequestParams{ID: id, EmployeeID: employeeID})
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.Status != store.LeaveRequestsStatusDRAFT && cur.Status != store.LeaveRequestsStatusNEEDSCHANGES {
			return ErrNotEditable
		}
		if cur.Version != in.Version {
			return httpx.ErrConflict
		}
		params := store.UpdateLeaveDraftParams{
			StartDate: nullDate(in.StartDate), EndDate: nullDate(in.EndDate),
			StartTime: nullClock(in.StartTime), EndTime: nullClock(in.EndTime),
			ReasonCode: nullStr(str(in.ReasonCode)), ReasonText: strings.TrimSpace(in.ReasonText),
			TotalMinutes: estimate, ID: id, EmployeeID: employeeID, Version: in.Version,
		}
		if t := str(in.LeaveType); t != "" {
			params.LeaveType = sql.NullString{String: t, Valid: true}
		}
		n, err := q.UpdateLeaveDraft(ctx, params)
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		// Coalesced audit: at most one DRAFT_SAVED event per draft every few minutes, never with the reason text.
		last, err := q.LastDraftSavedEventAt(ctx, id)
		if err != nil {
			return err
		}
		if t, ok := last.(time.Time); !ok || s.now().Sub(t) >= draftEventEvery {
			return q.InsertLeaveEvent(ctx, store.InsertLeaveEventParams{
				LeaveRequestID: id, Action: "DRAFT_SAVED", ActorID: actor(employeeID), RequestVersion: in.Version + 1,
			})
		}
		return nil
	})
	if err != nil {
		return Request{}, err
	}
	return s.Get(ctx, employeeID, id)
}

func (s *Service) DeleteDraft(ctx context.Context, employeeID, id uint64, version uint32) error {
	n, err := s.q.DeleteLeaveDraft(ctx, store.DeleteLeaveDraftParams{ID: id, EmployeeID: employeeID, Version: version})
	if err != nil {
		return err
	}
	if n == 1 {
		return nil
	}
	if _, err := s.Get(ctx, employeeID, id); err != nil {
		return err // not found
	}
	return httpx.ErrConflict // changed, submitted before, or no longer a draft
}

// Evaluation is the live result shown while editing and on the review page.
type Evaluation struct {
	Calculation   *workcal.Result `json:"calculation"`
	MinutesPerDay int             `json:"minutes_per_day"`
	Issues        []Issue         `json:"issues"`
	CanSubmit     bool            `json:"can_submit"`

	parsed parsed
	policy settings.Policy
}

// Preview evaluates unsaved form values (excludeID = the draft being edited, for the overlap check).
func (s *Service) Preview(ctx context.Context, employeeID, excludeID uint64, f Fields) (Evaluation, error) {
	return s.evaluateFields(ctx, s.q, employeeID, excludeID, f, true)
}

func (s *Service) evaluateFields(ctx context.Context, q *store.Queries, employeeID, excludeID uint64, f Fields, full bool) (Evaluation, error) {
	pol, err := settings.CurrentWith(ctx, q)
	if err != nil {
		return Evaluation{}, err
	}
	reasons, err := s.settings.Reasons(ctx)
	if err != nil {
		return Evaluation{}, err
	}
	loc, err := time.LoadLocation(pol.Timezone)
	if err != nil {
		loc = time.UTC
	}
	in := checkInput{fields: f, policy: pol, reasons: reasons, today: civil.DateOf(s.now().In(loc))}

	// Load the CONFIRMED calendar for the requested range (only when the dates parse).
	if pr, _ := parseRange(f, pol.LeaveTypes, pol.LeaveUnitMinutes); pr.haveRange {
		if in.rules, in.unconfirmed, err = loadCalendar(ctx, q, pr.interval.StartDate, pr.interval.EndDate); err != nil {
			return Evaluation{}, err
		}
	}
	p, calc, issues := evaluate(in)
	ev := Evaluation{Calculation: calc, MinutesPerDay: pol.MinutesPerDay, Issues: issues, parsed: p, policy: pol}

	if full {
		emp, err := q.GetEmployeeByID(ctx, employeeID)
		if err != nil {
			return Evaluation{}, err
		}
		ev.Issues = append(ev.Issues, managerIssues(ctx, q, emp)...)
		if p.haveRange {
			if is, err := overlapIssue(ctx, q, employeeID, excludeID, p.interval); err != nil {
				return Evaluation{}, err
			} else if is != nil {
				ev.Issues = append(ev.Issues, *is)
			}
		}
	}
	if ev.Issues == nil {
		ev.Issues = []Issue{}
	}
	ev.CanSubmit = len(ev.Issues) == 0 && calc != nil
	return ev, nil
}

func loadCalendar(ctx context.Context, q *store.Queries, from, to civil.Date) (map[civil.Date]workcal.Rule, []int, error) {
	rows, err := q.ListConfirmedDays(ctx, store.ListConfirmedDaysParams{FromDate: from.UTC(), ToDate: to.UTC()})
	if err != nil {
		return nil, nil, err
	}
	rules := make(map[civil.Date]workcal.Rule, len(rows))
	for _, r := range rows {
		rule := workcal.Rule{Date: civil.DateOf(r.DayDate), Kind: string(r.Kind), Name: r.Name}
		if r.StartTime.Valid && r.EndTime.Valid {
			s, err1 := civil.ParseClock(r.StartTime.String)
			e, err2 := civil.ParseClock(r.EndTime.String)
			if err1 == nil && err2 == nil {
				rule.Start, rule.End = &s, &e
			}
		}
		rules[rule.Date] = rule
	}
	var unconfirmed []int
	for y := from.Year; y <= to.Year; y++ {
		st, err := q.GetCalendarYearStatus(ctx, uint16(y))
		if errors.Is(err, sql.ErrNoRows) || (err == nil && st != store.CalendarYearsStatusCONFIRMED) {
			unconfirmed = append(unconfirmed, y)
			continue
		}
		if err != nil {
			return nil, nil, err
		}
	}
	return rules, unconfirmed, nil
}

func managerIssues(ctx context.Context, q *store.Queries, emp store.Employee) []Issue {
	if !emp.ManagerID.Valid {
		return []Issue{issue("no_manager", "", "You do not have a direct manager yet. Ask an administrator to assign one.", nil)}
	}
	if uint64(emp.ManagerID.Int64) == emp.ID {
		return []Issue{issue("no_manager", "", "You do not have a valid direct manager. Ask an administrator.", nil)}
	}
	m, err := q.GetManagerInfo(ctx, uint64(emp.ManagerID.Int64))
	if err != nil || !m.Active {
		return []Issue{issue("manager_inactive", "", "Your direct manager's account is inactive. Ask an administrator to assign an active manager.", nil)}
	}
	return nil
}

func wallTime(d civil.Date, c civil.Clock) time.Time {
	return d.UTC().Add(time.Duration(c) * time.Minute)
}

func overlapIssue(ctx context.Context, q *store.Queries, employeeID, excludeID uint64, iv workcal.Interval) (*Issue, error) {
	r, err := q.FindOverlappingRequest(ctx, store.FindOverlappingRequestParams{
		EmployeeID: employeeID, ExcludeID: excludeID,
		RangeEnd:   sql.NullTime{Time: wallTime(iv.EndDate, iv.EndTime), Valid: true},
		RangeStart: sql.NullTime{Time: wallTime(iv.StartDate, iv.StartTime), Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	code := r.RequestCode.String
	is := issue("overlap", "start_date", fmt.Sprintf("This time overlaps your request %s (%s).", code, r.Status),
		map[string]any{"code": code, "status": string(r.Status)})
	return &is, nil
}

// Review is everything shown on the Review & Confirm step.
type Review struct {
	Request    Request                `json:"request"`
	Evaluation Evaluation             `json:"evaluation"`
	Manager    *Person                `json:"manager"`
	Email      *notification.Rendered `json:"email_preview"`
}

func (s *Service) Review(ctx context.Context, employeeID, id uint64) (Review, error) {
	req, err := s.Get(ctx, employeeID, id)
	if err != nil {
		return Review{}, err
	}
	if !req.Editable {
		return Review{}, ErrNotEditable
	}
	ev, err := s.evaluateFields(ctx, s.q, employeeID, id, req.fields(), true)
	if err != nil {
		return Review{}, err
	}
	out := Review{Request: req, Evaluation: ev}
	c, err := s.Context(ctx, employeeID)
	if err != nil {
		return Review{}, err
	}
	out.Manager = c.Manager
	if ev.Calculation != nil && c.Manager != nil {
		email, err := s.renderApprovalEmail(ctx, c, req, ev)
		if err != nil {
			return Review{}, err
		}
		out.Email = &email
	}
	return out, nil
}

func (s *Service) renderApprovalEmail(ctx context.Context, c Context, req Request, ev Evaluation) (notification.Rendered, error) {
	ov, err := s.settings.Overview(ctx)
	if err != nil {
		return notification.Rendered{}, err
	}
	reason := strings.TrimSpace(req.ReasonText)
	if req.ReasonLabel != nil && *req.ReasonLabel != "" {
		if reason != "" {
			reason = *req.ReasonLabel + " - " + reason
		} else {
			reason = *req.ReasonLabel
		}
	}
	p := ev.parsed
	return notification.ApprovalRequired(notification.LeaveEmail{
		EmployeeName: c.Employee.Name, ManagerName: c.Manager.Name, ManagerEmail: c.Manager.Email, CC: ov.FinanceCC,
		LeaveType: p.leaveType, StartDate: p.interval.StartDate, EndDate: p.interval.EndDate,
		StartTime: p.interval.StartTime, EndTime: p.interval.EndTime,
		TotalMinutes: ev.Calculation.TotalMinutes, MinutesPerDay: ev.MinutesPerDay, Reason: reason,
		ReviewLink: fmt.Sprintf("%s/leader/requests/%d", s.origin, req.ID),
	}), nil
}

// SubmitError carries the blocking issues found when re-validating at submit time.
type SubmitError struct{ Issues []Issue }

func (e *SubmitError) Error() string { return "submission blocked by validation issues" }

func validation(issues []Issue) error {
	v := httpx.Validation{}
	for _, is := range issues {
		v.Add(cmpOr(is.Field, "form"), is.Message)
	}
	return v.Err()
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// Submit re-validates everything inside one transaction and moves the request to PENDING.
// The idempotency key makes retries (double clicks, network retries) return the same result.
func (s *Service) Submit(ctx context.Context, employeeID, id uint64, version uint32, key string) (Request, error) {
	if l := len(key); l < 16 || l > 128 {
		return Request{}, httpx.E(http.StatusBadRequest, "idempotency_key_required", "A valid Idempotency-Key header is required.")
	}
	keyHash := sha256.Sum256([]byte(key))
	reqHash := sha256.Sum256(fmt.Appendf(nil, "%d:%d", id, version))
	now := s.now()

	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)

		// Replay of an earlier successful submit with the same key returns that result.
		if prev, err := q.GetIdempotencyKey(ctx, store.GetIdempotencyKeyParams{ActorID: employeeID, Scope: idempotencyScope, KeyHash: keyHash[:]}); err == nil {
			if string(prev.RequestHash) != string(reqHash[:]) || uint64(prev.ResourceID.Int64) != id {
				return ErrIdempotency
			}
			return errReplay
		} else if !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		emp, err := q.LockEmployeeForSubmit(ctx, employeeID) // serializes this employee's submissions
		if err != nil {
			return err
		}
		if !emp.Active {
			return httpx.ErrForbidden
		}
		cur, err := q.LockOwnLeaveRequest(ctx, store.LockOwnLeaveRequestParams{ID: id, EmployeeID: employeeID})
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.Status != store.LeaveRequestsStatusDRAFT && cur.Status != store.LeaveRequestsStatusNEEDSCHANGES {
			return ErrNotEditable
		}
		if cur.Version != version {
			return httpx.ErrConflict
		}

		f := Fields{ReasonCode: nstr(cur.ReasonCode), ReasonText: cur.ReasonText,
			StartDate: ndate(cur.StartDate), EndDate: ndate(cur.EndDate), StartTime: nclock(cur.StartTime), EndTime: nclock(cur.EndTime)}
		if cur.LeaveType.Valid {
			t := cur.LeaveType.String
			f.LeaveType = &t
		}
		ev, err := s.evaluateFields(ctx, q, employeeID, id, f, true)
		if err != nil {
			return err
		}
		if !ev.CanSubmit {
			return &SubmitError{Issues: ev.Issues}
		}
		mgr, err := q.GetManagerInfo(ctx, uint64(emp.ManagerID.Int64))
		if err != nil {
			return err
		}

		code := cur.RequestCode
		if !code.Valid {
			c, err := allocateCode(ctx, q, now, ev.policy.Timezone)
			if err != nil {
				return err
			}
			code = sql.NullString{String: c, Valid: true}
		}
		snap := mustJSON(snapshot(ev.policy, *ev.Calculation))
		n, err := q.SubmitLeaveRequest(ctx, store.SubmitLeaveRequestParams{
			RequestCode: code, ReviewerID: sql.NullInt64{Int64: int64(mgr.ID), Valid: true},
			TotalMinutes: uint32(ev.Calculation.TotalMinutes), PolicyVersionID: sql.NullInt64{Int64: int64(ev.policy.VersionID), Valid: true},
			CalculationSnapshot: dbtypes.JSON(snap), ManagerName: nullStr(mgr.FullName), ManagerEmail: nullStr(mgr.Email),
			Now: sql.NullTime{Time: now, Valid: true}, ID: id, EmployeeID: employeeID, Version: version,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}

		if err := q.DeleteLeaveDays(ctx, id); err != nil {
			return err
		}
		for _, d := range ev.Calculation.Days {
			if err := q.InsertLeaveDay(ctx, store.InsertLeaveDayParams{LeaveRequestID: id, WorkDate: d.Date.UTC(), Minutes: uint16(d.Minutes)}); err != nil {
				return err
			}
		}
		submitNo := cur.SubmitCount + 1
		p := ev.parsed
		if err := q.InsertLeaveVersion(ctx, store.InsertLeaveVersionParams{
			LeaveRequestID: id, SubmitNo: submitNo, LeaveType: p.leaveType,
			StartDate: p.interval.StartDate.UTC(), EndDate: p.interval.EndDate.UTC(),
			StartTime: p.interval.StartTime.SQL(), EndTime: p.interval.EndTime.SQL(),
			TotalMinutes: uint32(ev.Calculation.TotalMinutes), ReasonCode: nullStr(p.reasonCode), ReasonText: p.reasonText,
			ReviewerID: mgr.ID, ManagerName: mgr.FullName, ManagerEmail: mgr.Email,
			PolicyVersionID: ev.policy.VersionID, CalculationSnapshot: snap, CreatedBy: employeeID,
		}); err != nil {
			return err
		}
		action := "SUBMITTED"
		if cur.Status == store.LeaveRequestsStatusNEEDSCHANGES {
			action = "RESUBMITTED"
		}
		meta, _ := json.Marshal(map[string]any{"total_minutes": ev.Calculation.TotalMinutes, "reviewer_id": mgr.ID, "policy_version_no": ev.policy.VersionNo})
		if err := q.InsertLeaveEvent(ctx, store.InsertLeaveEventParams{
			LeaveRequestID: id, Action: action, ActorID: actor(employeeID),
			FromStatus: nullStr(string(cur.Status)), ToStatus: nullStr("PENDING"),
			RequestVersion: version + 1, SubmitNo: sql.NullInt16{Int16: int16(submitNo), Valid: true}, Metadata: meta,
		}); err != nil {
			return err
		}
		// Notification + Mock Email outbox rows are written in this same transaction (never a real send).
		if err := notification.Submitted(ctx, tx, notification.Ref{
			RequestID: id, Code: code.String, Version: version + 1, SubmitNo: submitNo,
			EmployeeID: employeeID, EmployeeName: emp.FullName, ReviewerID: mgr.ID, ReviewerName: mgr.FullName,
		}, action == "RESUBMITTED"); err != nil {
			return err
		}
		return q.InsertIdempotencyKey(ctx, store.InsertIdempotencyKeyParams{
			ActorID: employeeID, Scope: idempotencyScope, KeyHash: keyHash[:], RequestHash: reqHash[:],
			ResourceID: sql.NullInt64{Int64: int64(id), Valid: true}, ExpiresAt: now.Add(idempotencyTTL),
		})
	})
	if errors.Is(err, errReplay) || database.IsDuplicateKey(err) {
		// Replay, or a concurrent identical submit committed first: report the current state.
		return s.Get(ctx, employeeID, id)
	}
	if err != nil {
		return Request{}, err
	}
	return s.Get(ctx, employeeID, id)
}

var errReplay = errors.New("idempotent replay")

// allocateCode returns the next LR-YYYY-NNNNNN code for the submission year (company time zone).
func allocateCode(ctx context.Context, q *store.Queries, now time.Time, tz string) (string, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	year := uint16(now.In(loc).Year())
	if err := q.EnsureRequestCodeSequence(ctx, year); err != nil {
		return "", err
	}
	n, err := q.LockRequestCodeSequence(ctx, year)
	if err != nil {
		return "", err
	}
	if err := q.BumpRequestCodeSequence(ctx, year); err != nil {
		return "", err
	}
	return fmt.Sprintf("LR-%d-%06d", year, n), nil
}

func actor(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: true} }

func nullStr(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func nstr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func ntime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func ndate(t sql.NullTime) *string {
	if !t.Valid {
		return nil
	}
	s := civil.DateOf(t.Time).String()
	return &s
}

func nclock(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	c, err := civil.ParseClock(s.String)
	if err != nil {
		return nil
	}
	v := c.String()
	return &v
}

func nullDate(p *string) sql.NullTime {
	d, err := civil.Parse(str(p))
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: d.UTC(), Valid: true}
}

func nullClock(p *string) sql.NullString {
	c, err := civil.ParseClock(str(p))
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: c.SQL(), Valid: true}
}
