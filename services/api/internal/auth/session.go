package auth

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	// SessionCookieName is the HttpOnly cookie storing the server-side session ID.
	SessionCookieName = "escalite_session"
	// DefaultSessionTTL is how long a web session remains valid without re-login.
	DefaultSessionTTL = 30 * 24 * time.Hour
)

// SetSessionCookie writes the session cookie for a newly authenticated browser client.
func SetSessionCookie(w http.ResponseWriter, sessionID uuid.UUID, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID.String(),
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}
