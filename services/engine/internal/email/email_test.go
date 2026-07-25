package email

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildMIMEMessageIncludesPlainAndHTMLParts(t *testing.T) {
	payload := string(buildMIMEMessage(
		"noreply@example.com",
		"oncall@example.com",
		"[HIGH] Disk full",
		"plain body",
		"<p>html body</p>",
	))

	require.Contains(t, payload, "multipart/alternative")
	require.Contains(t, payload, "text/plain; charset=UTF-8")
	require.Contains(t, payload, "plain body")
	require.Contains(t, payload, "text/html; charset=UTF-8")
	require.Contains(t, payload, "<p>html body</p>")
	require.True(t, strings.Contains(payload, "Subject: =?utf-8?q?=5BHIGH=5D_Disk_full?=") ||
		strings.Contains(payload, "Subject: [HIGH] Disk full"))
}
