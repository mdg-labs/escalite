package config

import (
	"testing"

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
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
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
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_ENCRYPTION_KEY")
	assert.Contains(t, err.Error(), "openssl rand -hex 32")
	assert.Contains(t, err.Error(), "docs/specs/07-security-and-auth.md")
}

func TestLoadRejectsPlaceholderEncryptionKey(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
	t.Setenv("ESCALITE_ENCRYPTION_KEY", "changeme")

	_, err := Load(Options{
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_ENCRYPTION_KEY")
	assert.Contains(t, err.Error(), "placeholder")
}

func TestLoadRejectsAllZeroEncryptionKey(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
	t.Setenv("ESCALITE_ENCRYPTION_KEY", "0000000000000000000000000000000000000000000000000000000000000000")

	_, err := Load(Options{
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_ENCRYPTION_KEY")
	assert.Contains(t, err.Error(), "all zeros")
}

func TestLoadSucceedsWithRequiredEnv(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ESCALITE_HTTP_ADDR", ":9090")
	t.Setenv("ESCALITE_LOG_LEVEL", "debug")

	cfg, err := Load(Options{
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, ":9090", cfg.ListenAddr)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable", cfg.DatabaseURL)
	assert.Len(t, cfg.EncryptionKey, 32)
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("ESCALITE_LOG_LEVEL", "trace")

	_, err := Load(Options{
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_LOG_LEVEL")
}
