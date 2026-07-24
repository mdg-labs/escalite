package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadOIDCDisabledWhenUnset(t *testing.T) {
	t.Setenv("ESCALITE_OIDC_ISSUER_URL", "")
	t.Setenv("ESCALITE_OIDC_CLIENT_ID", "")
	t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "")

	cfg, err := loadOIDC()
	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestLoadOIDCRequiresAllCredentials(t *testing.T) {
	t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
	t.Setenv("ESCALITE_OIDC_CLIENT_ID", "")
	t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "")

	_, err := loadOIDC()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "partial OIDC configuration")
}

func TestLoadOIDCRequiresRedirectOrPublicURL(t *testing.T) {
	t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
	t.Setenv("ESCALITE_OIDC_CLIENT_ID", "client-id")
	t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "client-secret")
	t.Setenv("ESCALITE_OIDC_REDIRECT_URL", "")
	t.Setenv("ESCALITE_PUBLIC_URL", "")

	_, err := loadOIDC()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redirect URL unknown")
}

func TestLoadOIDCDerivesRedirectFromPublicURL(t *testing.T) {
	t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
	t.Setenv("ESCALITE_OIDC_CLIENT_ID", "client-id")
	t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "client-secret")
	t.Setenv("ESCALITE_OIDC_REDIRECT_URL", "")
	t.Setenv("ESCALITE_PUBLIC_URL", "https://escalite.example.com/")

	cfg, err := loadOIDC()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "https://escalite.example.com/api/v1/auth/oidc/callback", cfg.RedirectURL)
	assert.Equal(t, "https://escalite.example.com", cfg.SuccessURL)
}

func TestLoadOIDCUsesExplicitRedirectURL(t *testing.T) {
	t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
	t.Setenv("ESCALITE_OIDC_CLIENT_ID", "client-id")
	t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "client-secret")
	t.Setenv("ESCALITE_OIDC_REDIRECT_URL", "https://escalite.example.com/api/v1/auth/oidc/callback")
	t.Setenv("ESCALITE_PUBLIC_URL", "https://escalite.example.com")

	cfg, err := loadOIDC()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "https://escalite.example.com/api/v1/auth/oidc/callback", cfg.RedirectURL)
}

func TestLoadIncludesOIDCWhenConfigured(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
	t.Setenv("ESCALITE_OIDC_CLIENT_ID", "client-id")
	t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "client-secret")
	t.Setenv("ESCALITE_PUBLIC_URL", "https://escalite.example.com")

	cfg, err := Load(Options{
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.NoError(t, err)
	require.NotNil(t, cfg.OIDC)
	assert.Equal(t, "client-id", cfg.OIDC.ClientID)
}
