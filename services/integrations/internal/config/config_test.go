package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUsesDefaultsWithoutDatabase(t *testing.T) {
	t.Setenv("ESCALITE_DATABASE_URL", "")
	t.Setenv("ESCALITE_HTTP_ADDR", "")

	cfg, err := Load(Options{
		ServiceName:       "integrations",
		DefaultListenAddr: ":8082",
		RequireDatabase:   false,
	})
	require.NoError(t, err)
	assert.Equal(t, ":8082", cfg.ListenAddr)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Empty(t, cfg.DatabaseURL)
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	t.Setenv("ESCALITE_LOG_LEVEL", "verbose")

	_, err := Load(Options{
		ServiceName:       "integrations",
		DefaultListenAddr: ":8082",
		RequireDatabase:   false,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ESCALITE_LOG_LEVEL")
}
