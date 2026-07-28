package config

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"time"

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
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
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
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
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
		t.Setenv("ESCALITE_ENCRYPTION_KEY", "placeholder")

		_, err := Load(Options{
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_ENCRYPTION_KEY")
		assert.Contains(a, err.Error(), "placeholder")
	})
}

func TestLoadSucceedsWithRequiredEnv(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		setRequiredEnv(t)

		cfg, err := Load(Options{
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
			RequireDatabase:   true,
		})
		require.NoError(a, err)
		assert.Equal(a, ":8081", cfg.ListenAddr)
		assert.Equal(a, "info", cfg.LogLevel)
		assert.Equal(a, time.Minute, cfg.HeartbeatScanInterval)
		assert.Len(a, cfg.EncryptionKey, 32)
	})
}

func TestLoadHeartbeatScanIntervalFromEnv(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		setRequiredEnv(t)
		t.Setenv("ESCALITE_HEARTBEAT_SCAN_INTERVAL", "30s")

		cfg, err := Load(Options{
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
			RequireDatabase:   true,
		})
		require.NoError(a, err)
		assert.Equal(a, 30*time.Second, cfg.HeartbeatScanInterval)
	})
}

func TestLoadRejectsInvalidHeartbeatScanInterval(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		setRequiredEnv(t)
		t.Setenv("ESCALITE_HEARTBEAT_SCAN_INTERVAL", "not-a-duration")

		_, err := Load(Options{
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
			RequireDatabase:   true,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_HEARTBEAT_SCAN_INTERVAL")
	})
}

func TestLoadSMTPRequiresFromWhenHostSet(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		setRequiredEnv(t)
		t.Setenv("ESCALITE_SMTP_HOST", "smtp.example.com")

		_, err := Load(Options{
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
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
			ServiceName:       "engine",
			DefaultListenAddr: ":8081",
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
