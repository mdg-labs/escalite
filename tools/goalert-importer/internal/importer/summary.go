package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/goalert"
	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/mapping"
)

// Summary describes what an import would create without writing.
type Summary struct {
	Counts          goalert.Counts
	OrganizationID  uuid.UUID
	TeamID          uuid.UUID
	IncludeAlerts   bool
	UserSamples     []string
	ScheduleSamples []string
	ServiceSamples  []string
	PolicySamples   []string
}

// FormatSummary prints a human-readable dry-run report.
func FormatSummary(w io.Writer, summary Summary) error {
	_, err := fmt.Fprintf(w, "GoAlert → Escalite import dry-run\n")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Target organization: %s\n", summary.OrganizationID)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "Target team: %s\n", summary.TeamID)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "\nEntity mapping (source count → Escalite table):\n")
	if err != nil {
		return err
	}

	lines := []struct {
		label  string
		count  int
		target string
	}{
		{"users", summary.Counts.Users, "users + team_memberships"},
		{"schedules", summary.Counts.Schedules, "schedules"},
		{"rotations", summary.Counts.Rotations, "rotations"},
		{"services", summary.Counts.Services, "services"},
		{"escalation_policies", summary.Counts.EscalationPolicies, "escalation_policies (per service)"},
		{"escalation_policy_steps", summary.Counts.EscalationSteps, "escalation_steps"},
		{"escalation_policy_actions", summary.Counts.EscalationActions, "escalation_step_targets"},
	}
	for _, line := range lines {
		_, err = fmt.Fprintf(w, "  %s: %d → %s\n", line.label, line.count, line.target)
		if err != nil {
			return err
		}
	}
	if summary.IncludeAlerts {
		_, err = fmt.Fprintf(w, "  alerts: %d → alerts (historical)\n", summary.Counts.Alerts)
		if err != nil {
			return err
		}
	} else {
		_, err = fmt.Fprintf(w, "  alerts: skipped (use --include-alerts to import %d historical alerts)\n", summary.Counts.Alerts)
		if err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(w, "\nNo writes performed (dry-run).\n")
	if err != nil {
		return err
	}

	if len(summary.UserSamples) > 0 {
		_, err = fmt.Fprintf(w, "\nSample users: %s\n", strings.Join(summary.UserSamples, ", "))
		if err != nil {
			return err
		}
	}
	if len(summary.ScheduleSamples) > 0 {
		_, err = fmt.Fprintf(w, "Sample schedules: %s\n", strings.Join(summary.ScheduleSamples, ", "))
		if err != nil {
			return err
		}
	}
	if len(summary.ServiceSamples) > 0 {
		_, err = fmt.Fprintf(w, "Sample services: %s\n", strings.Join(summary.ServiceSamples, ", "))
		if err != nil {
			return err
		}
	}
	if len(summary.PolicySamples) > 0 {
		_, err = fmt.Fprintf(w, "Sample escalation policies: %s\n", strings.Join(summary.PolicySamples, ", "))
		if err != nil {
			return err
		}
	}
	return nil
}

// RotationRRule maps GoAlert rotation type to an Escalite RRULE string.
func RotationRRule(rotationType string, shiftLength int64) string {
	interval := shiftLength
	if interval < 1 {
		interval = 1
	}
	switch strings.ToLower(strings.TrimSpace(rotationType)) {
	case "hourly":
		return fmt.Sprintf("FREQ=HOURLY;INTERVAL=%d", interval)
	case "weekly":
		return fmt.Sprintf("FREQ=WEEKLY;INTERVAL=%d", interval)
	case "monthly":
		return fmt.Sprintf("FREQ=MONTHLY;INTERVAL=%d", interval)
	default:
		return fmt.Sprintf("FREQ=DAILY;INTERVAL=%d", interval)
	}
}

// MapUserRole converts GoAlert role to Escalite role.
func MapUserRole(goalertRole string) string {
	if strings.EqualFold(strings.TrimSpace(goalertRole), "admin") {
		return "admin"
	}
	return "member"
}

// MapAlertStatus converts GoAlert alert status to Escalite status.
func MapAlertStatus(goalertStatus string) string {
	switch strings.ToLower(strings.TrimSpace(goalertStatus)) {
	case "closed":
		return "closed"
	case "active":
		return "triggered"
	default:
		return "triggered"
	}
}

// EncodeParticipants maps GoAlert user UUIDs to Escalite participant JSON.
func EncodeParticipants(goalertUserIDs []uuid.UUID, userMap map[string]uuid.UUID) ([]byte, error) {
	ids := make([]string, 0, len(goalertUserIDs))
	for _, srcID := range goalertUserIDs {
		mapped, ok := userMap[srcID.String()]
		if !ok {
			continue
		}
		ids = append(ids, mapped.String())
	}
	return json.Marshal(ids)
}

// ChannelsJSON returns default notification channels for imported targets.
func ChannelsJSON() []byte {
	return []byte(`["email"]`)
}

// BuildSummary loads source metadata for dry-run display.
func BuildSummary(
	reader *goalert.Reader,
	orgID, teamID uuid.UUID,
	includeAlerts bool,
) (Summary, error) {
	counts, err := reader.Counts(contextWithTimeout(), includeAlerts)
	if err != nil {
		return Summary{}, err
	}

	users, err := reader.ListUsers(contextWithTimeout())
	if err != nil {
		return Summary{}, err
	}
	schedules, err := reader.ListSchedules(contextWithTimeout())
	if err != nil {
		return Summary{}, err
	}
	services, err := reader.ListServices(contextWithTimeout())
	if err != nil {
		return Summary{}, err
	}
	policies, err := reader.ListEscalationPolicies(contextWithTimeout())
	if err != nil {
		return Summary{}, err
	}

	summary := Summary{
		Counts:          counts,
		OrganizationID:  orgID,
		TeamID:          teamID,
		IncludeAlerts:   includeAlerts,
		UserSamples:     sampleNames(users, 3, func(i int) string { return users[i].Email }),
		ScheduleSamples: sampleNames(schedules, 3, func(i int) string { return schedules[i].Name }),
		ServiceSamples:  sampleNames(services, 3, func(i int) string { return services[i].Name }),
		PolicySamples:   sampleNames(policies, 3, func(i int) string { return policies[i].Name }),
	}
	return summary, nil
}

func sampleNames[T any](items []T, limit int, nameFn func(int) string) []string {
	if len(items) == 0 {
		return nil
	}
	if limit > len(items) {
		limit = len(items)
	}
	out := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, nameFn(i))
	}
	return out
}

func contextWithTimeout() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	return ctx
}

// ImportResult contains post-import statistics.
type ImportResult struct {
	Mapping *mapping.Store
	Counts  mapping.File
}
