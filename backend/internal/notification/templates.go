// Package notification renders notifications and Mock Email content. Emails are never delivered.
package notification

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/g-asiapacific/elm/backend/internal/civil"
)

// LeaveEmail holds what the company's leave email format (mailsample.txt) requires.
type LeaveEmail struct {
	EmployeeName  string
	ManagerName   string
	ManagerEmail  string
	CC            []string
	LeaveType     string // English display name of the leave type
	StartDate     civil.Date
	EndDate       civil.Date
	StartTime     civil.Clock
	EndTime       civil.Clock
	TotalMinutes  int
	MinutesPerDay int
	Reason        string
	ReviewLink    string
}

type Rendered struct {
	To      string   `json:"to"`
	CC      []string `json:"cc"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
}

func ddmmyyyy(d civil.Date) string { return fmt.Sprintf("%02d/%02d/%04d", d.Day, int(d.Month), d.Year) }

// LeaveDates formats "dd/mm/yyyy" or "dd/mm/yyyy - dd/mm/yyyy".
func LeaveDates(start, end civil.Date) string {
	if start == end {
		return ddmmyyyy(start)
	}
	return ddmmyyyy(start) + " - " + ddmmyyyy(end)
}

func trimNum(f float64) string {
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// FormatDuration renders e.g. "16 Hrs (2 Days)" or "4 Hrs (0.5 Day)".
func FormatDuration(minutes, minutesPerDay int) string {
	hours := float64(minutes) / 60
	h := fmt.Sprintf("%s Hrs", trimNum(float64(int(hours*100+0.5))/100))
	if minutesPerDay <= 0 {
		return h
	}
	days := float64(int(float64(minutes)/float64(minutesPerDay)*100+0.5)) / 100
	unit := "Days"
	if days <= 1 {
		unit = "Day"
	}
	return fmt.Sprintf("%s (%s %s)", h, trimNum(days), unit)
}

// ApprovalRequired renders the "Apply for Leave" email sent to the direct manager on submission.
func ApprovalRequired(e LeaveEmail) Rendered {
	subject := fmt.Sprintf("Apply for Leave_%s_%s", e.EmployeeName, LeaveDates(e.StartDate, e.EndDate))
	var b strings.Builder
	fmt.Fprintf(&b, "Dear %s,\n\n", e.ManagerName)
	fmt.Fprintf(&b, "%s has submitted a leave application for your review.\n\n", e.EmployeeName)
	fmt.Fprintf(&b, "Kind of leave: %s\n", e.LeaveType)
	fmt.Fprintf(&b, "Leave from date: %s\n", ddmmyyyy(e.StartDate))
	fmt.Fprintf(&b, "Leave to date: %s\n", ddmmyyyy(e.EndDate))
	fmt.Fprintf(&b, "Time-off: from %s to %s\n", e.StartTime, e.EndTime)
	fmt.Fprintf(&b, "Total requested time-off: %s\n", FormatDuration(e.TotalMinutes, e.MinutesPerDay))
	fmt.Fprintf(&b, "Leave Reason: %s\n\n", e.Reason)
	fmt.Fprintf(&b, "Review Leave Request: %s\n\n", e.ReviewLink)
	b.WriteString("Please review and approve or reject the request in the leave system.\n\n")
	cc := e.CC
	if cc == nil {
		cc = []string{}
	}
	return Rendered{To: e.ManagerEmail, CC: cc, Subject: subject, Body: b.String()}
}

var statusSubjects = map[string]string{
	"APPROVED":            "Leave Approved_%s_%s",
	"REJECTED":            "Leave Rejected_%s_%s",
	"CHANGES_REQUESTED":   "Leave Needs Changes_%s_%s",
	"RESUBMITTED":         "Apply for Leave (Resubmitted)_%s_%s",
	"CANCELLED":           "Leave Cancelled_%s_%s",
	"REVIEWER_REASSIGNED": "Apply for Leave (Reassigned to you)_%s_%s",
}

var statusIntros = map[string]string{
	"APPROVED":            "Your leave application has been approved.",
	"REJECTED":            "Your leave application has been rejected.",
	"CHANGES_REQUESTED":   "Your manager has requested changes to your leave application. Please update and resubmit it.",
	"RESUBMITTED":         "%s has updated and resubmitted a leave application for your review.",
	"CANCELLED":           "%s has cancelled a leave application that was assigned to you.",
	"REVIEWER_REASSIGNED": "A leave application from %s has been assigned to you for review.",
}

// StatusEmail renders the follow-up emails (approved, rejected, changes requested, resubmitted, cancelled,
// reassigned). recipientName greets the reader; comment is the reviewer's reason when there is one.
func StatusEmail(emailType, recipientName string, e LeaveEmail, comment string) Rendered {
	subject := fmt.Sprintf(statusSubjects[emailType], e.EmployeeName, LeaveDates(e.StartDate, e.EndDate))
	intro := statusIntros[emailType]
	if strings.Contains(intro, "%s") {
		intro = fmt.Sprintf(intro, e.EmployeeName)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Dear %s,\n\n%s\n\n", recipientName, intro)
	fmt.Fprintf(&b, "Kind of leave: %s\n", e.LeaveType)
	fmt.Fprintf(&b, "Leave from date: %s\n", ddmmyyyy(e.StartDate))
	fmt.Fprintf(&b, "Leave to date: %s\n", ddmmyyyy(e.EndDate))
	fmt.Fprintf(&b, "Time-off: from %s to %s\n", e.StartTime, e.EndTime)
	fmt.Fprintf(&b, "Total requested time-off: %s\n", FormatDuration(e.TotalMinutes, e.MinutesPerDay))
	fmt.Fprintf(&b, "Leave Reason: %s\n", e.Reason)
	if comment != "" {
		fmt.Fprintf(&b, "Reviewer comment: %s\n", comment)
	}
	fmt.Fprintf(&b, "\nView Leave Request: %s\n\n", e.ReviewLink)
	cc := e.CC
	if cc == nil {
		cc = []string{}
	}
	return Rendered{To: e.ManagerEmail, CC: cc, Subject: subject, Body: b.String()}
}
