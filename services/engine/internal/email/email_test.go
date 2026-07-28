package email

import (
	"strings"
	"testing"

	allure "github.com/allure-framework/allure-go/commons/gotest"

	"github.com/allure-framework/allure-go/testify/require"
)

func TestBuildMIMEMessageIncludesPlainAndHTMLParts(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {

		payload := string(buildMIMEMessage(
			"noreply@example.com",
			"oncall@example.com",
			"[HIGH] Disk full",
			"plain body",
			"<p>html body</p>",
		))

		require.Contains(a, payload, "multipart/alternative")
		require.Contains(a, payload, "text/plain; charset=UTF-8")
		require.Contains(a, payload, "plain body")
		require.Contains(a, payload, "text/html; charset=UTF-8")
		require.Contains(a, payload, "<p>html body</p>")
		require.True(a, strings.Contains(payload, "Subject: =?utf-8?q?=5BHIGH=5D_Disk_full?=") ||
			strings.Contains(payload, "Subject: [HIGH] Disk full"))
	})
}
