package config

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestLoadOIDCDisabledWhenUnset(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_OIDC_ISSUER_URL", "")
		t.Setenv("ESCALITE_OIDC_CLIENT_ID", "")
		t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "")

		cfg, err := loadOIDC()
		require.NoError(a, err)
		assert.Nil(a, cfg)
	})
}

func TestLoadOIDCRequiresAllCredentials(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
		t.Setenv("ESCALITE_OIDC_CLIENT_ID", "")
		t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "")

		_, err := loadOIDC()
		require.Error(a, err)
		assert.Contains(a, err.Error(), "partial OIDC configuration")
	})
}

func TestLoadOIDCRequiresRedirectOrPublicURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
		t.Setenv("ESCALITE_OIDC_CLIENT_ID", "client-id")
		t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "client-secret")
		t.Setenv("ESCALITE_OIDC_REDIRECT_URL", "")
		t.Setenv("ESCALITE_PUBLIC_URL", "")

		_, err := loadOIDC()
		require.Error(a, err)
		assert.Contains(a, err.Error(), "redirect URL unknown")
	})
}

func TestLoadOIDCDerivesRedirectFromPublicURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
		t.Setenv("ESCALITE_OIDC_CLIENT_ID", "client-id")
		t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "client-secret")
		t.Setenv("ESCALITE_OIDC_REDIRECT_URL", "")
		t.Setenv("ESCALITE_PUBLIC_URL", "https://escalite.example.com/")

		cfg, err := loadOIDC()
		require.NoError(a, err)
		require.NotNil(a, cfg)
		assert.Equal(a, "https://escalite.example.com/api/v1/auth/oidc/callback", cfg.RedirectURL)
		assert.Equal(a, "https://escalite.example.com", cfg.SuccessURL)
	})
}

func TestLoadOIDCUsesExplicitRedirectURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_OIDC_ISSUER_URL", "https://idp.example.com")
		t.Setenv("ESCALITE_OIDC_CLIENT_ID", "client-id")
		t.Setenv("ESCALITE_OIDC_CLIENT_SECRET", "client-secret")
		t.Setenv("ESCALITE_OIDC_REDIRECT_URL", "https://escalite.example.com/api/v1/auth/oidc/callback")
		t.Setenv("ESCALITE_PUBLIC_URL", "https://escalite.example.com")

		cfg, err := loadOIDC()
		require.NoError(a, err)
		require.NotNil(a, cfg)
		assert.Equal(a, "https://escalite.example.com/api/v1/auth/oidc/callback", cfg.RedirectURL)
	})
}

func TestLoadIncludesOIDCWhenConfigured(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
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
		require.NoError(a, err)
		require.NotNil(a, cfg.OIDC)
		assert.Equal(a, "client-id", cfg.OIDC.ClientID)
	})
}
