// Package settings manages company settings and versioned leave policies.
// A policy version is immutable: every change inserts a new version and moves the pointer, so leave
// requests keep the exact rules they were calculated with.
package settings

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

var LeaveUnits = []int{15, 30, 60}

type LeaveTypeRule struct {
	AdvanceNoticeDays int  `json:"advance_notice_days"`
	AllowPastDates    bool `json:"allow_past_dates"`
	MaxPastDays       int  `json:"max_past_days"`
	ReasonRequired    bool `json:"reason_required"`
}

// PolicyInput is what an Admin submits. Times are HH:mm, workdays are ISO weekdays (1 = Monday ... 7 = Sunday).
type PolicyInput struct {
	Version                 uint32                   `json:"version"` // company_settings version (optimistic lock)
	Timezone                string                   `json:"timezone"`
	WorkStart               string                   `json:"work_start"`
	WorkEnd                 string                   `json:"work_end"`
	LunchStart              string                   `json:"lunch_start"`
	LunchEnd                string                   `json:"lunch_end"`
	Workdays                []int                    `json:"workdays"`
	MinLeaveMinutes         int                      `json:"min_leave_minutes"`
	LeaveUnitMinutes        int                      `json:"leave_unit_minutes"`
	ReminderBasis           string                   `json:"reminder_basis"`
	ReminderIntervalMinutes int                      `json:"reminder_interval_minutes"`
	ReminderMaxCount        int                      `json:"reminder_max_count"`
	LeaveTypes              map[string]LeaveTypeRule `json:"leave_types"`
	ChangeNote              string                   `json:"change_note"`
}

// Policy is a stored, validated policy version.
type Policy struct {
	VersionID               uint64                   `json:"version_id"`
	VersionNo               uint32                   `json:"version_no"`
	Timezone                string                   `json:"timezone"`
	WorkStart               civil.Clock              `json:"-"`
	WorkEnd                 civil.Clock              `json:"-"`
	LunchStart              civil.Clock              `json:"-"`
	LunchEnd                civil.Clock              `json:"-"`
	WorkdaysMask            uint8                    `json:"-"`
	MinutesPerDay           int                      `json:"minutes_per_day"`
	MinLeaveMinutes         int                      `json:"min_leave_minutes"`
	LeaveUnitMinutes        int                      `json:"leave_unit_minutes"`
	ReminderBasis           string                   `json:"reminder_basis"`
	ReminderIntervalMinutes int                      `json:"reminder_interval_minutes"`
	ReminderMaxCount        int                      `json:"reminder_max_count"`
	LeaveTypes              map[string]LeaveTypeRule `json:"leave_types"`
	ChangeNote              string                   `json:"change_note"`
	CreatedAt               time.Time                `json:"created_at"`
}

// policyJSON adds the human-readable time and weekday fields.
type policyJSON struct {
	Policy
	WorkStart  string `json:"work_start"`
	WorkEnd    string `json:"work_end"`
	LunchStart string `json:"lunch_start"`
	LunchEnd   string `json:"lunch_end"`
	Workdays   []int  `json:"workdays"`
}

func (p Policy) JSON() policyJSON {
	return policyJSON{Policy: p, WorkStart: p.WorkStart.String(), WorkEnd: p.WorkEnd.String(),
		LunchStart: p.LunchStart.String(), LunchEnd: p.LunchEnd.String(), Workdays: MaskToWeekdays(p.WorkdaysMask)}
}

// MaskToWeekdays converts the storage bitmask (bit 0 = Monday) to ISO weekdays.
func MaskToWeekdays(mask uint8) []int {
	var out []int
	for i := range 7 {
		if mask&(1<<i) != 0 {
			out = append(out, i+1)
		}
	}
	return out
}

func WeekdaysToMask(days []int) uint8 {
	var m uint8
	for _, d := range days {
		if d >= 1 && d <= 7 {
			m |= 1 << (d - 1)
		}
	}
	return m
}

// Validate checks an Admin's input against the active leave types and returns the normalized policy
// (without version fields).
func Validate(in PolicyInput, types []string) (Policy, error) {
	v := httpx.Validation{}
	p := Policy{
		Timezone:                strings.TrimSpace(in.Timezone),
		MinLeaveMinutes:         in.MinLeaveMinutes,
		LeaveUnitMinutes:        in.LeaveUnitMinutes,
		ReminderBasis:           in.ReminderBasis,
		ReminderIntervalMinutes: in.ReminderIntervalMinutes,
		ReminderMaxCount:        in.ReminderMaxCount,
		ChangeNote:              strings.TrimSpace(in.ChangeNote),
		LeaveTypes:              map[string]LeaveTypeRule{},
	}

	if _, err := time.LoadLocation(p.Timezone); err != nil || p.Timezone == "" || p.Timezone == "Local" {
		v.Add("timezone", "Enter a valid IANA time zone, e.g. Asia/Ho_Chi_Minh.")
	}
	clock := func(field, s string) civil.Clock {
		c, err := civil.ParseClock(s)
		if err != nil || c >= civil.EndOfDay {
			v.Add(field, "Enter a time as HH:mm.")
		} else if c%5 != 0 {
			v.Add(field, "Use 5-minute steps.")
		}
		return c
	}
	p.WorkStart, p.WorkEnd = clock("work_start", in.WorkStart), clock("work_end", in.WorkEnd)
	p.LunchStart, p.LunchEnd = clock("lunch_start", in.LunchStart), clock("lunch_end", in.LunchEnd)
	if p.WorkStart >= p.WorkEnd {
		v.Add("work_end", "Working hours must end after they start.")
	}
	if p.LunchStart > p.LunchEnd {
		v.Add("lunch_end", "Lunch must end after it starts.")
	}
	if p.LunchStart < p.WorkStart || p.LunchEnd > p.WorkEnd {
		v.Add("lunch_start", "Lunch break must be within working hours.")
	}
	p.MinutesPerDay = int(p.WorkEnd-p.WorkStart) - int(p.LunchEnd-p.LunchStart)
	if p.MinutesPerDay < 60 {
		v.Add("work_end", "A working day must have at least 1 working hour.")
	}

	seen := map[int]bool{}
	for _, d := range in.Workdays {
		if d < 1 || d > 7 || seen[d] {
			v.Add("workdays", "Select valid working days.")
		}
		seen[d] = true
	}
	if len(in.Workdays) == 0 {
		v.Add("workdays", "Select at least one working day.")
	}
	p.WorkdaysMask = WeekdaysToMask(in.Workdays)

	if !slices.Contains(LeaveUnits, p.LeaveUnitMinutes) {
		v.Add("leave_unit_minutes", "Leave unit must be 15, 30 or 60 minutes.")
	} else if p.MinLeaveMinutes < p.LeaveUnitMinutes || p.MinLeaveMinutes%p.LeaveUnitMinutes != 0 {
		v.Add("min_leave_minutes", "Minimum leave must be a multiple of the leave unit.")
	}
	if p.MinutesPerDay >= 60 && p.MinLeaveMinutes > p.MinutesPerDay {
		v.Add("min_leave_minutes", "Minimum leave cannot exceed one working day.")
	}

	if p.ReminderBasis != "WORKING_HOURS" && p.ReminderBasis != "CLOCK_HOURS" {
		v.Add("reminder_basis", "Choose working hours or clock hours.")
	}
	if p.ReminderIntervalMinutes < 15 || p.ReminderIntervalMinutes > 7*24*60 {
		v.Add("reminder_interval_minutes", "Reminder interval must be between 15 minutes and 7 days.")
	}
	if p.ReminderMaxCount < 0 || p.ReminderMaxCount > 20 {
		v.Add("reminder_max_count", "Number of reminders must be between 0 and 20.")
	}

	for _, t := range types {
		r, ok := in.LeaveTypes[t]
		key := "leave_types." + t
		switch {
		case !ok:
			v.Add(key, "Rules for this leave type are required.")
		case r.AdvanceNoticeDays < 0 || r.AdvanceNoticeDays > 365:
			v.Add(key, "Advance notice must be between 0 and 365 days.")
		case r.MaxPastDays < 0 || r.MaxPastDays > 365:
			v.Add(key, "Past days must be between 0 and 365.")
		case !r.AllowPastDates && r.MaxPastDays != 0:
			v.Add(key, "Past days can only be set when past dates are allowed.")
		case r.AllowPastDates && r.MaxPastDays == 0:
			v.Add(key, "Set how many past days are allowed.")
		}
		p.LeaveTypes[t] = r
	}
	for t := range in.LeaveTypes {
		if !slices.Contains(types, t) {
			v.Add("leave_types", fmt.Sprintf("Unknown leave type %q.", t))
		}
	}
	if len([]rune(p.ChangeNote)) > 500 {
		v.Add("change_note", "Change note must be at most 500 characters.")
	}
	return p, v.Err()
}
