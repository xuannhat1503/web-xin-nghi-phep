// Package workcal is the WorkingCalendarService: it turns a leave interval into working minutes using the
// company policy and the CONFIRMED calendar. It is pure (no I/O); callers load the policy and calendar.
package workcal

import (
	"github.com/g-asiapacific/elm/backend/internal/civil"
)

const (
	KindNational     = "NATIONAL_HOLIDAY"
	KindCompany      = "COMPANY_HOLIDAY"
	KindCompensatory = "COMPENSATORY_HOLIDAY"
	KindMakeup       = "MAKEUP_WORKDAY"
)

// MaxDays bounds one request (a longer absence is split into several requests).
const MaxDays = 62

// Policy is the part of a leave policy the calculation needs.
type Policy struct {
	WorkStart, WorkEnd   civil.Clock
	LunchStart, LunchEnd civil.Clock
	WorkdaysMask         uint8 // bit 0 = Monday ... bit 6 = Sunday
}

// Rule is a CONFIRMED calendar entry. Start/End set = partial-day company holiday.
type Rule struct {
	Date       civil.Date
	Kind       string
	Name       string
	Start, End *civil.Clock
}

// Interval is the requested leave: from StartDate StartTime to EndDate EndTime.
type Interval struct {
	StartDate, EndDate civil.Date
	StartTime, EndTime civil.Clock
}

type Day struct {
	Date    civil.Date `json:"date"`
	Minutes int        `json:"minutes"`
	// Note explains a special day: "makeup_workday" or "partial_holiday".
	Note string `json:"note,omitempty"`
	Name string `json:"name,omitempty"`
}

type Excluded struct {
	Date   civil.Date `json:"date"`
	Reason string     `json:"reason"` // "rest_day" | "holiday"
	Kind   string     `json:"kind,omitempty"`
	Name   string     `json:"name,omitempty"`
}

type Result struct {
	TotalMinutes int        `json:"total_minutes"`
	Days         []Day      `json:"days"`
	Excluded     []Excluded `json:"excluded"`
}

func (p Policy) isWorkday(d civil.Date) bool {
	return p.WorkdaysMask&(1<<((int(d.Weekday())+6)%7)) != 0
}

func overlap(a1, a2, b1, b2 civil.Clock) int {
	lo, hi := max(a1, b1), min(a2, b2)
	if hi <= lo {
		return 0
	}
	return int(hi - lo)
}

// worked returns the working minutes in [from, to), excluding lunch.
func (p Policy) worked(from, to civil.Clock) int {
	from, to = max(from, p.WorkStart), min(to, p.WorkEnd)
	if to <= from {
		return 0
	}
	return int(to-from) - overlap(from, to, p.LunchStart, p.LunchEnd)
}

// Calculate returns the working minutes of the interval day by day. The interval must already be
// validated (StartDate <= EndDate, same-day StartTime < EndTime, at most MaxDays days).
func Calculate(p Policy, rules map[civil.Date]Rule, in Interval) Result {
	res := Result{Days: []Day{}, Excluded: []Excluded{}}
	for d := in.StartDate; !d.After(in.EndDate); d = d.AddDays(1) {
		from, to := p.WorkStart, p.WorkEnd
		if d == in.StartDate {
			from = max(from, in.StartTime)
		}
		if d == in.EndDate {
			to = min(to, in.EndTime)
		}

		rule, hasRule := rules[d]
		working := p.isWorkday(d)
		day := Day{Date: d}
		switch {
		case hasRule && rule.Kind == KindMakeup:
			working, day.Note, day.Name = true, "makeup_workday", rule.Name
		case hasRule && rule.Start == nil:
			// Full-day holiday (national, company or compensatory): never counted, even on a rest day.
			res.Excluded = append(res.Excluded, Excluded{Date: d, Reason: "holiday", Kind: rule.Kind, Name: rule.Name})
			continue
		}
		if !working {
			res.Excluded = append(res.Excluded, Excluded{Date: d, Reason: "rest_day"})
			continue
		}

		minutes := p.worked(from, to)
		if hasRule && rule.Start != nil && rule.Kind != KindMakeup {
			// Partial company holiday: the overlapping working time is already off.
			off := p.worked(max(from, *rule.Start), min(to, *rule.End))
			if off > 0 {
				minutes -= off
				day.Note, day.Name = "partial_holiday", rule.Name
			}
		}
		if minutes > 0 {
			day.Minutes = minutes
			res.Days = append(res.Days, day)
			res.TotalMinutes += minutes
		}
	}
	return res
}

// MinutesPerDay is the length of a full working day under the policy.
func (p Policy) MinutesPerDay() int { return p.worked(p.WorkStart, p.WorkEnd) }
