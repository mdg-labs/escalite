package mapping_test

import (
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/mapping"
)

func TestStoreToFileAndWriteJSON(t *testing.T) {
	orgID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000010")
	teamID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000011")
	userID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000012")

	store := mapping.NewStore(orgID, teamID)
	store.Users["PU1"] = userID
	store.AddSkipped("escalation_target", "PTNEST", "nested escalation policy references are not supported")

	importedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	file := store.ToFile(importedAt)
	require.Equal(t, orgID, file.OrganizationID)
	require.Equal(t, userID.String(), file.Users["PU1"])
	require.Len(t, file.Skipped, 1)

	path := t.TempDir() + "/mapping.json"
	require.NoError(t, mapping.WriteJSON(path, file))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Contains(t, string(data), orgID.String())
	require.Contains(t, string(data), "nested escalation policy references are not supported")
}
