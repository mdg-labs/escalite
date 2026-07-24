package escalation

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCanRepeatLastStep_RepeatBoundary(t *testing.T) {
	t.Parallel()

	maxOne := pgtype.Int4{Int32: 1, Valid: true}
	maxTwo := pgtype.Int4{Int32: 2, Valid: true}

	require.True(t, canRepeatLastStep(true, maxOne, 0))
	require.False(t, canRepeatLastStep(true, maxOne, 1))
	require.True(t, canRepeatLastStep(true, maxTwo, 1))
	require.False(t, canRepeatLastStep(true, maxTwo, 2))

	require.False(t, canRepeatLastStep(false, maxOne, 0))
	require.False(t, canRepeatLastStep(true, pgtype.Int4{}, 0))
}
