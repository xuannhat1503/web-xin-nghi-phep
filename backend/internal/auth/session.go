package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/g-asiapacific/elm/backend/internal/store"
)

var ErrNoSession = errors.New("no valid session")

// createSession stores the hash of a fresh opaque token and returns the raw token for the cookie.
func (s *Service) createSession(ctx context.Context, employeeID uint64, userAgent string) (string, error) {
	raw := randomToken()
	now := s.now()
	if len(userAgent) > 255 {
		userAgent = userAgent[:255]
	}
	err := s.q.InsertSession(ctx, store.InsertSessionParams{
		TokenHash:         hashToken(raw),
		EmployeeID:        employeeID,
		Now:               now,
		IdleExpiresAt:     now.Add(s.idle),
		AbsoluteExpiresAt: now.Add(s.absolute),
		UserAgent:         userAgent,
	})
	if err != nil {
		return "", fmt.Errorf("insert session: %w", err)
	}
	return raw, nil
}

// lookupSession validates the token and slides the idle timeout (at most one write per touchInterval).
// The employee row is read on every request, so deactivation and role changes apply immediately.
func (s *Service) lookupSession(ctx context.Context, raw string) (*Principal, error) {
	if raw == "" || len(raw) > 128 {
		return nil, ErrNoSession
	}
	now := s.now()
	row, err := s.q.GetValidSession(ctx, store.GetValidSessionParams{TokenHash: hashToken(raw), Now: now})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNoSession
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}
	// Tightening ALLOWED_EMAILS / ALLOWED_EMAIL_DOMAIN also ends sessions that were opened before.
	if !s.access.Allows(row.Email) && !s.bootstrap[strings.ToLower(row.Email)] {
		return nil, ErrNoSession
	}

	idleExp := row.IdleExpiresAt
	if now.Sub(row.LastSeenAt) >= touchInterval {
		idleExp = now.Add(s.idle)
		if idleExp.After(row.AbsoluteExpiresAt) {
			idleExp = row.AbsoluteExpiresAt
		}
		if err := s.q.TouchSession(ctx, store.TouchSessionParams{Now: now, IdleExpiresAt: idleExp, TokenHash: row.TokenHash}); err != nil {
			return nil, fmt.Errorf("touch session: %w", err)
		}
	}

	p := &Principal{
		EmployeeID:        row.EmployeeID,
		FullName:          row.FullName,
		Email:             row.Email,
		Role:              row.Role,
		Active:            row.Active,
		IdleExpiresAt:     idleExp,
		AbsoluteExpiresAt: row.AbsoluteExpiresAt,
		rawToken:          raw,
	}
	if row.ManagerID.Valid {
		m := uint64(row.ManagerID.Int64)
		p.ManagerID = &m
	}
	return p, nil
}

func (s *Service) revokeSession(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.q.RevokeSession(ctx, store.RevokeSessionParams{Now: sql.NullTime{Time: s.now(), Valid: true}, TokenHash: hashToken(raw)})
}

// RevokeEmployeeSessions logs an employee out everywhere (used when an Admin deactivates them).
func (s *Service) RevokeEmployeeSessions(ctx context.Context, q *store.Queries, employeeID uint64) error {
	_, err := q.RevokeEmployeeSessions(ctx, store.RevokeEmployeeSessionsParams{
		Now: sql.NullTime{Time: s.now(), Valid: true}, EmployeeID: employeeID,
	})
	return err
}

// Cleanup deletes dead sessions and expired login attempts. Run periodically by the worker.
func (s *Service) Cleanup(ctx context.Context) error {
	now := s.now()
	if _, err := s.q.DeleteDeadSessions(ctx, store.DeleteDeadSessionsParams{Now: now}); err != nil {
		return fmt.Errorf("delete sessions: %w", err)
	}
	if _, err := s.q.DeleteExpiredOAuthStates(ctx, now); err != nil {
		return fmt.Errorf("delete oauth states: %w", err)
	}
	return nil
}

func (s *Service) setSessionCookie(w http.ResponseWriter, raw string) {
	http.SetCookie(w, &http.Cookie{
		Name:     s.SessionCookieName(),
		Value:    raw,
		Path:     "/",
		MaxAge:   int(s.absolute.Seconds()),
		HttpOnly: true,
		Secure:   s.secureCookies,
		SameSite: http.SameSiteLaxMode, // Lax: the cookie must survive the top-level redirect back from Google
	})
}

func (s *Service) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: s.SessionCookieName(), Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	})
}
