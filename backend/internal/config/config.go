// Package config loads and validates runtime configuration from the environment (optionally a .env file).
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Env string

const (
	EnvDevelopment Env = "development"
	EnvProduction  Env = "production"
)

type DB struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	QueryTimeout    time.Duration
}

type Google struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	IssuerURL    string
}

// SMTP is used when an Admin selects TEST_EMAIL or LIVE. Credentials never leave the environment.
type SMTP struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func (s SMTP) Configured() bool { return s.Host != "" && s.Port > 0 && s.From != "" }

type Session struct {
	IdleTimeout     time.Duration
	AbsoluteTimeout time.Duration
	Secret          []byte // HMAC key for CSRF tokens
}

type Config struct {
	Env       Env
	AppOrigin *url.URL
	HTTPAddr  string
	LogLevel  slog.Level
	DB        DB
	Google    Google
	Session   Session
	SMTP      SMTP
	// EmailLive unlocks the LIVE delivery mode (mail to the real recipients). Off unless EMAIL_LIVE_ENABLED=true.
	EmailLive bool
	// Access limits who may sign in and who may receive LIVE mail (empty = anyone).
	Access Access
	// Lower-cased verified Google emails that become ADMIN when their account is first created.
	BootstrapAdminEmails []string
}

// Access is an allowlist of exact addresses and/or one domain. Empty means no restriction.
type Access struct {
	Emails []string // lower-cased
	Domain string   // lower-cased, without "@"
}

func (a Access) Restricted() bool { return len(a.Emails) > 0 || a.Domain != "" }

// Allows reports whether an address may sign in / receive LIVE mail.
func (a Access) Allows(email string) bool {
	if !a.Restricted() {
		return true
	}
	e := strings.ToLower(strings.TrimSpace(email))
	if a.Domain != "" && strings.HasSuffix(e, "@"+a.Domain) {
		return true
	}
	return slices.Contains(a.Emails, e)
}

// SecureCookies reports whether cookies must carry the Secure attribute (HTTPS origin).
func (c *Config) SecureCookies() bool { return c.AppOrigin.Scheme == "https" }

// ValidateServer checks settings only the HTTP server needs (migrate/seed run without them).
func (c *Config) ValidateServer() error {
	var errs []error
	if c.Google.ClientID == "" || c.Google.ClientSecret == "" {
		errs = append(errs, errors.New("GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are required"))
	}
	if want := c.AppOrigin.String() + "/auth/google/callback"; c.Google.RedirectURL != want {
		errs = append(errs, fmt.Errorf("GOOGLE_REDIRECT_URL must be %s", want))
	}
	if len(c.Session.Secret) < 32 {
		errs = append(errs, errors.New("SESSION_SECRET must be base64 of at least 32 random bytes"))
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid server configuration: %w", errors.Join(errs...))
	}
	return nil
}

// LoadDotEnv loads the first .env file found. Existing environment variables always win.
// ENV_FILE overrides the search; otherwise ./.env then ../.env (commands usually run from backend/).
func LoadDotEnv() (string, error) {
	candidates := []string{".env", "../.env"}
	if p := os.Getenv("ENV_FILE"); p != "" {
		candidates = []string{p}
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, godotenv.Load(p)
		}
	}
	return "", nil
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	return load(os.Getenv)
}

func load(get func(string) string) (*Config, error) {
	r := reader{get: get}
	cfg := &Config{
		Env:      Env(r.str("APP_ENV", string(EnvDevelopment))),
		HTTPAddr: r.str("HTTP_ADDR", "127.0.0.1:8080"),
		DB: DB{
			Host:            r.str("DB_HOST", "127.0.0.1"),
			Port:            r.int("DB_PORT", 3306),
			User:            r.str("DB_USER", "root"),
			Password:        get("DB_PASSWORD"),
			Name:            r.str("DB_NAME", "employee_leave_management"),
			MaxOpenConns:    r.int("DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    r.int("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: r.duration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
			QueryTimeout:    r.duration("DB_QUERY_TIMEOUT", 5*time.Second),
		},
		Google: Google{
			ClientID:     strings.TrimSpace(get("GOOGLE_CLIENT_ID")),
			ClientSecret: strings.TrimSpace(get("GOOGLE_CLIENT_SECRET")),
			RedirectURL:  r.str("GOOGLE_REDIRECT_URL", "http://localhost:5173/auth/google/callback"),
			IssuerURL:    "https://accounts.google.com",
		},
		SMTP: SMTP{
			Host: strings.TrimSpace(get("SMTP_HOST")), Port: r.int("SMTP_PORT", 587),
			Username: strings.TrimSpace(get("SMTP_USERNAME")), Password: get("SMTP_PASSWORD"),
			From: strings.TrimSpace(get("SMTP_FROM")),
		},
		Session: Session{
			IdleTimeout:     r.duration("SESSION_IDLE_TIMEOUT", 30*time.Minute),
			AbsoluteTimeout: r.duration("SESSION_ABSOLUTE_TIMEOUT", 12*time.Hour),
		},
	}

	for _, e := range strings.Split(get("BOOTSTRAP_ADMIN_EMAILS"), ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			cfg.BootstrapAdminEmails = append(cfg.BootstrapAdminEmails, e)
		}
	}
	cfg.EmailLive = strings.EqualFold(strings.TrimSpace(get("EMAIL_LIVE_ENABLED")), "true")
	for _, e := range strings.Split(get("ALLOWED_EMAILS"), ",") {
		if e = strings.ToLower(strings.TrimSpace(e)); e != "" {
			if !strings.Contains(e, "@") {
				r.fail("ALLOWED_EMAILS must be a comma-separated list of email addresses")
			}
			cfg.Access.Emails = append(cfg.Access.Emails, e)
		}
	}
	if d := strings.ToLower(strings.TrimPrefix(strings.TrimSpace(get("ALLOWED_EMAIL_DOMAIN")), "@")); d != "" {
		if !strings.Contains(d, ".") || strings.ContainsAny(d, "@ ,") {
			r.fail("ALLOWED_EMAIL_DOMAIN must be a domain such as g-asiapac.com.vn")
		}
		cfg.Access.Domain = d
	}
	if s := strings.TrimSpace(get("SESSION_SECRET")); s != "" {
		key, err := base64.StdEncoding.DecodeString(s)
		if err != nil {
			r.fail("SESSION_SECRET must be valid base64")
		}
		cfg.Session.Secret = key
	}
	if cfg.Session.IdleTimeout < time.Minute || cfg.Session.AbsoluteTimeout < cfg.Session.IdleTimeout || cfg.Session.AbsoluteTimeout > 7*24*time.Hour {
		r.fail("SESSION_IDLE_TIMEOUT must be >= 1m and <= SESSION_ABSOLUTE_TIMEOUT (max 168h)")
	}

	switch cfg.Env {
	case EnvDevelopment, EnvProduction:
	default:
		r.fail("APP_ENV must be development or production")
	}

	origin, err := url.Parse(r.str("APP_ORIGIN", "http://localhost:5173"))
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" || (origin.Path != "" && origin.Path != "/") {
		r.fail("APP_ORIGIN must be an absolute http(s) origin without path, e.g. http://localhost:5173")
	} else {
		origin.Path = ""
		cfg.AppOrigin = origin
	}
	if cfg.Env == EnvProduction && cfg.AppOrigin != nil && cfg.AppOrigin.Scheme != "https" {
		r.fail("APP_ORIGIN must use https in production")
	}

	if err := cfg.LogLevel.UnmarshalText([]byte(r.str("LOG_LEVEL", "info"))); err != nil {
		r.fail("LOG_LEVEL must be debug, info, warn or error")
	}

	if cfg.DB.Port < 1 || cfg.DB.Port > 65535 {
		r.fail("DB_PORT is out of range")
	}
	if cfg.DB.Name == "" || strings.ContainsAny(cfg.DB.Name, "`'\"/\\ ") {
		r.fail("DB_NAME is empty or contains invalid characters")
	}
	if cfg.DB.MaxOpenConns < 1 || cfg.DB.MaxIdleConns < 0 || cfg.DB.MaxIdleConns > cfg.DB.MaxOpenConns {
		r.fail("DB_MAX_OPEN_CONNS must be >= 1 and DB_MAX_IDLE_CONNS between 0 and DB_MAX_OPEN_CONNS")
	}
	if cfg.DB.QueryTimeout <= 0 {
		r.fail("DB_QUERY_TIMEOUT must be positive")
	}

	if len(r.errs) > 0 {
		return nil, fmt.Errorf("invalid configuration: %w", errors.Join(r.errs...))
	}
	return cfg, nil
}

type reader struct {
	get  func(string) string
	errs []error
}

func (r *reader) fail(msg string) { r.errs = append(r.errs, errors.New(msg)) }

func (r *reader) str(key, def string) string {
	if v := strings.TrimSpace(r.get(key)); v != "" {
		return v
	}
	return def
}

func (r *reader) int(key string, def int) int {
	v := strings.TrimSpace(r.get(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		r.fail(key + " must be an integer")
		return def
	}
	return n
}

func (r *reader) duration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(r.get(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		r.fail(key + " must be a duration such as 30s, 5m or 12h")
		return def
	}
	return d
}
