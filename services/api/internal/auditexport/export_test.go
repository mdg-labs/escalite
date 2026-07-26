package auditexport

import (
	"encoding/csv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExportUsesRFC3339Timestamps(t *testing.T) {
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
	require.NoError(t, err)

	records, err := csv.NewReader(strings.NewReader(string(data))).ReadAll()
	require.NoError(t, err)
	require.Len(t, records, 2)
	require.Equal(t, "created_at", records[0][1])
	require.Equal(t, createdAt.UTC().Format(time.RFC3339), records[1][1])
}
