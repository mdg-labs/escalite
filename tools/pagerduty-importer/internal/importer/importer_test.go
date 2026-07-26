package importer_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/importer"
	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/mapping"
	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/pagerduty"
)

func TestFormatSummaryDryRun(t *testing.T) {
	orgID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000001")
	teamID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000002")

	summary := importer.Summary{
		Counts: pagerduty.Counts{
			Users:              3,
			Schedules:          2,
			ScheduleLayers:     4,
			Services:           5,
			EscalationPolicies: 2,
			EscalationRules:    6,
		},
		OrganizationID:  orgID,
		TeamID:          teamID,
		UserSamples:     []string{"oncall@example.com"},
		ScheduleSamples: []string{"Primary"},
		ServiceSamples:  []string{"API"},
		PolicySamples:   []string{"Default"},
		UnsupportedObjects: []pagerduty.UnsupportedObject{
			{ObjectType: "escalation_target", SourceID: "PTABC", Reason: "nested escalation policy references are not supported"},
		},
	}

	var buf bytes.Buffer
	require.NoError(t, importer.FormatSummary(&buf, summary))
	out := buf.String()

	require.Contains(t, out, "dry-run")
	require.Contains(t, out, "users: 3")
	require.Contains(t, out, "No writes performed")
	require.Contains(t, out, "Unsupported objects (will be skipped)")
	require.Contains(t, out, "escalation_target PTABC")
}

func TestFormatImportReportListsSkipped(t *testing.T) {
	store := mapping.NewStore(uuid.New(), uuid.New())
	store.Users["PU123"] = uuid.New()
	store.AddSkipped("escalation_target", "PTNEST", "nested escalation policy references are not supported")
	store.AddSkipped("user", "PUNOEMAIL", "missing email")

	var buf bytes.Buffer
	require.NoError(t, importer.FormatImportReport(&buf, store, time.Second, "/tmp/mapping.json"))
	out := buf.String()

	require.Contains(t, out, "Skipped unsupported objects (2)")
	require.Contains(t, out, "escalation_target PTNEST")
	require.Contains(t, out, "user PUNOEMAIL: missing email")
	require.Contains(t, out, "/tmp/mapping.json")
}

func TestLayerRRule(t *testing.T) {
	require.Equal(t, "FREQ=HOURLY;INTERVAL=1", importer.LayerRRule(3600))
	require.Equal(t, "FREQ=DAILY;INTERVAL=1", importer.LayerRRule(86400))
	require.Equal(t, "FREQ=WEEKLY;INTERVAL=1", importer.LayerRRule(604800))
	require.Equal(t, "FREQ=DAILY;INTERVAL=2", importer.LayerRRule(172800))
}

func TestMapUserRole(t *testing.T) {
	require.Equal(t, "admin", importer.MapUserRole("admin"))
	require.Equal(t, "admin", importer.MapUserRole("owner"))
	require.Equal(t, "member", importer.MapUserRole("limited_user"))
}

func TestEncodeParticipants(t *testing.T) {
	dst := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000021")
	participants, err := importer.EncodeParticipants([]string{"PU1"}, map[string]uuid.UUID{
		"PU1": dst,
	})
	require.NoError(t, err)
	require.True(t, strings.Contains(string(participants), dst.String()))
}
