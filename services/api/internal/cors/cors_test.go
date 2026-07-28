package cors

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/allure-framework/allure-go/testify/assert"
	"github.com/allure-framework/allure-go/testify/require"
)

func TestMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	t.Parallel()
	allure.Wrap(t, func(a *allure.Context) {

		handler := Middleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.Header.Set("Origin", "http://localhost:5173")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Equal(a, "http://localhost:5173", rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(a, "true", rec.Header().Get("Access-Control-Allow-Credentials"))
		assert.Equal(a, http.StatusOK, rec.Code)
	})
}

func TestMiddlewareRejectsOtherOrigins(t *testing.T) {
	t.Parallel()
	allure.Wrap(t, func(a *allure.Context) {

		handler := Middleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
		req.Header.Set("Origin", "http://evil.example")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		assert.Empty(a, rec.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(a, http.StatusOK, rec.Code)
	})
}

func TestMiddlewareHandlesPreflight(t *testing.T) {
	t.Parallel()
	allure.Wrap(t, func(a *allure.Context) {

		handler := Middleware("http://localhost:5173")(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			t.Fatal("next handler should not run for OPTIONS")
		}))

		req := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
		req.Header.Set("Origin", "http://localhost:5173")
		req.Header.Set("Access-Control-Request-Method", "POST")

		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		require.Equal(a, http.StatusNoContent, rec.Code)
		assert.Equal(a, "http://localhost:5173", rec.Header().Get("Access-Control-Allow-Origin"))
	})
}
