package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "")

	_, err := Load(Options{
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_DATABASE_URL")
	assert.Contains(t, err.Error(), ".env.example")
}

func TestLoadSucceedsWithRequiredEnv(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")
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
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://localhost/escalite")
	t.Setenv("ESCALITE_LOG_LEVEL", "trace")

	_, err := Load(Options{
		ServiceName:       "api",
		DefaultListenAddr: ":8080",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_LOG_LEVEL")
}
