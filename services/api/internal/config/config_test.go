package config

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
	t.Setenv("ESCALITE_ENCRYPTION_KEY", testEncryptionKey)
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_DATABASE_URL", "")
		t.Setenv("ESCALITE_ENCRYPTION_KEY", testEncryptionKey)

		_, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_DATABASE_URL")
		assert.Contains(a, err.Error(), ".env.example")
	})
}

func TestLoadRequiresEncryptionKey(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
		t.Setenv("ESCALITE_ENCRYPTION_KEY", "")

		_, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_ENCRYPTION_KEY")
		assert.Contains(a, err.Error(), "openssl rand -hex 32")
		assert.Contains(a, err.Error(), "docs/specs/07-security-and-auth.md")
	})
}

func TestLoadRejectsPlaceholderEncryptionKey(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
		t.Setenv("ESCALITE_ENCRYPTION_KEY", "changeme")

		_, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_ENCRYPTION_KEY")
		assert.Contains(a, err.Error(), "placeholder")
	})
}

func TestLoadRejectsAllZeroEncryptionKey(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
		t.Setenv("ESCALITE_ENCRYPTION_KEY", "0000000000000000000000000000000000000000000000000000000000000000")

		_, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_ENCRYPTION_KEY")
		assert.Contains(a, err.Error(), "all zeros")
	})
}

func TestLoadSucceedsWithRequiredEnv(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)
		t.Setenv("ESCALITE_HTTP_ADDR", ":9090")
		t.Setenv("ESCALITE_LOG_LEVEL", "debug")

		cfg, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.NoError(a, err)
		assert.Equal(a, ":9090", cfg.ListenAddr)
		assert.Equal(a, "debug", cfg.LogLevel)
		assert.Equal(a, "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable", cfg.DatabaseURL)
		assert.Len(a, cfg.EncryptionKey, 32)
	})
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)
		t.Setenv("ESCALITE_LOG_LEVEL", "trace")

		_, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_LOG_LEVEL")
	})
}

func TestLoadSMTPOptional(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)

		cfg, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.NoError(a, err)
		assert.Nil(a, cfg.SMTP)
	})
}

func TestLoadSMTPRequiresFromWhenHostSet(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)
		t.Setenv("ESCALITE_SMTP_HOST", "smtp.example.com")

		_, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_SMTP_FROM")
	})
}

func TestLoadSMTPParsesConfig(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)
		t.Setenv("ESCALITE_SMTP_HOST", "smtp.example.com")
		t.Setenv("ESCALITE_SMTP_PORT", "2525")
		t.Setenv("ESCALITE_SMTP_USERNAME", "user")
		t.Setenv("ESCALITE_SMTP_PASSWORD", "secret")
		t.Setenv("ESCALITE_SMTP_FROM", "noreply@example.com")

		cfg, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.NoError(a, err)
		require.NotNil(a, cfg.SMTP)
		assert.Equal(a, "smtp.example.com", cfg.SMTP.Host)
		assert.Equal(a, 2525, cfg.SMTP.Port)
		assert.Equal(a, "user", cfg.SMTP.Username)
		assert.Equal(a, "secret", cfg.SMTP.Password)
		assert.Equal(a, "noreply@example.com", cfg.SMTP.From)
	})
}

func TestLoadInboundEmailOptional(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)

		cfg, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.NoError(a, err)
		assert.Empty(a, cfg.InboundEmail.RelaySecret)
	})
}

func TestLoadInboundEmailRequiresDomainWhenSecretSet(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)
		t.Setenv("ESCALITE_INBOUND_EMAIL_RELAY_SECRET", "relay-secret")

		_, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_INBOUND_EMAIL_DOMAIN")
	})
}

func TestLoadInboundEmailParsesConfig(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		setRequiredEnv(t)
		t.Setenv("ESCALITE_INBOUND_EMAIL_RELAY_SECRET", "relay-secret")
		t.Setenv("ESCALITE_INBOUND_EMAIL_DOMAIN", "inbound.example.com")
		t.Setenv("ESCALITE_INBOUND_EMAIL_REQUIRE_AUTHENTICATED", "false")

		cfg, err := Load(Options{
			ServiceName:       "api",
			DefaultListenAddr: ":8080",
			RequireDatabase:   true,
		})
		require.NoError(a, err)
		assert.Equal(a, "relay-secret", cfg.InboundEmail.RelaySecret)
		assert.Equal(a, "inbound.example.com", cfg.InboundEmail.Domain)
		assert.False(a, cfg.InboundEmail.RequireAuthenticated)
	})
}
