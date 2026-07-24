package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testEncryptionKey = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
	t.Setenv("ESCALITE_ENCRYPTION_KEY", testEncryptionKey)
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "")
	t.Setenv("ESCALITE_ENCRYPTION_KEY", testEncryptionKey)

	_, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_DATABASE_URL")
	assert.Contains(t, err.Error(), ".env.example")
}

func TestLoadRequiresEncryptionKey(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
	t.Setenv("ESCALITE_ENCRYPTION_KEY", "")

	_, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_ENCRYPTION_KEY")
	assert.Contains(t, err.Error(), "openssl rand -hex 32")
	assert.Contains(t, err.Error(), "docs/specs/07-security-and-auth.md")
}

func TestLoadRejectsPlaceholderEncryptionKey(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
	t.Setenv("ESCALITE_ENCRYPTION_KEY", "placeholder")

	_, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_ENCRYPTION_KEY")
	assert.Contains(t, err.Error(), "placeholder")
}

func TestLoadSucceedsWithRequiredEnv(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, ":8081", cfg.ListenAddr)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, time.Minute, cfg.HeartbeatScanInterval)
	assert.Len(t, cfg.EncryptionKey, 32)
}

func TestLoadHeartbeatScanIntervalFromEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ESCALITE_HEARTBEAT_SCAN_INTERVAL", "30s")

	cfg, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.HeartbeatScanInterval)
}

func TestLoadRejectsInvalidHeartbeatScanInterval(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ESCALITE_HEARTBEAT_SCAN_INTERVAL", "not-a-duration")

	_, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_HEARTBEAT_SCAN_INTERVAL")
}
