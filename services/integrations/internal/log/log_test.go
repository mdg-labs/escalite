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
		base := slog.New(slog.NewJSONHandler(&buf, nil)).With("service", "integrations")

		logger := WithContext(base, Context{OrgID: "org-789"})
		logger.Info("webhook received")

		var entry map[string]any
		require.NoError(a, json.Unmarshal(buf.Bytes(), &entry))
		assert.Equal(a, "org-789", entry["org_id"])
	})
}
