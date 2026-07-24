package oidc

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

const stateCookieName = "escalite_oidc_state"

// GenerateState returns a cryptographically random CSRF state value.
func GenerateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate oidc state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SetStateCookie stores the OIDC CSRF state in a short-lived HttpOnly cookie.
func SetStateCookie(w http.ResponseWriter, state string) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Path:     "/api/v1/auth/oidc",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

// ValidateStateCookie compares the callback state query param with the stored cookie.
func ValidateStateCookie(r *http.Request, callbackState string) bool {
	cookie, err := r.Cookie(stateCookieName)
	if err != nil || cookie.Value == "" || callbackState == "" {
		return false
	}
	return cookie.Value == callbackState
}

// ClearStateCookie removes the OIDC CSRF state cookie.
func ClearStateCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     stateCookieName,
		Value:    "",
		Path:     "/api/v1/auth/oidc",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
