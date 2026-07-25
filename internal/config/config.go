// Package config loads getbud configuration from GETBUD_* environment
// variables. Environment variables are the only configuration mechanism.
package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	// Addr is the listen address, e.g. ":8080".
	Addr string
	// DatabaseURL is a postgres connection string.
	DatabaseURL string
	// BaseURL is the public URL of this instance, e.g.
	// "https://getbud.example.com". Required when Google auth is enabled;
	// OAuth redirect URIs are derived from it.
	BaseURL string
	// GoogleClientID / GoogleClientSecret enable "Sign in with Google" when
	// both are set.
	GoogleClientID     string
	GoogleClientSecret string
	// Generic OIDC provider (Authentik, Keycloak, …). All three of issuer,
	// client id, and client secret must be set together; Name labels the
	// login button (defaults to "SSO").
	OIDCIssuerURL    string
	OIDCClientID     string
	OIDCClientSecret string
	OIDCName         string
	// LocalAuth enables email/password sign-in (intended for dev/testing).
	LocalAuth bool
	// AllowedEmails restricts who may sign up beyond the first user. Empty
	// means the instance is closed after the first sign-in.
	AllowedEmails []string
}

// GoogleEnabled reports whether Google sign-in is configured.
func (c Config) GoogleEnabled() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

func Load() (Config, error) {
	cfg := Config{
		Addr:               getenv("GETBUD_ADDR", ":8080"),
		DatabaseURL:        os.Getenv("GETBUD_DATABASE_URL"),
		BaseURL:            strings.TrimSuffix(getenv("GETBUD_BASE_URL", "http://localhost:8080"), "/"),
		GoogleClientID:     os.Getenv("GETBUD_GOOGLE_CLIENT_ID"),
		GoogleClientSecret: os.Getenv("GETBUD_GOOGLE_CLIENT_SECRET"),
		// The OIDC issuer is an exact-match identifier (Authentik's ends
		// with a slash) — never normalize it; see #108.
		OIDCIssuerURL:    os.Getenv("GETBUD_OIDC_ISSUER_URL"),
		OIDCClientID:     os.Getenv("GETBUD_OIDC_CLIENT_ID"),
		OIDCClientSecret: os.Getenv("GETBUD_OIDC_CLIENT_SECRET"),
		OIDCName:         os.Getenv("GETBUD_OIDC_NAME"),
		LocalAuth:        os.Getenv("GETBUD_LOCAL_AUTH") == "true",
		AllowedEmails:    splitList(os.Getenv("GETBUD_ALLOWED_EMAILS")),
	}
	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("GETBUD_DATABASE_URL is required")
	}
	if (cfg.GoogleClientID == "") != (cfg.GoogleClientSecret == "") {
		return Config{}, fmt.Errorf("GETBUD_GOOGLE_CLIENT_ID and GETBUD_GOOGLE_CLIENT_SECRET must be set together")
	}
	oidcSet := 0
	for _, v := range []string{cfg.OIDCIssuerURL, cfg.OIDCClientID, cfg.OIDCClientSecret} {
		if v != "" {
			oidcSet++
		}
	}
	if oidcSet != 0 && oidcSet != 3 {
		return Config{}, fmt.Errorf("GETBUD_OIDC_ISSUER_URL, GETBUD_OIDC_CLIENT_ID, and GETBUD_OIDC_CLIENT_SECRET must be set together")
	}
	return cfg, nil
}

func splitList(s string) []string {
	var out []string
	for _, v := range strings.Split(s, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
