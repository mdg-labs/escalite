package auditexport

import (
	"encoding/csv"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"strings"
	"testing"
	"time"

	"github.com/allure-framework/allure-go/testify/require"
	"github.com/google/uuid"
)

func TestExportUsesRFC3339Timestamps(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		eventID := uuid.MustParse("018f8f2a-7b3c-7d4e-8f9a-0b1c2d3e4f50")
		createdAt := time.Date(2026, 7, 26, 10, 15, 30, 0, time.FixedZone("CEST", 2*60*60))

		rows := []Row{
			{
				ID:        eventID,
				CreatedAt: createdAt,
				Action:    "auth.login",
				Metadata:  `{"ip":"127.0.0.1"}`,
			},
		}

		data, err := Export(rows)
		require.NoError(a, err)

		records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
		require.NoError(a, err)
		require.Len(a, records, 2)
		require.Equal(a, "created_at", records[0][1])
		require.Equal(a, createdAt.UTC().Format(time.RFC3339), records[1][1])
	})
}
