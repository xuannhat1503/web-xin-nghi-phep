package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

// DefaultRule applies to a leave type until an Admin saves rules for it in a policy version.
var DefaultRule = LeaveTypeRule{ReasonRequired: true}

// LeaveType is an Admin-managed kind of leave. The code is permanent; names, order and status are editable.
type LeaveType struct {
	Code      string `json:"code"`
	NameVI    string `json:"name_vi"`
	NameEN    string `json:"name_en"`
	SortOrder int16  `json:"sort_order"`
	Active    bool   `json:"active"`
}

func (s *Service) LeaveTypes(ctx context.Context) ([]LeaveType, error) { return leaveTypes(ctx, s.q) }

func leaveTypes(ctx context.Context, q *store.Queries) ([]LeaveType, error) {
	rows, err := q.ListLeaveTypes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LeaveType, 0, len(rows))
	for _, r := range rows {
		out = append(out, LeaveType{Code: r.Code, NameVI: r.NameVi, NameEN: r.NameEn, SortOrder: r.SortOrder, Active: r.Active})
	}
	return out, nil
}

// activeCodes lists the codes employees may choose, in display order.
func activeCodes(types []LeaveType) []string {
	out := []string{}
	for _, t := range types {
		if t.Active {
			out = append(out, t.Code)
		}
	}
	return out
}

type LeaveTypeInput struct {
	Code      string `json:"code"`
	NameVI    string `json:"name_vi"`
	NameEN    string `json:"name_en"`
	SortOrder int16  `json:"sort_order"`
	Active    *bool  `json:"active"`
}

func validateLeaveType(in *LeaveTypeInput, creating bool) error {
	v := httpx.Validation{}
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	in.NameVI = strings.Join(strings.Fields(in.NameVI), " ")
	in.NameEN = strings.Join(strings.Fields(in.NameEN), " ")
	if creating && !reasonCode.MatchString(in.Code) {
		v.Add("code", "Code must be 2-40 characters: A-Z, 0-9 and _ (starting with a letter).")
	}
	if l := len([]rune(in.NameVI)); l < 2 || l > 100 {
		v.Add("name_vi", "Name must be 2-100 characters.")
	}
	if l := len([]rune(in.NameEN)); l < 2 || l > 100 {
		v.Add("name_en", "Name must be 2-100 characters.")
	}
	if in.SortOrder < 0 || in.SortOrder > 9999 {
		v.Add("sort_order", "Sort order must be between 0 and 9999.")
	}
	return v.Err()
}

func (s *Service) CreateLeaveType(ctx context.Context, actorID uint64, in LeaveTypeInput) (LeaveType, error) {
	if err := validateLeaveType(&in, true); err != nil {
		return LeaveType{}, err
	}
	out := LeaveType{Code: in.Code, NameVI: in.NameVI, NameEN: in.NameEN, SortOrder: in.SortOrder, Active: true}
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		err := q.InsertLeaveType(ctx, store.InsertLeaveTypeParams{Code: in.Code, NameVi: in.NameVI, NameEn: in.NameEN, SortOrder: in.SortOrder})
		if database.IsDuplicateKey(err) {
			v := httpx.Validation{}
			v.Add("code", "This code already exists.")
			return v.Err()
		}
		if err != nil {
			return err
		}
		a, _ := json.Marshal(out)
		return q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: actor(actorID), Action: "LEAVE_TYPE_CREATED", AfterData: a})
	})
	return out, err
}

func (s *Service) UpdateLeaveType(ctx context.Context, actorID uint64, code string, in LeaveTypeInput) (LeaveType, error) {
	if err := validateLeaveType(&in, false); err != nil {
		return LeaveType{}, err
	}
	var out LeaveType
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.GetLeaveTypeForUpdate(ctx, code)
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		active := cur.Active
		if in.Active != nil {
			active = *in.Active
		}
		if cur.Active && !active {
			if err := keepOneActiveType(ctx, q, code); err != nil {
				return err
			}
		}
		if err := q.UpdateLeaveType(ctx, store.UpdateLeaveTypeParams{NameVi: in.NameVI, NameEn: in.NameEN, SortOrder: in.SortOrder, Active: active, Code: code}); err != nil {
			return err
		}
		before := LeaveType{Code: cur.Code, NameVI: cur.NameVi, NameEN: cur.NameEn, SortOrder: cur.SortOrder, Active: cur.Active}
		out = LeaveType{Code: cur.Code, NameVI: in.NameVI, NameEN: in.NameEN, SortOrder: in.SortOrder, Active: active}
		b, _ := json.Marshal(before)
		a, _ := json.Marshal(out)
		return q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: actor(actorID), Action: "LEAVE_TYPE_UPDATED", BeforeData: b, AfterData: a})
	})
	return out, err
}

// DeleteLeaveType removes a type that no request has used; a used type is deactivated instead so history
// keeps its name. It reports whether the row was really deleted.
func (s *Service) DeleteLeaveType(ctx context.Context, actorID uint64, code string) (deleted bool, err error) {
	err = database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.GetLeaveTypeForUpdate(ctx, code)
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.Active {
			if err := keepOneActiveType(ctx, q, code); err != nil {
				return err
			}
		}
		used, err := q.CountLeaveTypeUsage(ctx, code)
		if err != nil {
			return err
		}
		b, _ := json.Marshal(LeaveType{Code: cur.Code, NameVI: cur.NameVi, NameEN: cur.NameEn, SortOrder: cur.SortOrder, Active: cur.Active})
		action := "LEAVE_TYPE_DELETED"
		if used > 0 {
			action = "LEAVE_TYPE_DEACTIVATED"
			err = q.UpdateLeaveType(ctx, store.UpdateLeaveTypeParams{NameVi: cur.NameVi, NameEn: cur.NameEn, SortOrder: cur.SortOrder, Active: false, Code: code})
		} else {
			_, err = q.DeleteLeaveType(ctx, code)
			deleted = true
		}
		if err != nil {
			return err
		}
		return q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: actor(actorID), Action: action, BeforeData: b})
	})
	return deleted, err
}

func keepOneActiveType(ctx context.Context, q *store.Queries, code string) error {
	types, err := leaveTypes(ctx, q)
	if err != nil {
		return err
	}
	for _, c := range activeCodes(types) {
		if c != code {
			return nil
		}
	}
	return httpx.E(http.StatusUnprocessableEntity, "leave_type_last", "At least one leave type must stay active.")
}

// DeleteReason removes an unused reason; a used one is deactivated instead. "OTHER" cannot be removed.
func (s *Service) DeleteReason(ctx context.Context, actorID, id uint64) (deleted bool, err error) {
	err = database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.GetLeaveReasonForUpdate(ctx, id)
		if errors.Is(err, sql.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if cur.Code == "OTHER" {
			return httpx.E(http.StatusUnprocessableEntity, "reason_locked", `"Other" must stay active and require a description.`)
		}
		used, err := q.CountReasonUsage(ctx, cur.Code)
		if err != nil {
			return err
		}
		b, _ := json.Marshal(reasonFromRow(cur))
		action := "REASON_DELETED"
		if used > 0 {
			action = "REASON_DEACTIVATED"
			err = q.UpdateLeaveReason(ctx, store.UpdateLeaveReasonParams{Label: cur.Label, LabelEn: cur.LabelEn, RequiresText: cur.RequiresText, SortOrder: cur.SortOrder, Active: false, ID: id})
		} else {
			_, err = q.DeleteLeaveReason(ctx, id)
			deleted = true
		}
		if err != nil {
			return err
		}
		return q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: actor(actorID), Action: action, BeforeData: b})
	})
	return deleted, err
}

func reasonFromRow(r store.GetLeaveReasonForUpdateRow) Reason {
	return Reason{ID: r.ID, Code: r.Code, Label: r.Label, LabelEN: r.LabelEn, RequiresText: r.RequiresText, SortOrder: r.SortOrder, Active: r.Active}
}

func actor(id uint64) sql.NullInt64 { return sql.NullInt64{Int64: int64(id), Valid: true} }

// Catalog is what every page needs to label leave types and reasons (including inactive ones in history).
type Catalog struct {
	LeaveTypes []LeaveType `json:"leave_types"`
	Reasons    []Reason    `json:"reasons"`
}

func (s *Service) Catalog(ctx context.Context) (Catalog, error) {
	types, err := s.LeaveTypes(ctx)
	if err != nil {
		return Catalog{}, err
	}
	reasons, err := s.Reasons(ctx)
	if err != nil {
		return Catalog{}, err
	}
	return Catalog{LeaveTypes: types, Reasons: reasons}, nil
}
