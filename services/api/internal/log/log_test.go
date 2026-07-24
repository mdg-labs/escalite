package log

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithContextAddsAvailableKeys(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil)).With("service", "api")

	logger := WithContext(base, Context{
		OrgID:     "org-123",
		ServiceID: "svc-456",
	})
	logger.Info("test message")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "org-123", entry["org_id"])
	assert.Equal(t, "svc-456", entry["service_id"])
}

func TestWithContextOmitsEmptyKeys(t *testing.T) {
	var buf bytes.Buffer
	base := slog.New(slog.NewJSONHandler(&buf, nil)).With("service", "api")

	logger := WithContext(base, Context{OrgID: "org-123"})
	logger.Info("test message")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "org-123", entry["org_id"])
	_, hasServiceID := entry["service_id"]
	assert.False(t, hasServiceID)
}
