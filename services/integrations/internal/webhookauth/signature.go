// Package webhookauth verifies inbound webhook request authenticity.
package webhookauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"github.com/mdg-labs/escalite/services/integrations"
)

const signatureHeader = "X-Escalite-Signature"

// ErrInvalidSignature indicates the request signature did not match the configured secret.
type ErrInvalidSignature = integrations.ErrInvalidSignature

// Verify checks the request signature when secret is configured.
// An empty secret skips verification. The header value may be an HMAC digest
// (sha256=<hex>) or the raw shared secret for providers that only support static headers.
func Verify(secret string, body []byte, headers map[string][]string) error {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil
	}

	header := strings.TrimSpace(firstHeader(headers, signatureHeader))
	if header == "" {
		return integrations.ErrInvalidSignature{}
	}

	expected := SignBody(secret, body)
	if subtle.ConstantTimeCompare([]byte(header), []byte(expected)) == 1 {
		return nil
	}
	if subtle.ConstantTimeCompare([]byte(header), []byte(secret)) == 1 {
		return nil
	}
	return integrations.ErrInvalidSignature{}
}

// SignBody returns the HMAC-SHA256 signature for body using secret.
func SignBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func firstHeader(headers map[string][]string, name string) string {
	for key, values := range headers {
		if strings.EqualFold(key, name) && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
