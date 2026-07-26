package saml

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
)

const (
	requestIDCookieName = "escalite_saml_request_id"
	relayStateCookieName = "escalite_saml_relay_state"
)

// GenerateRelayState returns a cryptographically random relay state value.
func GenerateRelayState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate saml relay state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SetFlowCookies stores the AuthnRequest ID and relay state for ACS validation.
func SetFlowCookies(w http.ResponseWriter, requestID, relayState string) {
	http.SetCookie(w, &http.Cookie{
		Name:     requestIDCookieName,
		Value:    requestID,
		Path:     "/api/v1/auth/saml",
		MaxAge:   int((10 * time.Minute).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
	if relayState != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     relayStateCookieName,
			Value:    relayState,
			Path:     "/api/v1/auth/saml",
			MaxAge:   int((10 * time.Minute).Seconds()),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

// FlowCookies returns the stored AuthnRequest ID and relay state cookie values.
func FlowCookies(r *http.Request) (requestID string, relayState string) {
	if cookie, err := r.Cookie(requestIDCookieName); err == nil {
		requestID = cookie.Value
	}
	if cookie, err := r.Cookie(relayStateCookieName); err == nil {
		relayState = cookie.Value
	}
	return requestID, relayState
}

// ClearFlowCookies removes SAML flow cookies after callback handling.
func ClearFlowCookies(w http.ResponseWriter) {
	for _, name := range []string{requestIDCookieName, relayStateCookieName} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/api/v1/auth/saml",
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	}
}
