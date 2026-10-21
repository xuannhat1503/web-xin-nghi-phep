package leave

import (
	"context"
	"database/sql"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

// typeCode is the shape of a leave type code (filter values are matched against stored codes).
var typeCode = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,39}$`)

// ListPageSize is fixed by the business rules: 10 requests per page.
const ListPageSize = 10

var statuses = []string{"DRAFT", "PENDING", "NEEDS_CHANGES", "APPROVED", "REJECTED", "CANCELLED"}

// ListFilter holds validated list filters. Invalid values are ignored (treated as "not set").
type ListFilter struct {
	Year, Month                      int
	Status, LeaveType, Code, Keyword string
	From, To                         *civil.Date // leave dates overlapping this range
	SubmittedFrom, SubmittedTo       *civil.Date
	ReviewedFrom, ReviewedTo         *civil.Date
	Sort                             string // newest | oldest | start
}

func dateQuery(c *gin.Context, key string) *civil.Date {
	d, err := civil.Parse(c.Query(key))
	if err != nil {
		return nil
	}
	return &d
}

func trimTo(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		return string(r[:n])
	}
	return s
}

func ParseListFilter(c *gin.Context) ListFilter {
	f := ListFilter{
		Code: trimTo(c.Query("code"), 30), Keyword: trimTo(c.Query("q"), 100), Sort: c.Query("sort"),
		From: dateQuery(c, "from"), To: dateQuery(c, "to"),
		SubmittedFrom: dateQuery(c, "submitted_from"), SubmittedTo: dateQuery(c, "submitted_to"),
		ReviewedFrom: dateQuery(c, "reviewed_from"), ReviewedTo: dateQuery(c, "reviewed_to"),
	}
	if y, err := strconv.Atoi(c.Query("year")); err == nil && y >= 2000 && y <= 2200 {
		f.Year = y
	}
	if m, err := strconv.Atoi(c.Query("month")); err == nil && m >= 1 && m <= 12 && f.Year > 0 {
		f.Month = m
	}
	if s := c.Query("status"); slices.Contains(statuses, s) {
		f.Status = s
	}
	if t := c.Query("leave_type"); typeCode.MatchString(t) {
		f.LeaveType = t
	}
	if f.Sort != "oldest" && f.Sort != "start" {
		f.Sort = "newest"
	}
	return f
}

var vnZone = func() *time.Location {
	l, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*3600)
	}
	return l
}()

// startOfDayUTC converts a company-local calendar date to the UTC instant of its midnight.
func startOfDayUTC(d civil.Date) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, vnZone).UTC()
}

// where builds the parameterized filter clause. The scope (authorization) is always ANDed first.
func (f ListFilter) where(scope string, scopeArgs []any) (string, []any) {
	parts := []string{"(" + scope + ")"}
	args := append([]any{}, scopeArgs...)

	// Leave-date range: month/year intersected with the explicit range. Overlap, so a request spanning
	// two months appears in both.
	from, to := f.From, f.To
	if f.Year > 0 {
		ys, ye := civil.D(f.Year, time.January, 1), civil.D(f.Year, time.December, 31)
		if f.Month > 0 {
			ys = civil.D(f.Year, time.Month(f.Month), 1)
			ye = ys.AddDays(32)
			ye = civil.D(ye.Year, ye.Month, 1).AddDays(-1)
		}
		if from == nil || from.Before(ys) {
			from = &ys
		}
		if to == nil || to.After(ye) {
			to = &ye
		}
	}
	if from != nil {
		parts, args = append(parts, "lr.end_date >= ?"), append(args, from.UTC())
	}
	if to != nil {
		parts, args = append(parts, "lr.start_date <= ?"), append(args, to.UTC())
	}
	if f.Status != "" {
		parts, args = append(parts, "lr.status = ?"), append(args, f.Status)
	}
	if f.LeaveType != "" {
		parts, args = append(parts, "lr.leave_type = ?"), append(args, f.LeaveType)
	}
	if f.Code != "" {
		parts, args = append(parts, "lr.request_code LIKE ?"), append(args, httpx.LikePattern(strings.ToUpper(f.Code)))
	}
	if f.Keyword != "" {
		parts, args = append(parts, "lr.reason_text LIKE ?"), append(args, httpx.LikePattern(f.Keyword))
	}
	if f.SubmittedFrom != nil {
		parts, args = append(parts, "lr.submitted_at >= ?"), append(args, startOfDayUTC(*f.SubmittedFrom))
	}
	if f.SubmittedTo != nil {
		parts, args = append(parts, "lr.submitted_at < ?"), append(args, startOfDayUTC(f.SubmittedTo.AddDays(1)))
	}
	if f.ReviewedFrom != nil {
		parts, args = append(parts, "lr.reviewed_at >= ?"), append(args, startOfDayUTC(*f.ReviewedFrom))
	}
	if f.ReviewedTo != nil {
		parts, args = append(parts, "lr.reviewed_at < ?"), append(args, startOfDayUTC(f.ReviewedTo.AddDays(1)))
	}
	return strings.Join(parts, " AND "), args
}

func (f ListFilter) orderBy() string {
	switch f.Sort {
	case "oldest":
		return "lr.created_at ASC, lr.id ASC"
	case "start":
		return "lr.start_date IS NULL, lr.start_date ASC, lr.id ASC"
	default:
		return "lr.created_at DESC, lr.id DESC"
	}
}

type ListItem struct {
	ID           uint64     `json:"id"`
	Code         *string    `json:"request_code"`
	Status       string     `json:"status"`
	LeaveType    *string    `json:"leave_type"`
	StartDate    *string    `json:"start_date"`
	EndDate      *string    `json:"end_date"`
	StartTime    *string    `json:"start_time"`
	EndTime      *string    `json:"end_time"`
	TotalMinutes uint32     `json:"total_minutes"`
	EmployeeName string     `json:"employee_name"`
	Reviewer     *string    `json:"reviewer_name"`
	CreatedAt    time.Time  `json:"created_at"`
	SubmittedAt  *time.Time `json:"submitted_at"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
}

// ListRequests runs the count and page queries with the same scope and filters (no count leakage).
func ListRequests(ctx context.Context, db *sql.DB, scope string, scopeArgs []any, f ListFilter, p httpx.Page) (httpx.Paged[ListItem], error) {
	where, args := f.where(scope, scopeArgs)
	from := " FROM leave_requests lr JOIN employees e ON e.id = lr.employee_id WHERE " + where
	var total int64
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*)"+from, args...).Scan(&total); err != nil {
		return httpx.Paged[ListItem]{}, err
	}
	rows, err := db.QueryContext(ctx,
		`SELECT lr.id, lr.request_code, lr.status, lr.leave_type, lr.start_date, lr.end_date, lr.start_time, lr.end_time,
		        lr.total_minutes, e.full_name, lr.manager_name_snapshot, lr.created_at, lr.submitted_at, lr.reviewed_at`+
			from+" ORDER BY "+f.orderBy()+" LIMIT ? OFFSET ?", append(args, p.Limit(), p.Offset())...)
	if err != nil {
		return httpx.Paged[ListItem]{}, err
	}
	defer rows.Close()
	var items []ListItem
	for rows.Next() {
		var it ListItem
		var code, lt, st, et, reviewer sql.NullString
		var sd, ed, sub, rev sql.NullTime
		if err := rows.Scan(&it.ID, &code, &it.Status, &lt, &sd, &ed, &st, &et, &it.TotalMinutes, &it.EmployeeName,
			&reviewer, &it.CreatedAt, &sub, &rev); err != nil {
			return httpx.Paged[ListItem]{}, err
		}
		it.Code, it.LeaveType, it.Reviewer = nstr(code), nstr(lt), nstr(reviewer)
		it.StartDate, it.EndDate, it.StartTime, it.EndTime = ndate(sd), ndate(ed), nclock(st), nclock(et)
		it.SubmittedAt, it.ReviewedAt = ntime(sub), ntime(rev)
		items = append(items, it)
	}
	return httpx.NewPaged(items, p, total), rows.Err()
}

// StatusCounts counts requests per status within a scope.
func StatusCounts(ctx context.Context, db *sql.DB, scope string, args ...any) (map[string]int64, error) {
	out := map[string]int64{}
	for _, s := range statuses {
		out[s] = 0
	}
	rows, err := db.QueryContext(ctx, "SELECT lr.status, COUNT(*) FROM leave_requests lr WHERE "+scope+" GROUP BY lr.status", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var s string
		var n int64
		if err := rows.Scan(&s, &n); err != nil {
			return nil, err
		}
		out[s] = n
	}
	return out, rows.Err()
}

const ownerScope = "lr.employee_id = ?"

func (s *Service) List(ctx context.Context, employeeID uint64, f ListFilter, p httpx.Page) (httpx.Paged[ListItem], error) {
	return ListRequests(ctx, s.db, ownerScope, []any{employeeID}, f, p)
}

// Summary is the employee dashboard: own counts and the latest requests.
func (s *Service) Summary(ctx context.Context, employeeID uint64) (map[string]any, error) {
	counts, err := StatusCounts(ctx, s.db, ownerScope, employeeID)
	if err != nil {
		return nil, err
	}
	recent, err := ListRequests(ctx, s.db, ownerScope+" AND lr.status <> 'DRAFT'", []any{employeeID}, ListFilter{Sort: "newest"}, httpx.Page{Number: 1, Size: 5})
	if err != nil {
		return nil, err
	}
	return map[string]any{"counts": counts, "recent": recent.Items}, nil
}
