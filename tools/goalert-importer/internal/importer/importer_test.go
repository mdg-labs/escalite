package importer_test

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/goalert"
	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/importer"
	"github.com/mdg-labs/escalite/tools/goalert-importer/internal/mapping"
)

func TestFormatSummaryDryRun(t *testing.T) {
	orgID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000001")
	teamID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000002")

	summary := importer.Summary{
		Counts: goalert.Counts{
			Users:              3,
			Schedules:          2,
			Rotations:          4,
			Services:           5,
			EscalationPolicies: 2,
			EscalationSteps:    6,
			EscalationActions:  8,
			Alerts:             100,
		},
		OrganizationID:  orgID,
		TeamID:          teamID,
		IncludeAlerts:   false,
		UserSamples:     []string{"oncall@example.com"},
		ScheduleSamples: []string{"Primary"},
		ServiceSamples:  []string{"API"},
		PolicySamples:   []string{"Default"},
	}

	var buf bytes.Buffer
	require.NoError(t, importer.FormatSummary(&buf, summary))
	out := buf.String()

	require.Contains(t, out, "dry-run")
	require.Contains(t, out, "users: 3")
	require.Contains(t, out, "No writes performed")
	require.Contains(t, out, "skipped (use --include-alerts")
	require.NotContains(t, out, "alerts: 100 → alerts")
}

func TestFormatSummaryIncludesAlerts(t *testing.T) {
	summary := importer.Summary{
		Counts:         goalert.Counts{Alerts: 12},
		OrganizationID: uuid.New(),
		TeamID:         uuid.New(),
		IncludeAlerts:  true,
	}

	var buf bytes.Buffer
	require.NoError(t, importer.FormatSummary(&buf, summary))
	require.Contains(t, buf.String(), "alerts: 12 → alerts (historical)")
}

func TestRotationRRule(t *testing.T) {
	require.Equal(t, "FREQ=HOURLY;INTERVAL=2", importer.RotationRRule("hourly", 2))
	require.Equal(t, "FREQ=WEEKLY;INTERVAL=1", importer.RotationRRule("weekly", 1))
	require.Equal(t, "FREQ=DAILY;INTERVAL=3", importer.RotationRRule("daily", 3))
	require.Equal(t, "FREQ=DAILY;INTERVAL=1", importer.RotationRRule("unknown", 0))
}

func TestMapUserRole(t *testing.T) {
	require.Equal(t, "admin", importer.MapUserRole("admin"))
	require.Equal(t, "member", importer.MapUserRole("user"))
}

func TestMapAlertStatus(t *testing.T) {
	require.Equal(t, "closed", importer.MapAlertStatus("closed"))
	require.Equal(t, "triggered", importer.MapAlertStatus("active"))
}

func TestEncodeParticipants(t *testing.T) {
	src := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000020")
	dst := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000021")
	participants, err := importer.EncodeParticipants([]uuid.UUID{src}, map[string]uuid.UUID{
		src.String(): dst,
	})
	require.NoError(t, err)
	require.True(t, strings.Contains(string(participants), dst.String()))
}

func TestMappingFileRoundTrip(t *testing.T) {
	orgID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000010")
	teamID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000011")
	userID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000012")

	store := mapping.NewStore(orgID, teamID)
	store.Users["goalert-user-1"] = userID
	store.Services["svc-1"] = uuid.MustParse("018f3b28-9f3a-7000-8000-000000000013")

	path := t.TempDir() + "/goalert-id-mapping.json"
	importedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	require.NoError(t, mapping.WriteJSON(path, store.ToFile(importedAt)))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), "goalert-user-1")
	require.Contains(t, string(data), "svc-1")
	require.Contains(t, string(data), userID.String())
}
