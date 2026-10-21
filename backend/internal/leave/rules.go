// Package leave implements the employee side of leave requests: drafts with auto-save, live duration
// preview, review & confirm, and submission (DRAFT/NEEDS_CHANGES -> PENDING).
package leave

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/settings"
	"github.com/g-asiapacific/elm/backend/internal/workcal"
)

const maxReasonText = 1000

// Fields are the editable parts of a request. Everything is optional while drafting.
type Fields struct {
	// EmployeeName is the applicant name written on this request and its emails (defaults to the profile name).
	EmployeeName *string `json:"employee_name"`
	LeaveType    *string `json:"leave_type"`
	StartDate    *string `json:"start_date"` // YYYY-MM-DD
	EndDate      *string `json:"end_date"`
	StartTime    *string `json:"start_time"` // HH:mm
	EndTime      *string `json:"end_time"`
	ReasonCode   *string `json:"reason_code"`
	ReasonText   string  `json:"reason_text"`
}

// Issue is a machine-readable validation problem; the UI translates Code with Params.
type Issue struct {
	Code    string         `json:"code"`
	Field   string         `json:"field,omitempty"`
	Message string         `json:"message"`
	Params  map[string]any `json:"params,omitempty"`
}

func issue(code, field, msg string, params map[string]any) Issue {
	return Issue{Code: code, Field: field, Message: msg, Params: params}
}

func str(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

// parsed is the validated, typed form of Fields.
type parsed struct {
	leaveType  string
	interval   workcal.Interval
	haveRange  bool
	reasonCode string
	reasonText string
}

// lenientCheck validates formats and lengths only (used by auto-save, which must accept partial data).
func lenientCheck(f Fields, types map[string]settings.LeaveTypeRule) []Issue {
	var out []Issue
	if n := utf8.RuneCountInString(requestName(f.EmployeeName, "")); n > maxNameLen {
		out = append(out, issue("name_invalid", "employee_name", "Name must be 2-100 characters.", nil))
	}
	if t := str(f.LeaveType); t != "" && !hasType(types, t) {
		out = append(out, issue("type_invalid", "leave_type", "Choose a valid leave type.", nil))
	}
	for field, v := range map[string]*string{"start_date": f.StartDate, "end_date": f.EndDate} {
		if s := str(v); s != "" {
			if _, err := civil.Parse(s); err != nil {
				out = append(out, issue("date_invalid", field, "Enter a valid date.", nil))
			}
		}
	}
	for field, v := range map[string]*string{"start_time": f.StartTime, "end_time": f.EndTime} {
		if s := str(v); s != "" {
			if c, err := civil.ParseClock(s); err != nil || c > civil.EndOfDay {
				out = append(out, issue("time_invalid", field, "Enter a time as HH:mm.", nil))
			}
		}
	}
	if len(str(f.ReasonCode)) > 40 {
		out = append(out, issue("reason_invalid", "reason_code", "Choose a reason from the list.", nil))
	}
	if utf8.RuneCountInString(strings.TrimSpace(f.ReasonText)) > maxReasonText {
		out = append(out, issue("reason_too_long", "reason_text", fmt.Sprintf("Reason must be at most %d characters.", maxReasonText), map[string]any{"max": maxReasonText}))
	}
	return out
}

// checkInput carries everything the full policy check needs (loaded by the service).
type checkInput struct {
	fields      Fields
	policy      settings.Policy
	reasons     []settings.Reason
	rules       map[civil.Date]workcal.Rule
	unconfirmed []int // calendar years in range that are not CONFIRMED
	today       civil.Date
	profileName string // used when the request has no name of its own
}

func toWorkcal(p settings.Policy) workcal.Policy {
	return workcal.Policy{WorkStart: p.WorkStart, WorkEnd: p.WorkEnd, LunchStart: p.LunchStart, LunchEnd: p.LunchEnd, WorkdaysMask: p.WorkdaysMask}
}

// parseRange validates type, dates and times. It returns the interval when it is usable for calculation.
func parseRange(f Fields, types map[string]settings.LeaveTypeRule, unit int) (parsed, []Issue) {
	var out []Issue
	p := parsed{leaveType: str(f.LeaveType), reasonCode: str(f.ReasonCode), reasonText: strings.TrimSpace(f.ReasonText)}
	switch {
	case p.leaveType == "":
		out = append(out, issue("type_required", "leave_type", "Choose a leave type.", nil))
	case !hasType(types, p.leaveType):
		out = append(out, issue("type_invalid", "leave_type", "Choose a valid leave type.", nil))
	}

	date := func(field string, v *string) (civil.Date, bool) {
		s := str(v)
		if s == "" {
			out = append(out, issue("date_required", field, "Enter a date.", nil))
			return civil.Date{}, false
		}
		d, err := civil.Parse(s)
		if err != nil {
			out = append(out, issue("date_invalid", field, "Enter a valid date.", nil))
			return civil.Date{}, false
		}
		return d, true
	}
	clock := func(field string, v *string) (civil.Clock, bool) {
		s := str(v)
		if s == "" {
			out = append(out, issue("time_required", field, "Enter a time.", nil))
			return 0, false
		}
		c, err := civil.ParseClock(s)
		if err != nil || c > civil.EndOfDay {
			out = append(out, issue("time_invalid", field, "Enter a time as HH:mm.", nil))
			return 0, false
		}
		if unit > 0 && int(c)%unit != 0 {
			out = append(out, issue("time_step", field, fmt.Sprintf("Use %d-minute steps.", unit), map[string]any{"minutes": unit}))
			return 0, false
		}
		return c, true
	}
	sd, ok1 := date("start_date", f.StartDate)
	ed, ok2 := date("end_date", f.EndDate)
	st, ok3 := clock("start_time", f.StartTime)
	et, ok4 := clock("end_time", f.EndTime)
	if !(ok1 && ok2 && ok3 && ok4) {
		return p, out
	}
	switch {
	case ed.Before(sd):
		out = append(out, issue("end_before_start", "end_date", "The end date must be on or after the start date.", nil))
	case sd.DaysUntil(ed)+1 > workcal.MaxDays:
		out = append(out, issue("range_too_long", "end_date", fmt.Sprintf("A single request can cover at most %d days.", workcal.MaxDays), map[string]any{"days": workcal.MaxDays}))
	case sd == ed && et <= st:
		out = append(out, issue("time_order", "end_time", "The end time must be after the start time.", nil))
	default:
		p.interval = workcal.Interval{StartDate: sd, EndDate: ed, StartTime: st, EndTime: et}
		p.haveRange = true
	}
	return p, out
}

// evaluate runs every policy rule that does not need the database. Calculation is nil when the range is invalid.
func evaluate(in checkInput) (parsed, *workcal.Result, []Issue) {
	p, issues := parseRange(in.fields, in.policy.LeaveTypes, in.policy.LeaveUnitMinutes)
	issues = append(issues, reasonIssues(p, in)...)
	if n := utf8.RuneCountInString(requestName(in.fields.EmployeeName, in.profileName)); n < 2 || n > maxNameLen {
		issues = append(issues, issue("name_invalid", "employee_name", "Name must be 2-100 characters.", nil))
	}
	if !p.haveRange {
		return p, nil, issues
	}

	if len(in.unconfirmed) > 0 {
		years := make([]string, len(in.unconfirmed))
		for i, y := range in.unconfirmed {
			years[i] = fmt.Sprint(y)
		}
		issues = append(issues, issue("calendar_not_confirmed", "start_date",
			"The company holiday calendar for "+strings.Join(years, ", ")+" has not been confirmed yet, so this request cannot be submitted.",
			map[string]any{"years": strings.Join(years, ", ")}))
	}

	res := workcal.Calculate(toWorkcal(in.policy), in.rules, p.interval)
	switch {
	case res.TotalMinutes == 0:
		issues = append(issues, issue("zero_duration", "start_date", "The selected time contains no working hours.", nil))
	case res.TotalMinutes < in.policy.MinLeaveMinutes:
		issues = append(issues, issue("below_minimum", "end_time", fmt.Sprintf("The minimum leave is %d minutes.", in.policy.MinLeaveMinutes), map[string]any{"minutes": in.policy.MinLeaveMinutes}))
	}

	if rule, ok := in.policy.LeaveTypes[p.leaveType]; ok {
		start := p.interval.StartDate
		if start.Before(in.today) {
			back := start.DaysUntil(in.today)
			switch {
			case !rule.AllowPastDates:
				issues = append(issues, issue("past_not_allowed", "start_date", "This leave type cannot start in the past.", nil))
			case back > rule.MaxPastDays:
				issues = append(issues, issue("past_too_far", "start_date", fmt.Sprintf("This leave type can start at most %d day(s) in the past.", rule.MaxPastDays), map[string]any{"days": rule.MaxPastDays}))
			}
		} else if ahead := in.today.DaysUntil(start); ahead < rule.AdvanceNoticeDays {
			issues = append(issues, issue("advance_notice", "start_date", fmt.Sprintf("This leave type must be requested at least %d day(s) in advance.", rule.AdvanceNoticeDays), map[string]any{"days": rule.AdvanceNoticeDays}))
		}
	}
	return p, &res, issues
}

func reasonIssues(p parsed, in checkInput) []Issue {
	var out []Issue
	rule := in.policy.LeaveTypes[p.leaveType]
	var reason *settings.Reason
	for i := range in.reasons {
		if in.reasons[i].Code == p.reasonCode && in.reasons[i].Active {
			reason = &in.reasons[i]
		}
	}
	switch {
	case p.reasonCode == "" && rule.ReasonRequired:
		out = append(out, issue("reason_required", "reason_code", "Choose a reason.", nil))
	case p.reasonCode != "" && reason == nil:
		out = append(out, issue("reason_invalid", "reason_code", "Choose a reason from the list.", nil))
	case reason != nil && reason.RequiresText && p.reasonText == "":
		out = append(out, issue("reason_text_required", "reason_text", "Describe the reason.", nil))
	}
	if utf8.RuneCountInString(p.reasonText) > maxReasonText {
		out = append(out, issue("reason_too_long", "reason_text", fmt.Sprintf("Reason must be at most %d characters.", maxReasonText), map[string]any{"max": maxReasonText}))
	}
	return out
}

// Snapshot is stored with each submission so history never depends on later policy or calendar changes.
type Snapshot struct {
	PolicyVersionID uint64             `json:"policy_version_id"`
	PolicyVersionNo uint32             `json:"policy_version_no"`
	WorkStart       string             `json:"work_start"`
	WorkEnd         string             `json:"work_end"`
	LunchStart      string             `json:"lunch_start"`
	LunchEnd        string             `json:"lunch_end"`
	MinutesPerDay   int                `json:"minutes_per_day"`
	TotalMinutes    int                `json:"total_minutes"`
	Days            []workcal.Day      `json:"days"`
	Excluded        []workcal.Excluded `json:"excluded"`
}

func snapshot(p settings.Policy, r workcal.Result) Snapshot {
	return Snapshot{
		PolicyVersionID: p.VersionID, PolicyVersionNo: p.VersionNo,
		WorkStart: p.WorkStart.String(), WorkEnd: p.WorkEnd.String(), LunchStart: p.LunchStart.String(), LunchEnd: p.LunchEnd.String(),
		MinutesPerDay: p.MinutesPerDay, TotalMinutes: r.TotalMinutes, Days: r.Days, Excluded: r.Excluded,
	}
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

const maxNameLen = 100

// requestName is the applicant name of a request: its own (spaces normalized) or, when empty, the profile name.
func requestName(own *string, profile string) string {
	if n := strings.Join(strings.Fields(str(own)), " "); n != "" {
		return n
	}
	return strings.Join(strings.Fields(profile), " ")
}

// hasType reports whether code is an active leave type of the policy.
func hasType(types map[string]settings.LeaveTypeRule, code string) bool {
	_, ok := types[code]
	return ok
}
