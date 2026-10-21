package holiday

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/dbtypes"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

const (
	MinYear = 2000
	MaxYear = 2200
)

var (
	ErrDateTaken     = httpx.E(http.StatusConflict, "date_taken", "Another calendar entry already exists on this date.")
	ErrNotDraft      = httpx.E(http.StatusConflict, "not_draft", "Only draft entries can be deleted. Revert it to draft first.")
	ErrYearEmpty     = httpx.E(http.StatusUnprocessableEntity, "year_empty", "Add the year's holidays before confirming it.")
	ErrYearNotExists = httpx.E(http.StatusNotFound, "year_not_found", "This calendar year has not been set up yet.")
)

var kinds = []Kind{KindNational, KindCompany, KindCompensatory, KindMakeup}

// Admin is the calendar administration service.
type Admin struct {
	db *sql.DB
	q  *store.Queries
}

func NewAdmin(db *sql.DB) *Admin { return &Admin{db: db, q: store.New(db)} }

type Day struct {
	ID            uint64     `json:"id"`
	Date          civil.Date `json:"date"`
	Name          string     `json:"name"`
	Kind          Kind       `json:"kind"`
	Status        string     `json:"status"`
	StartTime     *string    `json:"start_time"`
	EndTime       *string    `json:"end_time"`
	Source        string     `json:"source"`
	Note          string     `json:"note"`
	Seeded        bool       `json:"seeded"`
	AdminModified bool       `json:"admin_modified"`
	Version       uint32     `json:"version"`
	ConfirmedAt   *time.Time `json:"confirmed_at"`
	ConfirmedBy   *string    `json:"confirmed_by"`
}

type Warning struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Params  map[string]any `json:"params,omitempty"`
}

type YearSummary struct {
	Year        int        `json:"year"`
	Status      string     `json:"status"`
	Version     uint32     `json:"version"`
	ConfirmedAt *time.Time `json:"confirmed_at"`
	TotalDays   int64      `json:"total_days"`
	DraftDays   int64      `json:"draft_days"`
}

type Event struct {
	DayID     *uint64      `json:"day_id"`
	Action    string       `json:"action"`
	Actor     string       `json:"actor"`
	Before    dbtypes.JSON `json:"before,omitempty"`
	After     dbtypes.JSON `json:"after,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

type YearView struct {
	Year        int              `json:"year"`
	Exists      bool             `json:"exists"`
	Status      string           `json:"status"`
	Version     uint32           `json:"version"`
	ConfirmedAt *time.Time       `json:"confirmed_at"`
	ConfirmedBy *string          `json:"confirmed_by"`
	Days        []Day            `json:"days"`
	Warnings    []Warning        `json:"warnings"`
	Workdays    []int            `json:"workdays"`
	Requests    map[string]int64 `json:"affected_requests"` // open/approved requests overlapping the year
	Events      []Event          `json:"events"`
	Years       []YearSummary    `json:"years"`
}

func validYear(y int) error {
	if y < MinYear || y > MaxYear {
		return httpx.E(http.StatusNotFound, "not_found", "Resource not found")
	}
	return nil
}

func (a *Admin) Year(ctx context.Context, year int) (YearView, error) {
	if err := validYear(year); err != nil {
		return YearView{}, err
	}
	v := YearView{Year: year, Status: "DRAFT", Days: []Day{}, Warnings: []Warning{}, Events: []Event{}, Requests: map[string]int64{}}

	mask, err := a.q.GetCurrentWorkdaysMask(ctx)
	if err != nil {
		return v, err
	}
	for i := range 7 {
		if mask&(1<<i) != 0 {
			v.Workdays = append(v.Workdays, i+1)
		}
	}
	years, err := a.q.ListCalendarYears(ctx)
	if err != nil {
		return v, err
	}
	for _, y := range years {
		v.Years = append(v.Years, YearSummary{Year: int(y.Year), Status: string(y.Status), Version: y.Version,
			ConfirmedAt: ptrTime(y.ConfirmedAt), TotalDays: y.TotalDays, DraftDays: y.DraftDays})
	}

	y, err := a.q.GetCalendarYear(ctx, uint16(year))
	if errors.Is(err, sql.ErrNoRows) {
		v.Warnings = append(v.Warnings, Warning{"year_missing", "This year has no calendar yet. Seed the common Vietnamese holidays or add days manually.", nil})
		return v, nil
	}
	if err != nil {
		return v, err
	}
	v.Exists, v.Status, v.Version = true, string(y.Status), y.Version
	v.ConfirmedAt, v.ConfirmedBy = ptrTime(y.ConfirmedAt), ptrStr(y.ConfirmedByName)

	rows, err := a.q.ListCalendarDaysAdmin(ctx, uint16(year))
	if err != nil {
		return v, err
	}
	for _, r := range rows {
		v.Days = append(v.Days, Day{
			ID: r.ID, Date: civil.DateOf(r.DayDate), Name: r.Name, Kind: Kind(r.Kind), Status: string(r.Status),
			StartTime: clockPtr(r.StartTime), EndTime: clockPtr(r.EndTime), Source: r.Source, Note: r.Note,
			Seeded: r.SeedKey.Valid, AdminModified: r.AdminModified, Version: r.Version,
			ConfirmedAt: ptrTime(r.ConfirmedAt), ConfirmedBy: ptrStr(r.ConfirmedByName),
		})
	}
	v.Warnings = yearWarnings(year, v.Status, v.Days, Workdays(mask))

	counts, err := a.q.CountRequestsInYear(ctx, store.CountRequestsInYearParams{
		YearStart: nullDate(civil.D(year, time.January, 1)), YearEnd: nullDate(civil.D(year, time.December, 31)),
	})
	if err != nil {
		return v, err
	}
	for _, c := range counts {
		v.Requests[string(c.Status)] = c.N
	}

	events, err := a.q.ListCalendarYearEvents(ctx, uint16(year))
	if err != nil {
		return v, err
	}
	for _, e := range events {
		ev := Event{Action: e.Action, Actor: "System", Before: e.BeforeData, After: e.AfterData, CreatedAt: e.CreatedAt}
		if e.ActorName.Valid {
			ev.Actor = e.ActorName.String
		}
		if e.CalendarDayID.Valid {
			id := uint64(e.CalendarDayID.Int64)
			ev.DayID = &id
		}
		v.Events = append(v.Events, ev)
	}
	return v, nil
}

// yearWarnings flags calendars that are probably incomplete. They inform; only an empty year blocks confirmation.
func yearWarnings(year int, status string, days []Day, wd Workdays) []Warning {
	w := []Warning{}
	if status != "CONFIRMED" {
		w = append(w, Warning{"year_not_confirmed", "Leave requests for this year cannot be submitted until the calendar is confirmed.", nil})
	}
	if len(days) == 0 {
		return append(w, Warning{"year_empty", "No holidays have been added for this year.", nil})
	}
	drafts, national, tet := 0, 0, 0
	tetFrom, tetTo := civil.D(year, time.January, 19), civil.D(year, time.February, 25)
	for _, d := range days {
		if d.Status == "DRAFT" {
			drafts++
		}
		if d.Kind == KindNational {
			national++
			if !d.Date.Before(tetFrom) && !d.Date.After(tetTo) {
				tet++
			}
			if !wd.IsWorkday(d.Date) {
				if !hasCompensatoryAfter(days, d.Date) {
					w = append(w, Warning{"no_compensatory", fmt.Sprintf("%s (%s) falls on a rest day and has no compensatory day after it.", d.Name, d.Date), map[string]any{"name": d.Name, "date": d.Date.String()}})
				}
			}
		}
	}
	if drafts > 0 {
		w = append(w, Warning{"drafts", fmt.Sprintf("%d day(s) are still DRAFT and are not used in calculations.", drafts), map[string]any{"n": drafts}})
	}
	if national == 0 {
		w = append(w, Warning{"no_national", "No national holidays have been added.", nil})
	}
	if tet < 5 {
		w = append(w, Warning{"tet_incomplete", fmt.Sprintf("Only %d Lunar New Year day(s) found between 19 Jan and 25 Feb (5 expected).", tet), map[string]any{"n": tet}})
	}
	return w
}

func hasCompensatoryAfter(days []Day, d civil.Date) bool {
	return slices.ContainsFunc(days, func(x Day) bool {
		return x.Kind == KindCompensatory && x.Date.After(d) && d.DaysUntil(x.Date) <= 14
	})
}

// DayInput is the create/update payload.
type DayInput struct {
	Version   uint32  `json:"version"`
	Date      string  `json:"date"`
	Name      string  `json:"name"`
	Kind      Kind    `json:"kind"`
	StartTime *string `json:"start_time"`
	EndTime   *string `json:"end_time"`
	Source    string  `json:"source"`
	Note      string  `json:"note"`
}

type dayValues struct {
	date       civil.Date
	name, note string
	source     string
	kind       Kind
	start, end sql.NullString
}

func validateDay(in DayInput, wd Workdays) (dayValues, error) {
	v := httpx.Validation{}
	out := dayValues{
		name: strings.Join(strings.Fields(in.Name), " "), kind: in.Kind,
		note: strings.TrimSpace(in.Note), source: strings.TrimSpace(in.Source),
	}
	d, err := civil.Parse(in.Date)
	if err != nil {
		v.Add("date", "Enter a valid date.")
	} else if d.Year < MinYear || d.Year > MaxYear {
		v.Add("date", fmt.Sprintf("Year must be between %d and %d.", MinYear, MaxYear))
	}
	out.date = d
	if l := len([]rune(out.name)); l < 2 || l > 200 {
		v.Add("name", "Name must be 2-200 characters.")
	}
	if !slices.Contains(kinds, in.Kind) {
		v.Add("kind", "Choose a valid type.")
	}
	if len([]rune(out.note)) > 1000 {
		v.Add("note", "Note must be at most 1000 characters.")
	}
	if len([]rune(out.source)) > 255 {
		v.Add("source", "Source must be at most 255 characters.")
	}
	if err == nil && in.Kind == KindMakeup && wd.IsWorkday(d) {
		v.Add("kind", "A makeup workday must fall on a regular day off (e.g. Saturday or Sunday).")
	}

	hasStart := in.StartTime != nil && strings.TrimSpace(*in.StartTime) != ""
	hasEnd := in.EndTime != nil && strings.TrimSpace(*in.EndTime) != ""
	switch {
	case !hasStart && !hasEnd:
	case in.Kind != KindCompany:
		v.Add("start_time", "Only company holidays can be part of a day.")
	case hasStart != hasEnd:
		v.Add("end_time", "Set both start and end time, or neither for a full day.")
	default:
		s, err1 := civil.ParseClock(*in.StartTime)
		e, err2 := civil.ParseClock(*in.EndTime)
		if err1 != nil || err2 != nil || s%5 != 0 || e%5 != 0 {
			v.Add("start_time", "Enter times as HH:mm in 5-minute steps.")
		} else if s >= e {
			v.Add("end_time", "End time must be after start time.")
		} else {
			out.start = sql.NullString{String: s.SQL(), Valid: true}
			out.end = sql.NullString{String: e.SQL(), Valid: true}
		}
	}
	return out, v.Err()
}

func (a *Admin) workdays(ctx context.Context, q *store.Queries) (Workdays, error) {
	m, err := q.GetCurrentWorkdaysMask(ctx)
	return Workdays(m), err
}

// ChangeResult reports how many existing requests were flagged because their dates include the change.
type ChangeResult struct {
	DayID            uint64           `json:"day_id,omitempty"`
	YearReopened     bool             `json:"year_reopened"`
	AffectedRequests map[string]int64 `json:"affected_requests"`
}

// mutate runs a calendar change with the year locked. A CONFIRMED year is reopened (submissions for it are
// blocked again until re-confirmed) and requests covering the touched dates are flagged — never recalculated.
func (a *Admin) mutate(ctx context.Context, actorID uint64, year int, dates []civil.Date, fn func(q *store.Queries) error) (ChangeResult, error) {
	res := ChangeResult{AffectedRequests: map[string]int64{}}
	err := database.InTx(ctx, a.db, func(tx *sql.Tx) error {
		q := a.q.WithTx(tx)
		if err := q.EnsureCalendarYear(ctx, uint16(year)); err != nil {
			return err
		}
		if _, err := q.LockCalendarYear(ctx, uint16(year)); err != nil {
			return err
		}
		if err := fn(q); err != nil {
			return err
		}
		n, err := q.ReopenCalendarYear(ctx, uint16(year))
		if err != nil {
			return err
		}
		if n == 1 {
			res.YearReopened = true
			if err := q.InsertCalendarDayEvent(ctx, store.InsertCalendarDayEventParams{
				Year: uint16(year), Action: "YEAR_REOPENED", ActorID: actorParam(actorID),
				AfterData: dbtypes.JSON(`{"status":"DRAFT","reason":"calendar changed after confirmation"}`),
			}); err != nil {
				return err
			}
		}
		for _, d := range dates {
			counts, err := q.CountRequestsCoveringDate(ctx, store.CountRequestsCoveringDateParams{DayDate: nullDate(d)})
			if err != nil {
				return err
			}
			for _, c := range counts {
				res.AffectedRequests[string(c.Status)] += c.N
			}
			if _, err := q.FlagRequestsForCalendarChange(ctx, store.FlagRequestsForCalendarChangeParams{DayDate: nullDate(d)}); err != nil {
				return err
			}
		}
		return nil
	})
	return res, err
}

func (a *Admin) CreateDay(ctx context.Context, actorID uint64, in DayInput) (ChangeResult, error) {
	wd, err := a.workdays(ctx, a.q)
	if err != nil {
		return ChangeResult{}, err
	}
	val, err := validateDay(in, wd)
	if err != nil {
		return ChangeResult{}, err
	}
	var id int64
	res, err := a.mutate(ctx, actorID, val.date.Year, []civil.Date{val.date}, func(q *store.Queries) error {
		var err error
		id, err = q.InsertCalendarDay(ctx, store.InsertCalendarDayParams{
			DayDate: val.date.UTC(), Name: val.name, Kind: store.CalendarDaysKind(val.kind),
			StartTime: val.start, EndTime: val.end, Source: val.source, Note: val.note, ActorID: actorParam(actorID),
		})
		if database.IsDuplicateKey(err) {
			return ErrDateTaken
		}
		if err != nil {
			return err
		}
		return a.event(ctx, q, val.date.Year, uint64(id), actorID, "CREATED", nil, val)
	})
	res.DayID = uint64(id)
	return res, err
}

func (a *Admin) UpdateDay(ctx context.Context, actorID, id uint64, in DayInput) (ChangeResult, error) {
	wd, err := a.workdays(ctx, a.q)
	if err != nil {
		return ChangeResult{}, err
	}
	val, err := validateDay(in, wd)
	if err != nil {
		return ChangeResult{}, err
	}
	cur, err := a.q.GetCalendarDayForUpdate(ctx, id) // read outside the tx only to find the year and old date
	if errors.Is(err, sql.ErrNoRows) {
		return ChangeResult{}, httpx.ErrNotFound
	}
	if err != nil {
		return ChangeResult{}, err
	}
	if val.date.Year != int(cur.Year) {
		v := httpx.Validation{}
		v.Add("date", "The date must stay within the same calendar year.")
		return ChangeResult{}, v.Err()
	}
	oldDate := civil.DateOf(cur.DayDate)
	dates := []civil.Date{oldDate}
	if val.date != oldDate {
		dates = append(dates, val.date)
	}
	res, err := a.mutate(ctx, actorID, int(cur.Year), dates, func(q *store.Queries) error {
		locked, err := q.GetCalendarDayForUpdate(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if locked.Version != in.Version {
			return httpx.ErrConflict
		}
		n, err := q.UpdateCalendarDay(ctx, store.UpdateCalendarDayParams{
			DayDate: val.date.UTC(), Name: val.name, Kind: store.CalendarDaysKind(val.kind),
			StartTime: val.start, EndTime: val.end, Source: val.source, Note: val.note,
			ActorID: actorParam(actorID), ID: id, Version: in.Version,
		})
		if database.IsDuplicateKey(err) {
			return ErrDateTaken
		}
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		return a.event(ctx, q, int(cur.Year), id, actorID, "UPDATED", dayJSON(locked), val)
	})
	res.DayID = id
	return res, err
}

type VersionInput struct {
	Version uint32 `json:"version"`
}

// SetDayStatus confirms a draft day or reverts a confirmed day to draft.
func (a *Admin) SetDayStatus(ctx context.Context, actorID, id uint64, confirm bool, in VersionInput) (ChangeResult, error) {
	cur, err := a.q.GetCalendarDayForUpdate(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ChangeResult{}, httpx.ErrNotFound
	}
	if err != nil {
		return ChangeResult{}, err
	}
	date := civil.DateOf(cur.DayDate)
	var dates []civil.Date
	if !confirm {
		dates = []civil.Date{date} // a rule stops applying: flag requests on that date
	}
	run := func(q *store.Queries) error {
		var n int64
		var err error
		now := sql.NullTime{Time: time.Now().UTC(), Valid: true}
		if confirm {
			n, err = q.ConfirmCalendarDay(ctx, store.ConfirmCalendarDayParams{ActorID: actorParam(actorID), Now: now, ID: id, Version: in.Version})
		} else {
			n, err = q.UnconfirmCalendarDay(ctx, store.UnconfirmCalendarDayParams{ActorID: actorParam(actorID), ID: id, Version: in.Version})
		}
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		action := "UNCONFIRMED"
		if confirm {
			action = "CONFIRMED"
		}
		return a.event(ctx, q, int(cur.Year), id, actorID, action, nil, nil)
	}
	if confirm {
		// Confirming a single day does not reopen a confirmed year (it cannot be in one as a draft).
		err := database.InTx(ctx, a.db, func(tx *sql.Tx) error { return run(a.q.WithTx(tx)) })
		return ChangeResult{DayID: id, AffectedRequests: map[string]int64{}}, err
	}
	res, err := a.mutate(ctx, actorID, int(cur.Year), dates, run)
	res.DayID = id
	return res, err
}

func (a *Admin) DeleteDay(ctx context.Context, actorID, id uint64, in VersionInput) (ChangeResult, error) {
	cur, err := a.q.GetCalendarDayForUpdate(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ChangeResult{}, httpx.ErrNotFound
	}
	if err != nil {
		return ChangeResult{}, err
	}
	if cur.Status != store.CalendarDaysStatusDRAFT {
		return ChangeResult{}, ErrNotDraft
	}
	res, err := a.mutate(ctx, actorID, int(cur.Year), nil, func(q *store.Queries) error {
		n, err := q.SoftDeleteCalendarDay(ctx, store.SoftDeleteCalendarDayParams{
			Now: sql.NullTime{Time: time.Now().UTC(), Valid: true}, ActorID: actorParam(actorID), ID: id, Version: in.Version,
		})
		if err != nil {
			return err
		}
		if n != 1 {
			// Either someone changed it or it was confirmed in the meantime.
			return httpx.ErrConflict
		}
		return a.event(ctx, q, int(cur.Year), id, actorID, "DELETED", dayJSON(cur), nil)
	})
	res.DayID = id
	return res, err
}

// ConfirmYear confirms every remaining draft day and the year itself in one transaction.
func (a *Admin) ConfirmYear(ctx context.Context, actorID uint64, year int, in VersionInput) (int64, error) {
	if err := validYear(year); err != nil {
		return 0, err
	}
	var confirmedDays int64
	err := database.InTx(ctx, a.db, func(tx *sql.Tx) error {
		q := a.q.WithTx(tx)
		y, err := q.LockCalendarYear(ctx, uint16(year))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrYearNotExists
		}
		if err != nil {
			return err
		}
		if y.Version != in.Version {
			return httpx.ErrConflict
		}
		if y.Status == store.CalendarYearsStatusCONFIRMED {
			return nil
		}
		days, err := q.ListCalendarDaysAdmin(ctx, uint16(year))
		if err != nil {
			return err
		}
		if len(days) == 0 {
			return ErrYearEmpty
		}
		now := sql.NullTime{Time: time.Now().UTC(), Valid: true}
		if confirmedDays, err = q.ConfirmDraftDaysOfYear(ctx, store.ConfirmDraftDaysOfYearParams{ActorID: actorParam(actorID), Now: now, Year: uint16(year)}); err != nil {
			return err
		}
		n, err := q.ConfirmCalendarYear(ctx, store.ConfirmCalendarYearParams{ActorID: actorParam(actorID), Now: now, Year: uint16(year), Version: in.Version})
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		after, _ := json.Marshal(map[string]any{"status": "CONFIRMED", "days_confirmed": confirmedDays, "total_days": len(days)})
		return q.InsertCalendarDayEvent(ctx, store.InsertCalendarDayEventParams{Year: uint16(year), Action: "YEAR_CONFIRMED", ActorID: actorParam(actorID), AfterData: after})
	})
	return confirmedDays, err
}

func (a *Admin) event(ctx context.Context, q *store.Queries, year int, dayID, actorID uint64, action string, before any, after any) error {
	var b, af dbtypes.JSON
	if before != nil {
		b, _ = json.Marshal(before)
	}
	if v, ok := after.(dayValues); ok {
		af, _ = json.Marshal(map[string]any{"date": v.date.String(), "name": v.name, "kind": v.kind,
			"start_time": nullClock(v.start), "end_time": nullClock(v.end), "source": v.source, "note": v.note})
	}
	return q.InsertCalendarDayEvent(ctx, store.InsertCalendarDayEventParams{
		Year: uint16(year), CalendarDayID: sql.NullInt64{Int64: int64(dayID), Valid: true},
		Action: action, ActorID: actorParam(actorID), BeforeData: b, AfterData: af,
	})
}

func dayJSON(d store.GetCalendarDayForUpdateRow) map[string]any {
	return map[string]any{"date": civil.DateOf(d.DayDate).String(), "name": d.Name, "kind": d.Kind, "status": d.Status,
		"start_time": nullClock(d.StartTime), "end_time": nullClock(d.EndTime), "source": d.Source, "note": d.Note}
}

// ConfirmedDay is the public view of the calendar.
type ConfirmedDay struct {
	Date      civil.Date `json:"date"`
	Name      string     `json:"name"`
	Kind      Kind       `json:"kind"`
	StartTime *string    `json:"start_time"`
	EndTime   *string    `json:"end_time"`
}

// Confirmed lists CONFIRMED days of a year for any active employee, plus whether the whole year is confirmed.
func (a *Admin) Confirmed(ctx context.Context, year int) (bool, []ConfirmedDay, error) {
	if err := validYear(year); err != nil {
		return false, nil, err
	}
	rows, err := a.q.ListConfirmedDays(ctx, store.ListConfirmedDaysParams{
		FromDate: civil.D(year, time.January, 1).UTC(), ToDate: civil.D(year, time.December, 31).UTC(),
	})
	if err != nil {
		return false, nil, err
	}
	out := make([]ConfirmedDay, 0, len(rows))
	for _, r := range rows {
		out = append(out, ConfirmedDay{Date: civil.DateOf(r.DayDate), Name: r.Name, Kind: Kind(r.Kind),
			StartTime: clockPtr(r.StartTime), EndTime: clockPtr(r.EndTime)})
	}
	y, err := a.q.GetCalendarYear(ctx, uint16(year))
	if errors.Is(err, sql.ErrNoRows) {
		return false, out, nil
	}
	if err != nil {
		return false, nil, err
	}
	return y.Status == store.CalendarYearsStatusCONFIRMED, out, nil
}

func actorParam(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: true} }
func nullDate(d civil.Date) sql.NullTime { return sql.NullTime{Time: d.UTC(), Valid: true} }

func ptrTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	return &t.Time
}

func ptrStr(s sql.NullString) *string {
	if !s.Valid {
		return nil
	}
	return &s.String
}

func clockPtr(s sql.NullString) *string {
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

func nullClock(s sql.NullString) *string { return clockPtr(s) }
