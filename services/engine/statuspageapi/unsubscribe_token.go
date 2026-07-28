package statuspageapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

const unsubscribeTokenVersion = "v1"

var (
	ErrInvalidUnsubscribeToken = errors.New("invalid unsubscribe token")
)

// UnsubscribeClaims identifies a status page email subscription.
type UnsubscribeClaims struct {
	SubscriptionID uuid.UUID
	OrganizationID uuid.UUID
}

// SignUnsubscribeToken returns a signed token for the given subscription.
func SignUnsubscribeToken(key []byte, claims UnsubscribeClaims) (string, error) {
	if len(key) == 0 {
		return "", fmt.Errorf("signing key is required")
	}
	if claims.SubscriptionID == uuid.Nil || claims.OrganizationID == uuid.Nil {
		return "", fmt.Errorf("subscription and organization ids are required")
	}

	message := fmt.Sprintf(
		"%s.%s.%s",
		unsubscribeTokenVersion,
		claims.SubscriptionID.String(),
		claims.OrganizationID.String(),
	)
	signature := signUnsubscribeMessage(key, message)
	return message + "." + signature, nil
}

// ParseUnsubscribeToken verifies and decodes a signed unsubscribe token.
func ParseUnsubscribeToken(key []byte, token string) (UnsubscribeClaims, error) {
	if len(key) == 0 {
		return UnsubscribeClaims{}, ErrInvalidUnsubscribeToken
	}

	token = strings.TrimSpace(token)
	parts := strings.Split(token, ".")
	if len(parts) != 4 || parts[0] != unsubscribeTokenVersion {
		return UnsubscribeClaims{}, ErrInvalidUnsubscribeToken
	}

	subscriptionID, err := uuid.Parse(parts[1])
	if err != nil {
		return UnsubscribeClaims{}, ErrInvalidUnsubscribeToken
	}
	organizationID, err := uuid.Parse(parts[2])
	if err != nil {
		return UnsubscribeClaims{}, ErrInvalidUnsubscribeToken
	}

	message := strings.Join(parts[:3], ".")
	expected := signUnsubscribeMessage(key, message)
	if !hmac.Equal([]byte(parts[3]), []byte(expected)) {
		return UnsubscribeClaims{}, ErrInvalidUnsubscribeToken
	}

	return UnsubscribeClaims{
		SubscriptionID: subscriptionID,
		OrganizationID: organizationID,
	}, nil
}

// BuildUnsubscribeURL returns the public status page unsubscribe link.
func BuildUnsubscribeURL(publicURL, token string) string {
	base := strings.TrimSuffix(strings.TrimSpace(publicURL), "/")
	if base == "" {
		base = "http://localhost:5174"
	}
	return base + "/unsubscribe?token=" + url.QueryEscape(strings.TrimSpace(token))
}

func signUnsubscribeMessage(key []byte, message string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
