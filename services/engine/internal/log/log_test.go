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
	base := slog.New(slog.NewJSONHandler(&buf, nil)).With("service", "engine")

	logger := WithContext(base, Context{
		OrgID:     "org-abc",
		ServiceID: "svc-def",
	})
	logger.Info("job started")

	var entry map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &entry))
	assert.Equal(t, "org-abc", entry["org_id"])
	assert.Equal(t, "svc-def", entry["service_id"])
}
