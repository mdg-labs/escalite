package config

import (
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestLoadUsesDefaultsWithoutDatabase(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		t.Setenv("ESCALITE_DATABASE_URL", "")
		t.Setenv("ESCALITE_HTTP_ADDR", "")

		cfg, err := Load(Options{
			ServiceName:       "integrations",
			DefaultListenAddr: ":8082",
			RequireDatabase:   false,
		})
		require.NoError(a, err)
		assert.Equal(a, ":8082", cfg.ListenAddr)
		assert.Equal(a, "info", cfg.LogLevel)
		assert.Empty(a, cfg.DatabaseURL)
	})
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		t.Setenv("ESCALITE_LOG_LEVEL", "verbose")

		_, err := Load(Options{
			ServiceName:       "integrations",
			DefaultListenAddr: ":8082",
			RequireDatabase:   false,
		})
		require.Error(a, err)
		assert.Contains(a, err.Error(), "ESCALITE_LOG_LEVEL")
	})
}
