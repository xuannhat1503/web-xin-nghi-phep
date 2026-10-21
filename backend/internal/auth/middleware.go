package auth

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/g-asiapacific/elm/backend/internal/httpx"
)

// CurrentPrincipal returns the authenticated principal or nil.
func CurrentPrincipal(c *gin.Context) *Principal {
	if v, ok := c.Get(principalCtxKey); ok {
		return v.(*Principal)
	}
	return nil
}

// LoadSession attaches the principal when a valid session cookie is present; it never rejects by itself.
func (s *Service) LoadSession() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := c.Cookie(s.SessionCookieName())
		if err != nil || raw == "" {
			c.Next()
			return
		}
		p, err := s.lookupSession(c.Request.Context(), raw)
		switch {
		case err == nil:
			c.Set(principalCtxKey, p)
		case errors.Is(err, ErrNoSession):
			s.clearSessionCookie(c.Writer)
		default:
			s.log.Error("session lookup failed", slog.String("error", err.Error()))
			httpx.Abort(c, http.StatusServiceUnavailable, "unavailable", "Service temporarily unavailable")
			return
		}
		c.Next()
	}
}

func isSafeMethod(m string) bool {
	return m == http.MethodGet || m == http.MethodHead || m == http.MethodOptions
}

// CSRF protects state-changing requests: the Origin header must be the app origin and, when a session
// exists, the synchronizer token must match (header X-CSRF-Token, or form field csrf_token).
func (s *Service) CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		if isSafeMethod(c.Request.Method) {
			c.Next()
			return
		}
		if c.GetHeader("Origin") != s.origin {
			httpx.Abort(c, http.StatusForbidden, "invalid_origin", "Request origin is not allowed")
			return
		}
		if p := CurrentPrincipal(c); p != nil {
			token := c.GetHeader(csrfHeader)
			if token == "" && c.ContentType() == "application/x-www-form-urlencoded" {
				token = c.PostForm(csrfFormField)
			}
			if !s.validCSRF(p.rawToken, token) {
				httpx.Abort(c, http.StatusForbidden, "invalid_csrf", "Invalid or missing CSRF token")
				return
			}
		}
		c.Next()
	}
}

func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		if CurrentPrincipal(c) == nil {
			httpx.Abort(c, http.StatusUnauthorized, "unauthenticated", "Sign in required")
			return
		}
		c.Next()
	}
}

// RequireActive blocks inactive (pending activation or deactivated) accounts from business endpoints.
func RequireActive() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := CurrentPrincipal(c)
		if p == nil {
			httpx.Abort(c, http.StatusUnauthorized, "unauthenticated", "Sign in required")
			return
		}
		if !p.Active {
			httpx.Abort(c, http.StatusForbidden, "account_inactive", "Your account is pending activation")
			return
		}
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := CurrentPrincipal(c)
		if p == nil || !p.Active || !p.IsAdmin() {
			httpx.Abort(c, http.StatusForbidden, "forbidden", "You do not have access to this resource")
			return
		}
		c.Next()
	}
}
