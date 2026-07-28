package log

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestWithContextAddsAvailableKeys(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		var buf bytes.Buffer
		base := slog.New(slog.NewJSONHandler(&buf, nil)).With("service", "engine")

		logger := WithContext(base, Context{
			OrgID:     "org-abc",
			ServiceID: "svc-def",
		})
		logger.Info("job started")

		var entry map[string]any
		require.NoError(a, json.Unmarshal(buf.Bytes(), &entry))
		assert.Equal(a, "org-abc", entry["org_id"])
		assert.Equal(a, "svc-def", entry["service_id"])
	})
}
