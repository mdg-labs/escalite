package audit_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
)

func TestRequestMetaFromHTTP(t *testing.T) {
	t.Run("uses forwarded for header", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/login", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.10, 198.51.100.1")
		req.Header.Set("User-Agent", "test-agent/1.0")
		req.RemoteAddr = "127.0.0.1:12345"

		meta := audit.RequestMetaFromHTTP(req)
		require.Equal(t, "203.0.113.10", meta.IP)
		require.Equal(t, "test-agent/1.0", meta.UserAgent)
	})

	t.Run("strips port from remote addr", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/login", nil)
		req.RemoteAddr = "198.51.100.42:54321"

		meta := audit.RequestMetaFromHTTP(req)
		require.Equal(t, "198.51.100.42", meta.IP)
	})
}
