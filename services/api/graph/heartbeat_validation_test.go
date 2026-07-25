package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidatePositiveDurationSeconds(t *testing.T) {
	t.Run("accepts positive values", func(t *testing.T) {
		require.NoError(t, validatePositiveDurationSeconds(60, "intervalSeconds"))
	})

	t.Run("rejects zero", func(t *testing.T) {
		err := validatePositiveDurationSeconds(0, "graceSeconds")
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})

	t.Run("rejects negative values", func(t *testing.T) {
		err := validatePositiveDurationSeconds(-1, "intervalSeconds")
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})
}
