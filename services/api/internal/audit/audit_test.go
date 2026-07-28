package audit_test

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
)

func TestRequestMetaFromHTTP(t *testing.T) {
	allure.Test(t, "uses forwarded for header", func(a *allure.Context) {
		req := httptest.NewRequest("POST", "/api/v1/login", nil)
		req.Header.Set("X-Forwarded-For", "203.0.113.10, 198.51.100.1")
		req.Header.Set("User-Agent", "test-agent/1.0")
		req.RemoteAddr = "127.0.0.1:12345"

		meta := audit.RequestMetaFromHTTP(req)
		require.Equal(a, "203.0.113.10", meta.IP)
		require.Equal(a, "test-agent/1.0", meta.UserAgent)
	})
	allure.Test(t, "strips port from remote addr", func(a *allure.Context) {
		req := httptest.NewRequest("POST", "/api/v1/login", nil)
		req.RemoteAddr = "198.51.100.42:54321"

		meta := audit.RequestMetaFromHTTP(req)
		require.Equal(a, "198.51.100.42", meta.IP)
	})
}
