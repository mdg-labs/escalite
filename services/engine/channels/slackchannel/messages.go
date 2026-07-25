package slackchannel

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// FormatIncidentAnchor returns the top-level channel message for a new incident.
func FormatIncidentAnchor(title string, incidentID uuid.UUID) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled incident"
	}
	return fmt.Sprintf("*Incident opened:* %s (`%s`)", title, ShortID(incidentID))
}

// FormatNoteReply returns a threaded reply for a timeline note.
func FormatNoteReply(actorLabel, body string) string {
	actorLabel = strings.TrimSpace(actorLabel)
	if actorLabel == "" {
		actorLabel = "Someone"
	}
	return fmt.Sprintf("*%s* added a note:\n%s", actorLabel, strings.TrimSpace(body))
}

// FormatStatusChangeReply returns a threaded reply for a status change.
func FormatStatusChangeReply(actorLabel, fromStatus, toStatus, body string) string {
	actorLabel = strings.TrimSpace(actorLabel)
	if actorLabel == "" {
		actorLabel = "Someone"
	}
	fromStatus = strings.TrimSpace(fromStatus)
	toStatus = strings.TrimSpace(toStatus)
	body = strings.TrimSpace(body)

	message := fmt.Sprintf("*%s* changed status: %s → %s", actorLabel, fromStatus, toStatus)
	if body != "" {
		message += "\n" + body
	}
	return message
}

// TimelineSummaryLine is one row in a resolve summary.
type TimelineSummaryLine struct {
	CreatedAt time.Time
	EventType string
	Body      string
	Actor     string
}

// FormatResolveSummary returns a top-level channel message when an incident is resolved.
func FormatResolveSummary(title string, incidentID uuid.UUID, lines []TimelineSummaryLine) string {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Untitled incident"
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf(":white_check_mark: *Incident resolved:* %s (`%s`)\n", title, ShortID(incidentID)))
	if len(lines) == 0 {
		builder.WriteString("_No timeline events recorded._")
		return builder.String()
	}

	builder.WriteString("*Timeline summary:*\n")
	for _, line := range lines {
		timestamp := line.CreatedAt.UTC().Format(time.RFC3339)
		actor := strings.TrimSpace(line.Actor)
		if actor == "" {
			actor = "system"
		}
		body := strings.TrimSpace(line.Body)
		if body == "" {
			body = line.EventType
		}
		builder.WriteString(fmt.Sprintf("• [%s] %s — %s\n", timestamp, actor, body))
	}
	return strings.TrimRight(builder.String(), "\n")
}
