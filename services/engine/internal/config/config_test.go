package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "")

	_, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_DATABASE_URL")
	assert.Contains(t, err.Error(), ".env.example")
}

func TestLoadSucceedsWithRequiredEnv(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "postgres://escalite:escalite@localhost:5432/escalite?sslmode=disable")

	cfg, err := Load(Options{
		ServiceName:       "engine",
		DefaultListenAddr: ":8081",
		RequireDatabase:   true,
	})
	require.NoError(t, err)
	assert.Equal(t, ":8081", cfg.ListenAddr)
	assert.Equal(t, "info", cfg.LogLevel)
}
