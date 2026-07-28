package log

import (
	"bytes"
	"encoding/json"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"log/slog"
	"testing"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestWithContextAddsAvailableKeys(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		var buf bytes.Buffer
		base := slog.New(slog.NewJSONHandler(&buf, nil)).With("service", "api")

		logger := WithContext(base, Context{
			OrgID:     "org-123",
			ServiceID: "svc-456",
		})
		logger.Info("test message")

		var entry map[string]any
		require.NoError(a, json.Unmarshal(buf.Bytes(), &entry))
		assert.Equal(a, "org-123", entry["org_id"])
		assert.Equal(a, "svc-456", entry["service_id"])
	})
}

func TestWithContextOmitsEmptyKeys(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		var buf bytes.Buffer
		base := slog.New(slog.NewJSONHandler(&buf, nil)).With("service", "api")

		logger := WithContext(base, Context{OrgID: "org-123"})
		logger.Info("test message")

		var entry map[string]any
		require.NoError(a, json.Unmarshal(buf.Bytes(), &entry))
		assert.Equal(a, "org-123", entry["org_id"])
		_, hasServiceID := entry["service_id"]
		assert.False(a, hasServiceID)
	})
}
