// Package analytics serves ADMIN-only aggregate figures. Responses never contain names, emails,
// reasons or any per-person leave data.
package analytics

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/settings"
)

// The company calendar is Asia/Ho_Chi_Minh (UTC+7, no DST); timestamps are stored in UTC.
const tzOffset = "+07:00"

var vn = time.FixedZone("ICT", 7*3600)

type Handlers struct {
	db       *sql.DB
	settings *settings.Service
	log      *slog.Logger
}

func NewHandlers(db *sql.DB, st *settings.Service, log *slog.Logger) *Handlers {
	return &Handlers{db: db, settings: st, log: log}
}

func (h *Handlers) RegisterAdmin(g gin.IRouter) { g.GET("/analytics", h.get) }

type KPIs struct {
	Total              int64            `json:"total"`
	ByStatus           map[string]int64 `json:"by_status"`
	ApprovedMinutes    int64            `json:"approved_minutes"`
	AvgProcessingHours *float64         `json:"avg_processing_hours"`
	OverduePending     int64            `json:"overdue_pending"`
	InactiveEmployees  int64            `json:"inactive_employees"`
	WithoutManager     int64            `json:"employees_without_manager"`
	UnconfirmedYears   []int            `json:"unconfirmed_years"`
}

type TypeMinutes struct {
	Month     int    `json:"month"`
	LeaveType string `json:"leave_type"`
	Minutes   int64  `json:"minutes"`
}

type Result struct {
	Year           int           `json:"year"`
	Month          int           `json:"month"` // 0 = whole year
	MinutesPerDay  int           `json:"minutes_per_day"`
	KPIs           KPIs          `json:"kpis"`
	MonthlyCounts  []int64       `json:"monthly_counts"` // 12 entries, by first submission month
	ApprovedByType []TypeMinutes `json:"approved_by_month_type"`
}

func (h *Handlers) get(c *gin.Context) {
	now := time.Now().In(vn)
	year, month := now.Year(), 0
	if y, err := strconv.Atoi(c.Query("year")); err == nil && y >= 2000 && y <= 2200 {
		year = y
	}
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 {
		month = m
	}
	res, err := h.compute(c.Request.Context(), year, month)
	if err != nil {
		httpx.Fail(c, h.log, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *Handlers) compute(ctx context.Context, year, month int) (Result, error) {
	res := Result{Year: year, Month: month, MonthlyCounts: make([]int64, 12), ApprovedByType: []TypeMinutes{},
		KPIs: KPIs{ByStatus: map[string]int64{"PENDING": 0, "NEEDS_CHANGES": 0, "APPROVED": 0, "REJECTED": 0, "CANCELLED": 0}, UnconfirmedYears: []int{}}}
	pol, err := h.settings.Current(ctx)
	if err != nil {
		return res, err
	}
	res.MinutesPerDay = pol.MinutesPerDay

	// Period in company time -> UTC instants (for timestamps) and calendar dates (for leave days).
	startLocal := time.Date(year, time.January, 1, 0, 0, 0, 0, vn)
	endLocal := startLocal.AddDate(1, 0, 0)
	if month > 0 {
		startLocal = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, vn)
		endLocal = startLocal.AddDate(0, 1, 0)
	}
	from, to := startLocal.UTC(), endLocal.UTC()
	dFrom := time.Date(startLocal.Year(), startLocal.Month(), startLocal.Day(), 0, 0, 0, 0, time.UTC)
	dTo := time.Date(endLocal.Year(), endLocal.Month(), endLocal.Day(), 0, 0, 0, 0, time.UTC)

	// Requests counted once by their first submission (resubmissions are not new requests); current status.
	rows, err := h.db.QueryContext(ctx,
		`SELECT status, COUNT(*) FROM leave_requests WHERE first_submitted_at >= ? AND first_submitted_at < ? GROUP BY status`, from, to)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			rows.Close()
			return res, err
		}
		res.KPIs.ByStatus[s] = n
		res.KPIs.Total += n
	}
	rows.Close()

	// Approved time is allocated to the month of each leave day (cross-month requests split correctly),
	// using the minutes stored at submission (never recalculated).
	if err := h.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(d.minutes), 0) FROM leave_request_days d JOIN leave_requests lr ON lr.id = d.leave_request_id
		 WHERE lr.status = 'APPROVED' AND d.work_date >= ? AND d.work_date < ?`, dFrom, dTo).Scan(&res.KPIs.ApprovedMinutes); err != nil {
		return res, err
	}

	var avg sql.NullFloat64
	if err := h.db.QueryRowContext(ctx,
		`SELECT AVG(TIMESTAMPDIFF(MINUTE, first_submitted_at, reviewed_at)) / 60 FROM leave_requests
		 WHERE status IN ('APPROVED', 'REJECTED') AND reviewed_at >= ? AND reviewed_at < ?`, from, to).Scan(&avg); err != nil {
		return res, err
	}
	if avg.Valid {
		v := float64(int(avg.Float64*10+0.5)) / 10
		res.KPIs.AvgProcessingHours = &v
	}

	// Overdue: pending longer than the reminder interval (clock time, current state).
	if err := h.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM leave_requests WHERE status = 'PENDING' AND submitted_at < UTC_TIMESTAMP(3) - INTERVAL ? MINUTE`,
		pol.ReminderIntervalMinutes).Scan(&res.KPIs.OverduePending); err != nil {
		return res, err
	}
	if err := h.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(active = FALSE), 0), COALESCE(SUM(active = TRUE AND manager_id IS NULL), 0) FROM employees`).
		Scan(&res.KPIs.InactiveEmployees, &res.KPIs.WithoutManager); err != nil {
		return res, err
	}
	thisYear := time.Now().In(vn).Year()
	for _, y := range []int{thisYear, thisYear + 1} {
		var st sql.NullString
		err := h.db.QueryRowContext(ctx, "SELECT status FROM calendar_years WHERE year = ?", y).Scan(&st)
		if err != nil && err != sql.ErrNoRows {
			return res, err
		}
		if st.String != "CONFIRMED" {
			res.KPIs.UnconfirmedYears = append(res.KPIs.UnconfirmedYears, y)
		}
	}

	// Chart 1: requests per month of the year (first submission, company time).
	yFrom := time.Date(year, time.January, 1, 0, 0, 0, 0, vn).UTC()
	yTo := time.Date(year+1, time.January, 1, 0, 0, 0, 0, vn).UTC()
	rows, err = h.db.QueryContext(ctx,
		`SELECT MONTH(CONVERT_TZ(first_submitted_at, '+00:00', ?)) AS m, COUNT(*) FROM leave_requests
		 WHERE first_submitted_at >= ? AND first_submitted_at < ? GROUP BY m`, tzOffset, yFrom, yTo)
	if err != nil {
		return res, err
	}
	for rows.Next() {
		var m int
		var n int64
		if err := rows.Scan(&m, &n); err != nil {
			rows.Close()
			return res, err
		}
		if m >= 1 && m <= 12 {
			res.MonthlyCounts[m-1] = n
		}
	}
	rows.Close()

	// Chart 3: approved minutes per month and leave type (by leave day).
	rows, err = h.db.QueryContext(ctx,
		`SELECT MONTH(d.work_date) AS m, lr.leave_type, SUM(d.minutes) FROM leave_request_days d
		 JOIN leave_requests lr ON lr.id = d.leave_request_id
		 WHERE lr.status = 'APPROVED' AND d.work_date >= ? AND d.work_date < ?
		 GROUP BY m, lr.leave_type ORDER BY m, lr.leave_type`,
		time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		return res, err
	}
	defer rows.Close()
	for rows.Next() {
		var t TypeMinutes
		if err := rows.Scan(&t.Month, &t.LeaveType, &t.Minutes); err != nil {
			return res, err
		}
		res.ApprovedByType = append(res.ApprovedByType, t)
	}
	return res, rows.Err()
}
