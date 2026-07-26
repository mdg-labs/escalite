package testutil

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

// MockServer is a minimal OIDC provider for integration tests.
type MockServer struct {
	Server   *httptest.Server
	URL      string
	Email    string
	Subject  string
	ClientID string
	key      *rsa.PrivateKey
}

// NewMockServer starts a fake OIDC IdP that issues ID tokens for the given email.
func NewMockServer(t *testing.T, clientID, clientSecret, redirectURL, email string) *MockServer {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	mock := &MockServer{
		Email:    email,
		Subject:  "oidc-subject-" + email,
		ClientID: clientID,
		key:      key,
	}

	mux := http.NewServeMux()
	mock.Server = httptest.NewServer(mux)
	mock.URL = mock.Server.URL
	issuer := mock.URL

	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":                                issuer,
			"authorization_endpoint":                issuer + "/authorize",
			"token_endpoint":                        issuer + "/token",
			"jwks_uri":                              issuer + "/jwks",
			"response_types_supported":              []string{"code"},
			"subject_types_supported":               []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"},
		})
	})

	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{
				{
					"kty": "RSA",
					"kid": "test-key",
					"use": "sig",
					"alg": "RS256",
					"n":   base64.RawURLEncoding.EncodeToString(mock.key.N.Bytes()),
					"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(mock.key.E)).Bytes()),
				},
			},
		})
	})

	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		state := r.URL.Query().Get("state")
		target := redirectURL + "?code=test-auth-code&state=" + state
		http.Redirect(w, r, target, http.StatusFound)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if r.FormValue("client_id") != clientID || r.FormValue("client_secret") != clientSecret {
			http.Error(w, "invalid client", http.StatusUnauthorized)
			return
		}
		if r.FormValue("code") != "test-auth-code" {
			http.Error(w, "invalid code", http.StatusBadRequest)
			return
		}

		idToken, err := signIDToken(mock.key, issuer, clientID, mock.Email, mock.Subject)
		require.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"access_token": "access-token",
			"token_type":   "Bearer",
			"id_token":     idToken,
		})
	})

	return mock
}

// Close shuts down the mock IdP.
func (m *MockServer) Close() {
	m.Server.Close()
}

func signIDToken(key *rsa.PrivateKey, issuer, audience, email, subject string) (string, error) {
	claims := jwt.MapClaims{
		"iss":            issuer,
		"sub":            subject,
		"aud":            audience,
		"exp":            time.Now().Add(time.Hour).Unix(),
		"iat":            time.Now().Unix(),
		"email":          email,
		"email_verified": true,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "test-key"
	return token.SignedString(key)
}
