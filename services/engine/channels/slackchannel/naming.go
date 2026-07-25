package slackchannel

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

const DefaultNameTemplate = "incident-{short_id}"

var nonChannelNameChars = regexp.MustCompile(`[^a-z0-9_-]+`)

// ShortID returns the first eight hex characters of an incident UUID.
func ShortID(incidentID uuid.UUID) string {
	return strings.ToLower(strings.ReplaceAll(incidentID.String(), "-", ""))[:8]
}

// RenderChannelName applies template variables and normalizes the result for Slack.
func RenderChannelName(template string, incidentID uuid.UUID, title string) string {
	if strings.TrimSpace(template) == "" {
		template = DefaultNameTemplate
	}

	name := template
	name = strings.ReplaceAll(name, "{short_id}", ShortID(incidentID))
	name = strings.ReplaceAll(name, "{title}", sanitizeTitleSlug(title))
	return SanitizeChannelName(name)
}

func sanitizeTitleSlug(title string) string {
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = strings.ReplaceAll(slug, " ", "-")
	return nonChannelNameChars.ReplaceAllString(slug, "-")
}

// SanitizeChannelName enforces Slack public channel naming rules.
func SanitizeChannelName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "#", "")
	name = nonChannelNameChars.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-_")
	if name == "" {
		return "incident"
	}
	if len(name) > 80 {
		name = strings.Trim(name[:80], "-_")
	}
	return name
}
