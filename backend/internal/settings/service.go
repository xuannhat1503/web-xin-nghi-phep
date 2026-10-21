package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/civil"
	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

type Service struct {
	db             *sql.DB
	q              *store.Queries
	smtpConfigured bool
	smtpFrom       string
	liveEnabled    bool
	access         config.Access
}

func NewService(db *sql.DB) *Service { return &Service{db: db, q: store.New(db)} }

// Current returns the active policy (used by leave calculation and the settings page).
func (s *Service) Current(ctx context.Context) (Policy, error) { return current(ctx, s.q) }

func current(ctx context.Context, q *store.Queries) (Policy, error) {
	r, err := q.GetCurrentPolicy(ctx)
	if err != nil {
		return Policy{}, fmt.Errorf("load current policy: %w", err)
	}
	p := Policy{
		VersionID: r.ID, VersionNo: r.VersionNo, Timezone: r.Timezone, WorkdaysMask: r.WorkdaysMask,
		MinutesPerDay: int(r.MinutesPerDay.Int16), MinLeaveMinutes: int(r.MinLeaveMinutes),
		LeaveUnitMinutes: int(r.LeaveUnitMinutes), ReminderBasis: string(r.ReminderBasis),
		ReminderIntervalMinutes: int(r.ReminderIntervalMinutes), ReminderMaxCount: int(r.ReminderMaxCount),
		ChangeNote: r.ChangeNote, CreatedAt: r.CreatedAt, LeaveTypes: map[string]LeaveTypeRule{},
	}
	for field, src := range map[*civil.Clock]string{&p.WorkStart: r.WorkStart, &p.WorkEnd: r.WorkEnd, &p.LunchStart: r.LunchStart, &p.LunchEnd: r.LunchEnd} {
		c, err := civil.ParseClock(src)
		if err != nil {
			return Policy{}, fmt.Errorf("stored policy time %q: %w", src, err)
		}
		*field = c
	}
	stored, err := storedRules(ctx, q, r.ID)
	if err != nil {
		return Policy{}, err
	}
	types, err := leaveTypes(ctx, q)
	if err != nil {
		return Policy{}, err
	}
	// Only active types are selectable; a type without stored rules uses the default until the policy is saved.
	for _, code := range activeCodes(types) {
		rule, ok := stored[code]
		if !ok {
			rule = DefaultRule
		}
		p.LeaveTypes[code] = rule
	}
	return p, nil
}

func storedRules(ctx context.Context, q *store.Queries, policyID uint64) (map[string]LeaveTypeRule, error) {
	rows, err := q.ListLeaveTypePolicies(ctx, policyID)
	if err != nil {
		return nil, err
	}
	out := map[string]LeaveTypeRule{}
	for _, t := range rows {
		out[t.LeaveType] = LeaveTypeRule{
			AdvanceNoticeDays: int(t.AdvanceNoticeDays), AllowPastDates: t.AllowPastDates,
			MaxPastDays: int(t.MaxPastDays), ReasonRequired: t.ReasonRequired,
		}
	}
	return out, nil
}

type Reason struct {
	ID           uint64 `json:"id"`
	Code         string `json:"code"`
	Label        string `json:"label"`    // Vietnamese
	LabelEN      string `json:"label_en"` // English
	RequiresText bool   `json:"requires_text"`
	SortOrder    int16  `json:"sort_order"`
	Active       bool   `json:"active"`
}

type VersionSummary struct {
	VersionNo  uint32    `json:"version_no"`
	ChangeNote string    `json:"change_note"`
	CreatedAt  time.Time `json:"created_at"`
	CreatedBy  string    `json:"created_by"`
}

type Overview struct {
	SettingsVersion uint32           `json:"settings_version"`
	Policy          policyJSON       `json:"policy"`
	FinanceCC       []string         `json:"finance_cc_emails"`
	Reasons         []Reason         `json:"reasons"`
	LeaveTypes      []LeaveType      `json:"leave_types"`
	History         []VersionSummary `json:"history"`
	// Open requests that will be checked against the new policy when submitted/resubmitted.
	// Approved, rejected and cancelled requests keep their stored calculation.
	OpenRequests map[string]int64 `json:"open_requests"`
	Email        EmailSettings    `json:"email"`
}

func (s *Service) Overview(ctx context.Context) (Overview, error) {
	p, err := s.Current(ctx)
	if err != nil {
		return Overview{}, err
	}
	raw, err := s.q.GetCurrentPolicy(ctx)
	if err != nil {
		return Overview{}, err
	}
	reasons, err := s.Reasons(ctx)
	if err != nil {
		return Overview{}, err
	}
	types, err := s.LeaveTypes(ctx)
	if err != nil {
		return Overview{}, err
	}
	hist, err := s.q.ListPolicyVersions(ctx)
	if err != nil {
		return Overview{}, err
	}
	counts, err := s.q.CountOpenRequestsByStatus(ctx)
	if err != nil {
		return Overview{}, err
	}
	o := Overview{
		SettingsVersion: raw.SettingsVersion, Policy: p.JSON(), FinanceCC: splitEmails(raw.FinanceCcEmails),
		Reasons: reasons, LeaveTypes: types, OpenRequests: map[string]int64{"DRAFT": 0, "PENDING": 0, "NEEDS_CHANGES": 0},
	}
	for _, h := range hist {
		by := "System"
		if h.CreatedByName.Valid {
			by = h.CreatedByName.String
		}
		o.History = append(o.History, VersionSummary{VersionNo: h.VersionNo, ChangeNote: h.ChangeNote, CreatedAt: h.CreatedAt, CreatedBy: by})
	}
	for _, c := range counts {
		o.OpenRequests[string(c.Status)] = c.N
	}
	if o.Email, err = s.Email(ctx); err != nil {
		return Overview{}, err
	}
	return o, nil
}

// UpdatePolicy validates the input and stores it as a new immutable version.
func (s *Service) UpdatePolicy(ctx context.Context, actorID uint64, in PolicyInput) (Policy, error) {
	var out Policy
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		var err error
		out, err = updatePolicy(ctx, s.q.WithTx(tx), actorID, in)
		return err
	})
	return out, err
}

func updatePolicy(ctx context.Context, q *store.Queries, actorID uint64, in PolicyInput) (Policy, error) {
	types, err := leaveTypes(ctx, q)
	if err != nil {
		return Policy{}, err
	}
	p, err := Validate(in, activeCodes(types))
	if err != nil {
		return Policy{}, err
	}
	cs, err := q.LockCompanySettings(ctx)
	if err != nil {
		return Policy{}, err
	}
	if cs.Version != in.Version {
		return Policy{}, httpx.ErrConflict
	}
	before, err := current(ctx, q)
	if err != nil {
		return Policy{}, err
	}
	next, err := q.NextPolicyVersionNo(ctx)
	if err != nil {
		return Policy{}, err
	}
	actor := sql.NullInt64{Int64: int64(actorID), Valid: true}
	id, err := q.InsertPolicyVersion(ctx, store.InsertPolicyVersionParams{
		VersionNo: uint32(next), Timezone: p.Timezone,
		WorkStart: p.WorkStart.SQL(), WorkEnd: p.WorkEnd.SQL(), LunchStart: p.LunchStart.SQL(), LunchEnd: p.LunchEnd.SQL(),
		WorkdaysMask: p.WorkdaysMask, MinLeaveMinutes: uint16(p.MinLeaveMinutes), LeaveUnitMinutes: uint16(p.LeaveUnitMinutes),
		ReminderBasis: store.PolicyVersionsReminderBasis(p.ReminderBasis), ReminderIntervalMinutes: uint32(p.ReminderIntervalMinutes),
		ReminderMaxCount: uint8(p.ReminderMaxCount), ChangeNote: p.ChangeNote, CreatedBy: actor,
	})
	if err != nil {
		return Policy{}, err
	}
	// Inactive types keep their last rules so reactivating one restores them.
	rules, err := storedRules(ctx, q, before.VersionID)
	if err != nil {
		return Policy{}, err
	}
	for t, r := range p.LeaveTypes {
		rules[t] = r
	}
	for t, r := range rules {
		if err := q.InsertLeaveTypePolicy(ctx, store.InsertLeaveTypePolicyParams{
			PolicyVersionID: uint64(id), LeaveType: t,
			AdvanceNoticeDays: uint16(r.AdvanceNoticeDays), AllowPastDates: r.AllowPastDates,
			MaxPastDays: uint16(r.MaxPastDays), ReasonRequired: r.ReasonRequired,
		}); err != nil {
			return Policy{}, err
		}
	}
	n, err := q.SetCurrentPolicy(ctx, store.SetCurrentPolicyParams{PolicyVersionID: uint64(id), UpdatedBy: actor, Version: in.Version})
	if err != nil {
		return Policy{}, err
	}
	if n != 1 {
		return Policy{}, httpx.ErrConflict
	}
	after, err := current(ctx, q)
	if err != nil {
		return Policy{}, err
	}
	b, _ := json.Marshal(before.JSON())
	a, _ := json.Marshal(after.JSON())
	if err := q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: actor, Action: "POLICY_VERSION_CREATED", BeforeData: b, AfterData: a}); err != nil {
		return Policy{}, err
	}
	return after, nil
}

type FinanceInput struct {
	Version uint32   `json:"version"`
	Emails  []string `json:"finance_cc_emails"`
}

// UpdateFinanceCC stores the Finance CC list shown in Mock Email previews. Nothing is ever delivered to it.
func (s *Service) UpdateFinanceCC(ctx context.Context, actorID uint64, in FinanceInput) error {
	emails, err := validateEmails(in.Emails)
	if err != nil {
		return err
	}
	return database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cs, err := q.LockCompanySettings(ctx)
		if err != nil {
			return err
		}
		if cs.Version != in.Version {
			return httpx.ErrConflict
		}
		actor := sql.NullInt64{Int64: int64(actorID), Valid: true}
		n, err := q.SetFinanceCC(ctx, store.SetFinanceCCParams{FinanceCcEmails: strings.Join(emails, ","), UpdatedBy: actor, Version: in.Version})
		if err != nil {
			return err
		}
		if n != 1 {
			return httpx.ErrConflict
		}
		b, _ := json.Marshal(map[string]any{"finance_cc_emails": splitEmails(cs.FinanceCcEmails)})
		a, _ := json.Marshal(map[string]any{"finance_cc_emails": emails})
		return q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: actor, Action: "FINANCE_CC_UPDATED", BeforeData: b, AfterData: a})
	})
}

func validateEmails(in []string) ([]string, error) {
	v := httpx.Validation{}
	var out []string
	seen := map[string]bool{}
	for _, e := range in {
		e = strings.ToLower(strings.TrimSpace(e))
		if e == "" {
			continue
		}
		addr, err := mail.ParseAddress(e)
		if err != nil || addr.Address != e || len(e) > 254 {
			v.Add("finance_cc_emails", fmt.Sprintf("%q is not a valid email address.", e))
			continue
		}
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	if len(out) > 10 {
		v.Add("finance_cc_emails", "At most 10 addresses.")
	}
	return out, v.Err()
}

func splitEmails(s string) []string {
	out := []string{}
	for _, e := range strings.Split(s, ",") {
		if e = strings.TrimSpace(e); e != "" {
			out = append(out, e)
		}
	}
	return out
}

func (s *Service) Reasons(ctx context.Context) ([]Reason, error) {
	rows, err := s.q.ListLeaveReasons(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Reason, 0, len(rows))
	for _, r := range rows {
		out = append(out, Reason{ID: r.ID, Code: r.Code, Label: r.Label, LabelEN: r.LabelEn, RequiresText: r.RequiresText, SortOrder: r.SortOrder, Active: r.Active})
	}
	return out, nil
}

var reasonCode = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,39}$`)

type ReasonInput struct {
	Code         string `json:"code"`
	Label        string `json:"label"`
	LabelEN      string `json:"label_en"`
	RequiresText bool   `json:"requires_text"`
	SortOrder    int16  `json:"sort_order"`
	Active       *bool  `json:"active"`
}

func validateReason(in *ReasonInput, creating bool) error {
	v := httpx.Validation{}
	in.Label = strings.Join(strings.Fields(in.Label), " ")
	in.LabelEN = strings.Join(strings.Fields(in.LabelEN), " ")
	if in.LabelEN == "" {
		in.LabelEN = in.Label
	}
	in.Code = strings.ToUpper(strings.TrimSpace(in.Code))
	if creating && !reasonCode.MatchString(in.Code) {
		v.Add("code", "Code must be 2-40 characters: A-Z, 0-9 and _ (starting with a letter).")
	}
	if l := len([]rune(in.Label)); l < 2 || l > 200 {
		v.Add("label", "Label must be 2-200 characters.")
	}
	if l := len([]rune(in.LabelEN)); l < 2 || l > 200 {
		v.Add("label_en", "Label must be 2-200 characters.")
	}
	if in.SortOrder < 0 || in.SortOrder > 9999 {
		v.Add("sort_order", "Sort order must be between 0 and 9999.")
	}
	return v.Err()
}

func (s *Service) CreateReason(ctx context.Context, actorID uint64, in ReasonInput) (Reason, error) {
	if err := validateReason(&in, true); err != nil {
		return Reason{}, err
	}
	var out Reason
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		id, err := q.InsertLeaveReason(ctx, store.InsertLeaveReasonParams{Code: in.Code, Label: in.Label, LabelEn: in.LabelEN, RequiresText: in.RequiresText, SortOrder: in.SortOrder})
		if database.IsDuplicateKey(err) {
			v := httpx.Validation{}
			v.Add("code", "This code already exists.")
			return v.Err()
		}
		if err != nil {
			return err
		}
		out = Reason{ID: uint64(id), Code: in.Code, Label: in.Label, LabelEN: in.LabelEN, RequiresText: in.RequiresText, SortOrder: in.SortOrder, Active: true}
		a, _ := json.Marshal(out)
		return q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: sql.NullInt64{Int64: int64(actorID), Valid: true}, Action: "REASON_CREATED", AfterData: a})
	})
	return out, err
}

// UpdateReason edits label/order/flags. The code is immutable; "OTHER" must stay active and require text.
func (s *Service) UpdateReason(ctx context.Context, actorID, id uint64, in ReasonInput) (Reason, error) {
	if err := validateReason(&in, false); err != nil {
		return Reason{}, err
	}
	var out Reason
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		cur, err := q.GetLeaveReasonForUpdate(ctx, id)
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
		if cur.Code == "OTHER" && (!active || !in.RequiresText) {
			return httpx.E(http.StatusUnprocessableEntity, "reason_locked", `"Other" must stay active and require a description.`)
		}
		if err := q.UpdateLeaveReason(ctx, store.UpdateLeaveReasonParams{Label: in.Label, LabelEn: in.LabelEN, RequiresText: in.RequiresText, SortOrder: in.SortOrder, Active: active, ID: id}); err != nil {
			return err
		}
		before := reasonFromRow(cur)
		out = Reason{ID: cur.ID, Code: cur.Code, Label: in.Label, LabelEN: in.LabelEN, RequiresText: in.RequiresText, SortOrder: in.SortOrder, Active: active}
		b, _ := json.Marshal(before)
		a, _ := json.Marshal(out)
		return q.InsertSettingsEvent(ctx, store.InsertSettingsEventParams{ActorID: sql.NullInt64{Int64: int64(actorID), Valid: true}, Action: "REASON_UPDATED", BeforeData: b, AfterData: a})
	})
	return out, err
}

// CurrentWith reads the active policy with the given queries (e.g. inside a caller's transaction).
func CurrentWith(ctx context.Context, q *store.Queries) (Policy, error) { return current(ctx, q) }
