package mapping_test

import (
	"os"
	"testing"
	"time"

	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/tools/pagerduty-importer/internal/mapping"
)

func TestStoreToFileAndWriteJSON(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		orgID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000010")
		teamID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000011")
		userID := uuid.MustParse("018f3b28-9f3a-7000-8000-000000000012")

		store := mapping.NewStore(orgID, teamID)
		store.Users["PU1"] = userID
		store.AddSkipped("escalation_target", "PTNEST", "nested escalation policy references are not supported")

		importedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
		file := store.ToFile(importedAt)
		require.Equal(a, orgID, file.OrganizationID)
		require.Equal(a, userID.String(), file.Users["PU1"])
		require.Len(a, file.Skipped, 1)

		path := a.T().TempDir() + "/mapping.json"
		require.NoError(a, mapping.WriteJSON(path, file))

		data, err := os.ReadFile(path)
		require.NoError(a, err)
		require.Contains(a, string(data), orgID.String())
		require.Contains(a, string(data), "nested escalation policy references are not supported")
	})
}
