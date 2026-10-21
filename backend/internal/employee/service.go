// Package employee implements Admin employee management: activation, roles and manager assignment.
package employee

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/dbtypes"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

const (
	PageSize        = 20
	maxReasonLength = 500
	maxChainDepth   = 100
	orgChartLock    = "elm:org_chart"
)

var (
	ErrLastAdmin     = httpx.E(http.StatusConflict, "last_admin", "At least one active administrator must remain.")
	ErrSelfAccess    = httpx.E(http.StatusConflict, "self_change", "You cannot deactivate or demote your own account.")
	ErrSelfManager   = httpx.E(http.StatusUnprocessableEntity, "self_manager", "An employee cannot be their own manager.")
	ErrManagerCycle  = httpx.E(http.StatusUnprocessableEntity, "manager_cycle", "This assignment would create a reporting cycle.")
	ErrManagerActive = httpx.E(http.StatusUnprocessableEntity, "manager_inactive", "The manager must be an active employee.")
)

type Service struct {
	db *sql.DB
	q  *store.Queries
}

func NewService(db *sql.DB) *Service { return &Service{db: db, q: store.New(db)} }

type ListFilter struct {
	Search    string
	Status    string // all | active | inactive
	NoManager bool
}

type ListItem struct {
	ID            uint64     `json:"id"`
	FullName      string     `json:"full_name"`
	Email         string     `json:"email"`
	Role          string     `json:"role"`
	Active        bool       `json:"active"`
	Manager       *PersonRef `json:"manager"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	CreatedAt     time.Time  `json:"created_at"`
	Version       uint32     `json:"version"`
	ManagerActive *bool      `json:"manager_active,omitempty"`
}

type PersonRef struct {
	ID       uint64 `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email,omitempty"`
}

type Summary struct {
	Inactive             int64 `json:"inactive"`
	ActiveWithoutManager int64 `json:"active_without_manager"`
	Total                int64 `json:"total"`
}

type ListResult struct {
	httpx.Paged[ListItem]
	Summary Summary `json:"summary"`
}

func (s *Service) List(ctx context.Context, f ListFilter, p httpx.Page) (ListResult, error) {
	f.Search = strings.TrimSpace(f.Search)
	if len(f.Search) > 100 {
		f.Search = f.Search[:100]
	}
	switch f.Status {
	case "active", "inactive":
	default:
		f.Status = "all"
	}
	pattern := httpx.LikePattern(f.Search)

	total, err := s.q.CountEmployeesAdmin(ctx, store.CountEmployeesAdminParams{
		Search: f.Search, Pattern: pattern, Status: f.Status, NoManager: f.NoManager,
	})
	if err != nil {
		return ListResult{}, err
	}
	rows, err := s.q.ListEmployeesAdmin(ctx, store.ListEmployeesAdminParams{
		Search: f.Search, Pattern: pattern, Status: f.Status, NoManager: f.NoManager,
		Limit: p.Limit(), Offset: p.Offset(),
	})
	if err != nil {
		return ListResult{}, err
	}
	sum, err := s.q.EmployeeSummaryCounts(ctx)
	if err != nil {
		return ListResult{}, err
	}
	items := make([]ListItem, 0, len(rows))
	for _, r := range rows {
		it := ListItem{ID: r.ID, FullName: r.FullName, Email: r.Email, Role: string(r.Role), Active: r.Active,
			CreatedAt: r.CreatedAt, Version: r.Version, LastLoginAt: nullTime(r.LastLoginAt)}
		if r.ManagerID.Valid {
			it.Manager = &PersonRef{ID: uint64(r.ManagerID.Int64), FullName: r.ManagerName.String}
			it.ManagerActive = &r.ManagerActive.Bool
		}
		items = append(items, it)
	}
	return ListResult{
		Paged:   httpx.NewPaged(items, p, total),
		Summary: Summary{Inactive: sum.Inactive, ActiveWithoutManager: sum.ActiveWithoutManager, Total: sum.Total},
	}, nil
}

type HistoryEntry struct {
	Kind      string       `json:"kind"` // account | manager
	Action    string       `json:"action"`
	Actor     string       `json:"actor"`
	Before    dbtypes.JSON `json:"before,omitempty"`
	After     dbtypes.JSON `json:"after,omitempty"`
	Reason    string       `json:"reason,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

type Detail struct {
	ListItem
	UpdatedAt     time.Time      `json:"updated_at"`
	ActiveReports int64          `json:"active_reports"`
	History       []HistoryEntry `json:"history"`
}

func (s *Service) Get(ctx context.Context, id uint64) (Detail, error) {
	r, err := s.q.GetEmployeeAdmin(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return Detail{}, httpx.ErrNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	d := Detail{
		ListItem: ListItem{ID: r.ID, FullName: r.FullName, Email: r.Email, Role: string(r.Role), Active: r.Active,
			CreatedAt: r.CreatedAt, Version: r.Version, LastLoginAt: nullTime(r.LastLoginAt)},
		UpdatedAt: r.UpdatedAt, ActiveReports: r.ActiveReports,
	}
	if r.ManagerID.Valid {
		d.Manager = &PersonRef{ID: uint64(r.ManagerID.Int64), FullName: r.ManagerName.String, Email: r.ManagerEmail.String}
		d.ManagerActive = &r.ManagerActive.Bool
	}

	events, err := s.q.ListEmployeeEvents(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	mgr, err := s.q.ListManagerEvents(ctx, id)
	if err != nil {
		return Detail{}, err
	}
	for _, e := range events {
		d.History = append(d.History, HistoryEntry{Kind: "account", Action: e.Action, Actor: actor(e.ActorName),
			Before: e.BeforeData, After: e.AfterData, CreatedAt: e.CreatedAt})
	}
	for _, e := range mgr {
		after, _ := json.Marshal(map[string]any{"from": nullStr(e.OldManagerName), "to": nullStr(e.NewManagerName)})
		d.History = append(d.History, HistoryEntry{Kind: "manager", Action: "MANAGER_CHANGED", Actor: actor(e.ActorName),
			After: after, Reason: e.Reason, CreatedAt: e.CreatedAt})
	}
	sortHistory(d.History)
	return d, nil
}

type AccessChange struct {
	Version uint32  `json:"version"`
	Active  *bool   `json:"active"`
	Role    *string `json:"role"`
}

// UpdateAccess activates/deactivates or changes the role of an employee.
func (s *Service) UpdateAccess(ctx context.Context, actorID, id uint64, ch AccessChange) error {
	if ch.Role != nil && *ch.Role != "ADMIN" && *ch.Role != "EMPLOYEE" {
		v := httpx.Validation{}
		v.Add("role", "Role must be ADMIN or EMPLOYEE.")
		return v.Err()
	}
	return database.InTx(ctx, s.db, func(tx *sql.Tx) error { return updateAccess(ctx, s.q.WithTx(tx), actorID, id, ch) })
}

func updateAccess(ctx context.Context, q *store.Queries, actorID, id uint64, ch AccessChange) error {
	// Lock admins first (same order for every caller) so last-admin checks cannot race.
	admins, err := q.LockActiveAdmins(ctx)
	if err != nil {
		return err
	}
	cur, err := q.GetEmployeeForUpdate(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if cur.Version != ch.Version {
		return httpx.ErrConflict
	}

	active, role := cur.Active, cur.Role
	if ch.Active != nil {
		active = *ch.Active
	}
	if ch.Role != nil {
		role = store.EmployeesRole(*ch.Role)
	}
	if role == store.EmployeesRoleADMIN && cur.Role != store.EmployeesRoleADMIN {
		active = true // promoting always activates (mirrors the DB trigger)
	}
	if active == cur.Active && role == cur.Role {
		return nil
	}

	losingAdmin := cur.Role == store.EmployeesRoleADMIN && cur.Active && (role != store.EmployeesRoleADMIN || !active)
	if losingAdmin {
		if id == actorID {
			return ErrSelfAccess
		}
		if len(admins) <= 1 {
			return ErrLastAdmin
		}
	}
	if id == actorID && !active {
		return ErrSelfAccess
	}

	n, err := q.UpdateEmployeeAccess(ctx, store.UpdateEmployeeAccessParams{Active: active, Role: role, ID: id, Version: ch.Version})
	if err != nil {
		return err
	}
	if n != 1 {
		return httpx.ErrConflict
	}
	if cur.Active && !active {
		// Deactivated users lose access immediately, everywhere.
		if _, err := q.RevokeEmployeeSessions(ctx, store.RevokeEmployeeSessionsParams{
			Now: sql.NullTime{Time: time.Now().UTC(), Valid: true}, EmployeeID: id,
		}); err != nil {
			return err
		}
	}
	action := "ACCESS_CHANGED"
	switch {
	case !cur.Active && active && role == cur.Role:
		action = "ACTIVATED"
	case cur.Active && !active && role == cur.Role:
		action = "DEACTIVATED"
	case role != cur.Role:
		action = "ROLE_CHANGED"
	}
	before, _ := json.Marshal(map[string]any{"active": cur.Active, "role": cur.Role})
	after, _ := json.Marshal(map[string]any{"active": active, "role": role})
	return q.InsertEmployeeEvent(ctx, store.InsertEmployeeEventParams{
		EmployeeID: id, ActorID: sql.NullInt64{Int64: int64(actorID), Valid: true},
		Action: action, BeforeData: before, AfterData: after,
	})
}

type ManagerChange struct {
	Version   uint32  `json:"version"`
	ManagerID *uint64 `json:"manager_id"` // null removes the manager
	Reason    string  `json:"reason"`
}

// AssignManager sets or clears an employee's direct manager. Submitted leave requests keep their reviewer.
func (s *Service) AssignManager(ctx context.Context, actorID, id uint64, ch ManagerChange) error {
	ch.Reason = strings.TrimSpace(ch.Reason)
	if len([]rune(ch.Reason)) > maxReasonLength {
		v := httpx.Validation{}
		v.Add("reason", fmt.Sprintf("Reason must be at most %d characters.", maxReasonLength))
		return v.Err()
	}
	if ch.ManagerID != nil && *ch.ManagerID == id {
		return ErrSelfManager
	}

	// The named lock serializes all manager changes, so a cycle check cannot race with another assignment.
	err := database.WithNamedLock(ctx, s.db, orgChartLock, 5*time.Second, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.GetEmployeeForUpdate(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.Version != ch.Version {
			return httpx.ErrConflict
		}
		var newManager sql.NullInt64
		if ch.ManagerID != nil {
			m, err := q.GetEmployeeForUpdate(ctx, *ch.ManagerID)
			if errors.Is(err, sql.ErrNoRows) {
				v := httpx.Validation{}
				v.Add("manager_id", "Manager not found.")
				return v.Err()
			}
			if err != nil {
				return err
			}
			if !m.Active {
				return ErrManagerActive
			}
			if err := s.checkNoCycle(ctx, q, id, *ch.ManagerID); err != nil {
				return err
			}
			newManager = sql.NullInt64{Int64: int64(*ch.ManagerID), Valid: true}
		}
		if cur.ManagerID == newManager {
			return nil
		}
		n, err := q.UpdateEmployeeManager(ctx, store.UpdateEmployeeManagerParams{ManagerID: newManager, ID: id, Version: ch.Version})
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		return q.InsertManagerEvent(ctx, store.InsertManagerEventParams{
			EmployeeID: id, OldManagerID: cur.ManagerID, NewManagerID: newManager,
			ChangedBy: sql.NullInt64{Int64: int64(actorID), Valid: true}, Reason: ch.Reason,
		})
	})
	if errors.Is(err, database.ErrLockTimeout) {
		return httpx.E(http.StatusServiceUnavailable, "busy", "Another change is in progress. Please try again.")
	}
	return err
}

// checkNoCycle walks up from the proposed manager; reaching the employee means a cycle.
func (s *Service) checkNoCycle(ctx context.Context, q *store.Queries, employeeID, managerID uint64) error {
	cur := managerID
	for range maxChainDepth {
		next, err := q.GetManagerOf(ctx, cur)
		if err != nil {
			return err
		}
		if !next.Valid {
			return nil
		}
		if uint64(next.Int64) == employeeID {
			return ErrManagerCycle
		}
		cur = uint64(next.Int64)
	}
	return ErrManagerCycle // deeper than any real org chart: treat as invalid
}

type Candidate struct {
	ID       uint64 `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

func (s *Service) ManagerCandidates(ctx context.Context, employeeID uint64, search string) ([]Candidate, error) {
	search = strings.TrimSpace(search)
	if len(search) > 100 {
		search = search[:100]
	}
	rows, err := s.q.SearchManagerCandidates(ctx, store.SearchManagerCandidatesParams{
		ExcludeID: employeeID, Search: search, Pattern: httpx.LikePattern(search),
	})
	if err != nil {
		return nil, err
	}
	out := make([]Candidate, 0, len(rows))
	for _, r := range rows {
		out = append(out, Candidate{ID: r.ID, FullName: r.FullName, Email: r.Email})
	}
	return out, nil
}
