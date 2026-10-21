// Package auth implements Google OpenID Connect login, server-side MySQL sessions, CSRF protection
// and the authentication/authorization middleware. Go is the single source of truth for identity.
package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"log/slog"
	"strings"
	"time"

	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

// Identity is what we accept from a verified ID token. Role, activation and manager are never taken from it.
type Identity struct {
	Subject       string
	Email         string
	Name          string
	EmailVerified bool
}

// IdentityProvider abstracts the OIDC provider so the flow can be tested with a fake issuer.
type IdentityProvider interface {
	AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error)
	// Exchange redeems the code (with the PKCE verifier) and returns the verified identity.
	// Implementations must validate signature, issuer, audience, expiry and nonce.
	Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error)
}

// Principal is the authenticated employee for the current request.
type Principal struct {
	EmployeeID        uint64
	FullName          string
	Email             string
	Role              store.EmployeesRole
	Active            bool
	ManagerID         *uint64
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time

	rawToken string
}

func (p *Principal) IsAdmin() bool { return p.Role == store.EmployeesRoleADMIN }

const (
	stateTTL         = 10 * time.Minute
	touchInterval    = time.Minute
	stateCookieName  = "elm_oauth_state"
	stateCookiePath  = "/auth/google"
	csrfHeader       = "X-CSRF-Token"
	csrfFormField    = "csrf_token"
	principalCtxKey  = "auth.principal"
	maxSubjectLength = 255
	maxEmailLength   = 320
	maxNameLength    = 200
)

type Service struct {
	db            *sql.DB
	q             *store.Queries
	idp           IdentityProvider
	log           *slog.Logger
	origin        string
	secureCookies bool
	bootstrap     map[string]bool
	access        config.Access
	idle          time.Duration
	absolute      time.Duration
	csrfKey       []byte
	now           func() time.Time
}

func NewService(db *sql.DB, idp IdentityProvider, cfg *config.Config, log *slog.Logger) *Service {
	boot := make(map[string]bool, len(cfg.BootstrapAdminEmails))
	for _, e := range cfg.BootstrapAdminEmails {
		boot[strings.ToLower(e)] = true
	}
	return &Service{
		db:            db,
		q:             store.New(db),
		idp:           idp,
		log:           log,
		origin:        cfg.AppOrigin.String(),
		secureCookies: cfg.SecureCookies(),
		bootstrap:     boot,
		access:        cfg.Access,
		idle:          cfg.Session.IdleTimeout,
		absolute:      cfg.Session.AbsoluteTimeout,
		csrfKey:       cfg.Session.Secret,
		now:           func() time.Time { return time.Now().UTC().Truncate(time.Millisecond) },
	}
}

// SessionCookieName uses the __Host- prefix on HTTPS (requires Secure, Path=/, no Domain).
func (s *Service) SessionCookieName() string {
	if s.secureCookies {
		return "__Host-elm_session"
	}
	return "elm_session"
}

// randomToken returns 32 bytes of CSPRNG output, base64url encoded.
func randomToken() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func hashToken(raw string) []byte {
	h := sha256.Sum256([]byte(raw))
	return h[:]
}

// csrfToken derives the synchronizer token from the session token, so nothing extra is stored.
func (s *Service) csrfToken(rawSession string) string {
	m := hmac.New(sha256.New, s.csrfKey)
	m.Write([]byte("csrf\x00"))
	m.Write([]byte(rawSession))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *Service) validCSRF(rawSession, token string) bool {
	if token == "" {
		return false
	}
	return hmac.Equal([]byte(s.csrfToken(rawSession)), []byte(token))
}
