package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/g-asiapacific/elm/backend/internal/config"
)

// OIDCProvider is a standard OpenID Connect Authorization Code + PKCE client (Google in production).
// Discovery is lazy so the server can start while offline; it is retried on the next login.
type OIDCProvider struct {
	cfg    config.Google
	client *http.Client

	mu       sync.Mutex
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

func NewOIDCProvider(cfg config.Google) *OIDCProvider {
	return &OIDCProvider{cfg: cfg, client: &http.Client{Timeout: 10 * time.Second}}
}

func (p *OIDCProvider) init() (*oauth2.Config, *oidc.IDTokenVerifier, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.oauth != nil {
		return p.oauth, p.verifier, nil
	}
	// Background context on purpose: go-oidc keeps it for later JWKS refreshes. The client timeout bounds each call.
	ctx := oidc.ClientContext(context.Background(), p.client)
	provider, err := oidc.NewProvider(ctx, p.cfg.IssuerURL)
	if err != nil {
		return nil, nil, fmt.Errorf("oidc discovery: %w", err)
	}
	p.oauth = &oauth2.Config{
		ClientID:     p.cfg.ClientID,
		ClientSecret: p.cfg.ClientSecret,
		RedirectURL:  p.cfg.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
	}
	// Verifies signature (JWKS), issuer, audience (client ID) and expiry.
	p.verifier = provider.Verifier(&oidc.Config{ClientID: p.cfg.ClientID})
	return p.oauth, p.verifier, nil
}

func (p *OIDCProvider) AuthCodeURL(_ context.Context, state, nonce, verifier string) (string, error) {
	oc, _, err := p.init()
	if err != nil {
		return "", err
	}
	return oc.AuthCodeURL(state,
		oauth2.S256ChallengeOption(verifier),
		oidc.Nonce(nonce),
		oauth2.SetAuthURLParam("prompt", "select_account"),
	), nil
}

func (p *OIDCProvider) Exchange(ctx context.Context, code, verifier, nonce string) (Identity, error) {
	oc, v, err := p.init()
	if err != nil {
		return Identity{}, err
	}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, p.client)
	tok, err := oc.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return Identity{}, fmt.Errorf("token exchange: %w", err)
	}
	rawID, ok := tok.Extra("id_token").(string)
	if !ok || rawID == "" {
		return Identity{}, errors.New("token response has no id_token")
	}
	idToken, err := v.Verify(ctx, rawID)
	if err != nil {
		return Identity{}, fmt.Errorf("verify id_token: %w", err)
	}
	if subtle.ConstantTimeCompare([]byte(idToken.Nonce), []byte(nonce)) != 1 {
		return Identity{}, errors.New("id_token nonce mismatch")
	}
	var claims struct {
		Email         string    `json:"email"`
		EmailVerified claimBool `json:"email_verified"`
		Name          string    `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return Identity{}, fmt.Errorf("decode claims: %w", err)
	}
	return Identity{
		Subject:       idToken.Subject,
		Email:         claims.Email,
		Name:          claims.Name,
		EmailVerified: bool(claims.EmailVerified),
	}, nil
}

// claimBool accepts both true and "true" (some providers send booleans as strings).
type claimBool bool

func (b *claimBool) UnmarshalJSON(data []byte) error {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch t := v.(type) {
	case bool:
		*b = claimBool(t)
	case string:
		*b = claimBool(t == "true")
	default:
		*b = false
	}
	return nil
}
