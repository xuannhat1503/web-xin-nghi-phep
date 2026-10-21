package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

var ErrInvalidIdentity = errors.New("invalid identity")

// normalize rejects unverified or malformed identities and bounds field lengths.
func normalize(id Identity) (Identity, error) {
	id.Subject = strings.TrimSpace(id.Subject)
	id.Email = strings.ToLower(strings.TrimSpace(id.Email))
	id.Name = strings.Join(strings.Fields(id.Name), " ")
	switch {
	case !id.EmailVerified:
		return id, fmt.Errorf("%w: email not verified", ErrInvalidIdentity)
	case id.Subject == "" || len(id.Subject) > maxSubjectLength:
		return id, fmt.Errorf("%w: subject", ErrInvalidIdentity)
	case id.Email == "" || len(id.Email) > maxEmailLength || !strings.Contains(id.Email, "@"):
		return id, fmt.Errorf("%w: email", ErrInvalidIdentity)
	}
	if id.Name == "" {
		id.Name, _, _ = strings.Cut(id.Email, "@")
	}
	if utf8.RuneCountInString(id.Name) > maxNameLength {
		id.Name = string([]rune(id.Name)[:maxNameLength])
	}
	return id, nil
}

// upsertEmployee maps a verified identity to an employee, keyed by the stable Google subject.
// New accounts are inactive EMPLOYEEs, except bootstrap emails which become active ADMINs —
// only at creation, so an Admin who was later demoted is never re-promoted by logging in.
func (s *Service) upsertEmployee(ctx context.Context, raw Identity) (store.Employee, error) {
	id, err := normalize(raw)
	if err != nil {
		return store.Employee{}, err
	}
	emp, err := s.upsertOnce(ctx, id)
	if database.IsDuplicateKey(err) {
		// Two first logins raced; the other one created the row. Retry once as an existing account.
		emp, err = s.upsertOnce(ctx, id)
	}
	return emp, err
}

func (s *Service) upsertOnce(ctx context.Context, id Identity) (store.Employee, error) {
	var emp store.Employee
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		now := sql.NullTime{Time: s.now(), Valid: true}

		existing, err := q.GetEmployeeBySubject(ctx, id.Subject)
		switch {
		case err == nil:
			if existing.Email != id.Email || existing.FullName != id.Name {
				before, _ := json.Marshal(map[string]string{"email": existing.Email, "full_name": existing.FullName})
				after, _ := json.Marshal(map[string]string{"email": id.Email, "full_name": id.Name})
				if err := q.InsertEmployeeEvent(ctx, store.InsertEmployeeEventParams{
					EmployeeID: existing.ID, ActorID: sql.NullInt64{Int64: int64(existing.ID), Valid: true},
					Action: "PROFILE_SYNCED", BeforeData: before, AfterData: after,
				}); err != nil {
					return err
				}
			}
			if err := q.UpdateEmployeeLogin(ctx, store.UpdateEmployeeLoginParams{
				ID: existing.ID, Email: id.Email, FullName: id.Name, LastLoginAt: now,
			}); err != nil {
				return err
			}
		case errors.Is(err, sql.ErrNoRows):
			role, active := store.EmployeesRoleEMPLOYEE, false
			bootstrap := s.bootstrap[id.Email]
			if bootstrap {
				// Never bootstrap a second Google account that reuses an email already on file.
				inUse, err := q.EmailInUse(ctx, id.Email)
				if err != nil {
					return err
				}
				bootstrap = !inUse
			}
			if bootstrap {
				role, active = store.EmployeesRoleADMIN, true
			}
			newID, err := q.CreateEmployee(ctx, store.CreateEmployeeParams{
				GoogleSubjectID: id.Subject, Email: id.Email, FullName: id.Name,
				Role: role, Active: active, LastLoginAt: now,
			})
			if err != nil {
				return err
			}
			after, _ := json.Marshal(map[string]any{"role": role, "active": active, "bootstrap_admin": bootstrap})
			if err := q.InsertEmployeeEvent(ctx, store.InsertEmployeeEventParams{
				EmployeeID: uint64(newID), Action: "CREATED", AfterData: after,
			}); err != nil {
				return err
			}
		default:
			return err
		}
		emp, err = q.GetEmployeeBySubject(ctx, id.Subject)
		return err
	})
	return emp, err
}
