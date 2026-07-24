package cors

import (
	"net/http"
	"strings"
)

// Middleware restricts cross-origin browser access to the configured app origin.
func Middleware(appOrigin string) func(http.Handler) http.Handler {
	allowedOrigin := strings.TrimSuffix(strings.TrimSpace(appOrigin), "/")

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if allowedOrigin != "" && origin == allowedOrigin {
				w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set(
					"Access-Control-Allow-Headers",
					"Accept, Authorization, Content-Type, X-Request-ID",
				)
				w.Header().Set("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
