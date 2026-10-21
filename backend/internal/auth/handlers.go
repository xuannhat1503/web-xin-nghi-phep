package auth

import (
	"crypto/subtle"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

// Login starts the Authorization Code + PKCE flow: GET /auth/google/login?return_to=/path
func (s *Service) Login(c *gin.Context) {
	ctx := c.Request.Context()
	state, nonce, verifier := randomToken(), randomToken(), oauth2.GenerateVerifier()
	now := s.now()
	err := s.q.InsertOAuthState(ctx, store.InsertOAuthStateParams{
		StateHash:    hashToken(state),
		Nonce:        nonce,
		CodeVerifier: verifier,
		ReturnTo:     SafeReturnTo(c.Query("return_to")),
		Now:          now,
		ExpiresAt:    now.Add(stateTTL),
	})
	if err != nil {
		s.loginFailed(c, "login_failed", "store oauth state", err)
		return
	}
	authURL, err := s.idp.AuthCodeURL(ctx, state, nonce, verifier)
	if err != nil {
		s.loginFailed(c, "provider_unavailable", "build auth url", err)
		return
	}
	// Binds the login attempt to this browser (login CSRF protection) in addition to the server-side state.
	http.SetCookie(c.Writer, &http.Cookie{
		Name: stateCookieName, Value: state, Path: stateCookiePath, MaxAge: int(stateTTL.Seconds()),
		HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode,
	})
	c.Redirect(http.StatusFound, authURL)
}

// Callback completes the flow: GET /auth/google/callback?state=...&code=...
func (s *Service) Callback(c *gin.Context) {
	ctx := c.Request.Context()
	cookieState, _ := c.Cookie(stateCookieName)
	http.SetCookie(c.Writer, &http.Cookie{Name: stateCookieName, Value: "", Path: stateCookiePath, MaxAge: -1, HttpOnly: true, Secure: s.secureCookies, SameSite: http.SameSiteLaxMode})

	if e := c.Query("error"); e != "" {
		code := "login_failed"
		if e == "access_denied" {
			code = "access_denied"
		}
		s.loginFailed(c, code, "provider returned error", errors.New(truncate(e, 64)))
		return
	}
	state, code := c.Query("state"), c.Query("code")
	if state == "" || code == "" || cookieState == "" || subtle.ConstantTimeCompare([]byte(state), []byte(cookieState)) != 1 {
		s.loginFailed(c, "login_failed", "state mismatch", nil)
		return
	}

	// Consume the state exactly once.
	var st store.GetOAuthStateForUpdateRow
	err := database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		q := s.q.WithTx(tx)
		var err error
		if st, err = q.GetOAuthStateForUpdate(ctx, hashToken(state)); err != nil {
			return err
		}
		return q.DeleteOAuthState(ctx, hashToken(state))
	})
	if err != nil || !st.ExpiresAt.After(s.now()) {
		s.loginFailed(c, "login_expired", "unknown or expired state", err)
		return
	}

	id, err := s.idp.Exchange(ctx, code, st.CodeVerifier, st.Nonce)
	if err != nil {
		s.loginFailed(c, "login_failed", "code exchange / token verification", err)
		return
	}
	// ALLOWED_EMAILS / ALLOWED_EMAIL_DOMAIN: refuse before any account is created. Bootstrap admins always pass.
	if !s.access.Allows(id.Email) && !s.bootstrap[strings.ToLower(id.Email)] {
		s.loginFailed(c, "not_allowed", "email not in the access allowlist", nil)
		return
	}
	emp, err := s.upsertEmployee(ctx, id)
	if err != nil {
		s.loginFailed(c, "login_failed", "upsert employee", err)
		return
	}
	if err := s.startSession(c, emp.ID); err != nil {
		s.loginFailed(c, "login_failed", "create session", err)
		return
	}
	s.log.Info("login", slog.Uint64("employee_id", emp.ID), slog.Bool("active", emp.Active))

	dest := st.ReturnTo
	if !emp.Active {
		dest = "/pending-activation"
	}
	c.Redirect(http.StatusSeeOther, dest)
}

// startSession rotates: any session presented by this browser is revoked before a new one is issued.
func (s *Service) startSession(c *gin.Context, employeeID uint64) error {
	if old, err := c.Cookie(s.SessionCookieName()); err == nil && old != "" {
		if err := s.revokeSession(c.Request.Context(), old); err != nil {
			return err
		}
	}
	raw, err := s.createSession(c.Request.Context(), employeeID, c.Request.UserAgent())
	if err != nil {
		return err
	}
	s.setSessionCookie(c.Writer, raw)
	return nil
}

// loginFailed logs the internal reason and redirects with a generic code (no account enumeration).
func (s *Service) loginFailed(c *gin.Context, code, reason string, err error) {
	attrs := []any{slog.String("reason", reason), slog.String("code", code), slog.String("request_id", c.GetString("request_id"))}
	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}
	s.log.Warn("login failed", attrs...)
	c.Redirect(http.StatusSeeOther, "/login?error="+url.QueryEscape(code))
}

// Logout revokes the session: POST /auth/logout (CSRF protected by middleware).
func (s *Service) Logout(c *gin.Context) {
	if p := CurrentPrincipal(c); p != nil {
		if err := s.revokeSession(c.Request.Context(), p.rawToken); err != nil {
			s.log.Error("logout revoke failed", slog.String("error", err.Error()))
			httpx.Abort(c, http.StatusInternalServerError, "internal_error", "Could not sign out")
			return
		}
	}
	s.clearSessionCookie(c.Writer)
	if strings.Contains(c.GetHeader("Accept"), "application/json") {
		c.Status(http.StatusNoContent)
		return
	}
	c.Redirect(http.StatusSeeOther, "/login?signed_out=1")
}

type meResponse struct {
	Employee struct {
		ID       uint64 `json:"id"`
		FullName string `json:"full_name"`
		Email    string `json:"email"`
		Role     string `json:"role"`
		Active   bool   `json:"active"`
	} `json:"employee"`
	IsReviewer bool   `json:"is_reviewer"`
	Unread     int64  `json:"unread_notifications"`
	CSRFToken  string `json:"csrf_token"`
	Session    struct {
		IdleExpiresAt     time.Time `json:"idle_expires_at"`
		AbsoluteExpiresAt time.Time `json:"absolute_expires_at"`
	} `json:"session"`
}

// Me returns the current principal: GET /api/v1/me (works for inactive users so the UI can show Pending Activation).
func (s *Service) Me(c *gin.Context) {
	p := CurrentPrincipal(c)
	var res meResponse
	res.Employee.ID = p.EmployeeID
	res.Employee.FullName = p.FullName
	res.Employee.Email = p.Email
	res.Employee.Role = string(p.Role)
	res.Employee.Active = p.Active
	res.CSRFToken = s.csrfToken(p.rawToken)
	res.Session.IdleExpiresAt = p.IdleExpiresAt
	res.Session.AbsoluteExpiresAt = p.AbsoluteExpiresAt
	if p.Active {
		has, err := s.q.HasActiveDirectReports(c.Request.Context(), sql.NullInt64{Int64: int64(p.EmployeeID), Valid: true})
		if err != nil {
			s.log.Error("me: reviewer lookup", slog.String("error", err.Error()))
			httpx.Abort(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
			return
		}
		if !has {
			if has, err = s.q.HasAssignedReviews(c.Request.Context(), sql.NullInt64{Int64: int64(p.EmployeeID), Valid: true}); err != nil {
				httpx.Abort(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
				return
			}
		}
		res.IsReviewer = has
		if err := s.db.QueryRowContext(c.Request.Context(), "SELECT COUNT(*) FROM notifications WHERE recipient_id = ? AND read_at IS NULL", p.EmployeeID).Scan(&res.Unread); err != nil {
			httpx.Abort(c, http.StatusInternalServerError, "internal_error", "An unexpected error occurred")
			return
		}
	}
	c.JSON(http.StatusOK, res)
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n]
	}
	return s
}
