package graph

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidateEscalationStepInputs(t *testing.T) {
	t.Run("accepts contiguous orders", func(t *testing.T) {
		err := validateEscalationStepInputs([]*model.EscalationStepInput{
			{StepOrder: 1, DelayMinutes: 0},
			{StepOrder: 2, DelayMinutes: 5},
		})
		require.NoError(t, err)
	})

	t.Run("rejects order gaps", func(t *testing.T) {
		err := validateEscalationStepInputs([]*model.EscalationStepInput{
			{StepOrder: 1, DelayMinutes: 0},
			{StepOrder: 3, DelayMinutes: 5},
		})
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})

	t.Run("rejects empty steps", func(t *testing.T) {
		err := validateEscalationStepInputs(nil)
		require.Error(t, err)

		var coded *gqlerr.CodedError
		require.ErrorAs(t, err, &coded)
		require.Equal(t, handlers.CodeValidation, coded.Code)
	})
}
