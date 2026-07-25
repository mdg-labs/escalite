package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	slackSignatureHeader  = "X-Slack-Signature"
	slackTimestampHeader  = "X-Slack-Request-Timestamp"
	slackSignatureVersion = "v0"
	slackMaxRequestAge    = 5 * time.Minute
)

// ErrInvalidSlackSignature indicates the Slack request signature did not match.
var ErrInvalidSlackSignature = errors.New("invalid slack signature")

// VerifySlackSignature checks Slack interactive request authenticity.
func VerifySlackSignature(signingSecret string, body []byte, headers http.Header) error {
	signingSecret = strings.TrimSpace(signingSecret)
	if signingSecret == "" {
		return ErrInvalidSlackSignature
	}

	timestamp := strings.TrimSpace(headers.Get(slackTimestampHeader))
	signature := strings.TrimSpace(headers.Get(slackSignatureHeader))
	if timestamp == "" || signature == "" {
		return ErrInvalidSlackSignature
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrInvalidSlackSignature
	}

	now := time.Now().UTC()
	requestTime := time.Unix(ts, 0).UTC()
	if requestTime.Before(now.Add(-slackMaxRequestAge)) || requestTime.After(now.Add(slackMaxRequestAge)) {
		return ErrInvalidSlackSignature
	}

	expected := slackSignBody(signingSecret, timestamp, body)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return ErrInvalidSlackSignature
	}

	return nil
}

// SignSlackBody returns the Slack v0 signature for tests.
func SignSlackBody(signingSecret string, timestamp string, body []byte) string {
	return slackSignBody(signingSecret, timestamp, body)
}

func slackSignBody(signingSecret, timestamp string, body []byte) string {
	base := fmt.Sprintf("%s:%s:%s", slackSignatureVersion, timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	_, _ = mac.Write([]byte(base))
	return slackSignatureVersion + "=" + hex.EncodeToString(mac.Sum(nil))
}
