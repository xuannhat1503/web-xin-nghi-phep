package leave

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/notification"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

// Cancel withdraws a PENDING or NEEDS_CHANGES request (owner only). APPROVED requests are final.
func (s *Service) Cancel(ctx context.Context, employeeID, id uint64, version uint32) (Request, error) {
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.LockLeaveRequest(ctx, id)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && cur.EmployeeID != employeeID) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.Status != store.LeaveRequestsStatusPENDING && cur.Status != store.LeaveRequestsStatusNEEDSCHANGES {
			return ErrNotEditable
		}
		n, err := q.CancelLeaveRequest(ctx, store.CancelLeaveRequestParams{
			Now: sql.NullTime{Time: s.now(), Valid: true}, ID: id, EmployeeID: employeeID, Version: version,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		if err := q.InsertLeaveEvent(ctx, store.InsertLeaveEventParams{
			LeaveRequestID: id, Action: "CANCELLED", ActorID: actor(employeeID),
			FromStatus: nullStr(string(cur.Status)), ToStatus: nullStr("CANCELLED"), RequestVersion: version + 1,
		}); err != nil {
			return err
		}
		ref, err := RefFor(ctx, q, id, version+1)
		if err != nil {
			return err
		}
		return notification.Cancelled(ctx, tx, ref)
	})
	if err != nil {
		return Request{}, err
	}
	return s.Get(ctx, employeeID, id)
}

// RefFor loads the people involved in a request for notifications.
func RefFor(ctx context.Context, q *store.Queries, id uint64, version uint32) (notification.Ref, error) {
	cur, err := q.LockLeaveRequest(ctx, id)
	if err != nil {
		return notification.Ref{}, err
	}
	emp, err := q.GetEmployeeByID(ctx, cur.EmployeeID)
	if err != nil {
		return notification.Ref{}, err
	}
	ref := notification.Ref{
		RequestID: id, Code: cur.RequestCode.String, Version: version, SubmitNo: cur.SubmitCount,
		EmployeeID: emp.ID, EmployeeName: emp.FullName,
	}
	if cur.ReviewerID.Valid {
		rv, err := q.GetEmployeeByID(ctx, uint64(cur.ReviewerID.Int64))
		if err != nil {
			return notification.Ref{}, err
		}
		ref.ReviewerID, ref.ReviewerName = rv.ID, rv.FullName
	}
	return ref, nil
}

type TimelineEvent struct {
	Action     string    `json:"action"`
	Actor      string    `json:"actor"`
	FromStatus *string   `json:"from_status"`
	ToStatus   *string   `json:"to_status"`
	SubmitNo   *int16    `json:"submit_no"`
	Comment    *string   `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

type Version struct {
	SubmitNo     uint16    `json:"submit_no"`
	LeaveType    string    `json:"leave_type"`
	StartDate    string    `json:"start_date"`
	EndDate      string    `json:"end_date"`
	StartTime    string    `json:"start_time"`
	EndTime      string    `json:"end_time"`
	TotalMinutes uint32    `json:"total_minutes"`
	ReasonCode   *string   `json:"reason_code"`
	ReasonText   string    `json:"reason_text"`
	Reviewer     string    `json:"reviewer"`
	CreatedAt    time.Time `json:"created_at"`
}

// History returns the audit timeline and submitted versions. Callers must have authorized access first.
func History(ctx context.Context, q *store.Queries, id uint64) ([]TimelineEvent, []Version, error) {
	evs, err := q.ListLeaveRequestEvents(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	vs, err := q.ListLeaveRequestVersions(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	timeline := make([]TimelineEvent, 0, len(evs))
	for _, e := range evs {
		t := TimelineEvent{Action: e.Action, Actor: "System", FromStatus: nstr(e.FromStatus), ToStatus: nstr(e.ToStatus),
			Comment: nstr(e.Comment), CreatedAt: e.CreatedAt}
		if e.ActorName.Valid {
			t.Actor = e.ActorName.String
		}
		if e.SubmitNo.Valid {
			n := e.SubmitNo.Int16
			t.SubmitNo = &n
		}
		timeline = append(timeline, t)
	}
	versions := make([]Version, 0, len(vs))
	for _, v := range vs {
		st, _ := civil.ParseClock(v.StartTime)
		et, _ := civil.ParseClock(v.EndTime)
		versions = append(versions, Version{
			SubmitNo: v.SubmitNo, LeaveType: string(v.LeaveType),
			StartDate: civil.DateOf(v.StartDate).String(), EndDate: civil.DateOf(v.EndDate).String(),
			StartTime: st.String(), EndTime: et.String(), TotalMinutes: v.TotalMinutes,
			ReasonCode: nstr(v.ReasonCode), ReasonText: v.ReasonText, Reviewer: v.ManagerNameSnapshot, CreatedAt: v.CreatedAt,
		})
	}
	return timeline, versions, nil
}

// OwnHistory is History for the request owner (404 for anyone else).
func (s *Service) OwnHistory(ctx context.Context, employeeID, id uint64) ([]TimelineEvent, []Version, error) {
	if _, err := s.Get(ctx, employeeID, id); err != nil {
		return nil, nil, err
	}
	return History(ctx, s.q, id)
}
