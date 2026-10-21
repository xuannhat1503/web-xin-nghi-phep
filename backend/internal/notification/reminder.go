package notification

import (
	"context"
	"database/sql"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/settings"
	"github.com/g-asiapacific/elm/backend/internal/store"
	"github.com/g-asiapacific/elm/backend/internal/workcal"
)

// Reminders notifies the current reviewer of PENDING requests that have waited longer than the configured
// interval, once per interval up to the configured maximum. It never approves, rejects or reassigns.
type Reminders struct {
	DB  *sql.DB
	Now func() time.Time
}

func (r Reminders) Run(ctx context.Context) error {
	q := store.New(r.DB)
	pol, err := settings.CurrentWith(ctx, q)
	if err != nil {
		return err
	}
	if pol.ReminderMaxCount == 0 || pol.ReminderIntervalMinutes <= 0 {
		return nil
	}
	loc, err := time.LoadLocation(pol.Timezone)
	if err != nil {
		loc = time.UTC
	}
	now := time.Now().UTC()
	if r.Now != nil {
		now = r.Now()
	}

	rows, err := r.DB.QueryContext(ctx,
		`SELECT lr.id, lr.request_code, lr.version, lr.submit_count, lr.submitted_at, lr.employee_id, e.full_name, lr.reviewer_id
		 FROM leave_requests lr JOIN employees e ON e.id = lr.employee_id
		 JOIN employees rv ON rv.id = lr.reviewer_id AND rv.active = TRUE
		 WHERE lr.status = 'PENDING' AND lr.submitted_at <= ?
		 ORDER BY lr.submitted_at LIMIT 500`,
		now.Add(-time.Duration(pol.ReminderIntervalMinutes)*time.Minute))
	if err != nil {
		return err
	}
	type due struct {
		ref       Ref
		submitted time.Time
	}
	var list []due
	for rows.Next() {
		var d due
		var code sql.NullString
		if err := rows.Scan(&d.ref.RequestID, &code, &d.ref.Version, &d.ref.SubmitNo, &d.submitted,
			&d.ref.EmployeeID, &d.ref.EmployeeName, &d.ref.ReviewerID); err != nil {
			rows.Close()
			return err
		}
		d.ref.Code = code.String
		list = append(list, d)
	}
	rows.Close()
	if err := rows.Err(); err != nil || len(list) == 0 {
		return err
	}

	var rules map[civil.Date]workcal.Rule
	wc := workcal.Policy{WorkStart: pol.WorkStart, WorkEnd: pol.WorkEnd, LunchStart: pol.LunchStart, LunchEnd: pol.LunchEnd, WorkdaysMask: pol.WorkdaysMask}
	if pol.ReminderBasis == "WORKING_HOURS" {
		from := civil.DateOf(list[0].submitted.In(loc))
		to := civil.DateOf(now.In(loc))
		if from.DaysUntil(to) > workcal.MaxDays {
			from = to.AddDays(-workcal.MaxDays)
		}
		days, err := q.ListConfirmedDays(ctx, store.ListConfirmedDaysParams{FromDate: from.UTC(), ToDate: to.UTC()})
		if err != nil {
			return err
		}
		rules = make(map[civil.Date]workcal.Rule, len(days))
		for _, d := range days {
			rule := workcal.Rule{Date: civil.DateOf(d.DayDate), Kind: string(d.Kind), Name: d.Name}
			if d.StartTime.Valid && d.EndTime.Valid {
				s, e1 := civil.ParseClock(d.StartTime.String)
				e, e2 := civil.ParseClock(d.EndTime.String)
				if e1 == nil && e2 == nil {
					rule.Start, rule.End = &s, &e
				}
			}
			rules[rule.Date] = rule
		}
	}

	for _, d := range list {
		elapsed := int(now.Sub(d.submitted).Minutes())
		if pol.ReminderBasis == "WORKING_HOURS" {
			elapsed = workingMinutes(wc, rules, d.submitted.In(loc), now.In(loc))
		}
		level := min(elapsed/pol.ReminderIntervalMinutes, pol.ReminderMaxCount)
		if level < 1 {
			continue
		}
		// Only the latest level is created; the dedup key (per submission, reviewer and level) prevents repeats,
		// and including the reviewer restarts reminders after a transfer.
		if err := Reminder(ctx, r.DB, d.ref, level); err != nil {
			return err
		}
	}
	return nil
}

func clockOf(t time.Time) civil.Clock { return civil.Clock(t.Hour()*60 + t.Minute()) }

func workingMinutes(p workcal.Policy, rules map[civil.Date]workcal.Rule, from, to time.Time) int {
	sd, ed := civil.DateOf(from), civil.DateOf(to)
	st, et := clockOf(from), clockOf(to)
	if sd.DaysUntil(ed) > workcal.MaxDays {
		sd, st = ed.AddDays(-workcal.MaxDays), 0
	}
	if sd == ed && et <= st {
		return 0
	}
	return workcal.Calculate(p, rules, workcal.Interval{StartDate: sd, EndDate: ed, StartTime: st, EndTime: et}).TotalMinutes
}
