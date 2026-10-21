package settings

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/g-asiapacific/elm/backend/internal/config"
	"github.com/g-asiapacific/elm/backend/internal/database"
	"github.com/g-asiapacific/elm/backend/internal/httpx"
	"github.com/g-asiapacific/elm/backend/internal/store"
)

type EmailSettings struct {
	Mode           string `json:"mode"` // OFF | LIVE
	SMTPConfigured bool   `json:"smtp_configured"`
	SMTPFrom       string `json:"smtp_from"`
	// LiveEnabled mirrors EMAIL_LIVE_ENABLED; sending cannot be turned on without it.
	LiveEnabled bool `json:"live_enabled"`
	// The server's ALLOWED_EMAILS / ALLOWED_EMAIL_DOMAIN (empty = no restriction).
	AllowedEmails []string `json:"allowed_emails"`
	AllowedDomain string   `json:"allowed_domain"`
}

type EmailInput struct {
	Version uint32 `json:"version"`
	Mode    string `json:"mode"`
}

// SetSMTP tells the service whether SMTP credentials exist in .env (only the sender address is shown).
func (s *Service) SetSMTP(configured bool, from string) {
	s.smtpConfigured, s.smtpFrom = configured, from
}

// SetLive passes EMAIL_LIVE_ENABLED and the access allowlist from .env.
func (s *Service) SetLive(enabled bool, access config.Access) {
	s.liveEnabled, s.access = enabled, access
}

func (s *Service) Email(ctx context.Context) (EmailSettings, error) {
	var mode string
	if err := s.db.QueryRowContext(ctx, "SELECT email_mode FROM company_settings WHERE id = 1").Scan(&mode); err != nil {
		return EmailSettings{}, err
	}
	emails := s.access.Emails
	if emails == nil {
		emails = []string{}
	}
	return EmailSettings{
		Mode: mode, SMTPConfigured: s.smtpConfigured, SMTPFrom: s.smtpFrom,
		LiveEnabled: s.liveEnabled, AllowedEmails: emails, AllowedDomain: s.access.Domain,
	}, nil
}

func (s *Service) UpdateEmail(ctx context.Context, actorID uint64, in EmailInput) error {
	v := httpx.Validation{}
	if in.Mode != "OFF" && in.Mode != "LIVE" {
		v.Add("mode", "Choose whether emails are sent.")
	}
	if in.Mode == "LIVE" && !s.smtpConfigured {
		v.Add("mode", "Configure SMTP_HOST, SMTP_PORT, SMTP_USERNAME, SMTP_PASSWORD and SMTP_FROM in .env first.")
	}
	if in.Mode == "LIVE" && !s.liveEnabled {
		v.Add("mode", "Sending is locked: set EMAIL_LIVE_ENABLED=true in the server .env and restart.")
	}
	if err := v.Err(); err != nil {
		return err
	}
	before, _ := s.Email(ctx)
	return database.InTx(ctx, s.db, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			`UPDATE company_settings SET email_mode = ?, updated_by = ?, version = version + 1 WHERE id = 1 AND version = ?`,
			in.Mode, actorID, in.Version)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n != 1 {
			return httpx.ErrConflict
		}
		b, _ := json.Marshal(map[string]any{"mode": before.Mode})
		a, _ := json.Marshal(map[string]any{"mode": in.Mode})
		return s.q.WithTx(tx).InsertSettingsEvent(ctx, store.InsertSettingsEventParams{
			ActorID: sql.NullInt64{Int64: int64(actorID), Valid: true}, Action: "EMAIL_SETTINGS_UPDATED", BeforeData: b, AfterData: a,
		})
	})
}
