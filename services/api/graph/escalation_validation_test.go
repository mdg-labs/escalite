package graph

import (
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"testing"

	"github.com/allure-framework/allure-go/testify/require"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func TestValidateEscalationStepInputs(t *testing.T) {
	allure.Wrap(t, func(a *allure.Context) {
		validUserTarget := func() *model.EscalationStepTargetInput {
			userID := "550e8400-e29b-41d4-a716-446655440000"
			return &model.EscalationStepTargetInput{
				TargetType: "user",
				UserID:     &userID,
			}
		}

		t.Run("accepts contiguous orders", func(t *testing.T) {
			err := validateEscalationStepInputs([]*model.EscalationStepInput{
				{StepOrder: 1, DelayMinutes: 0, Targets: []*model.EscalationStepTargetInput{validUserTarget()}},
				{StepOrder: 2, DelayMinutes: 5, Targets: []*model.EscalationStepTargetInput{validUserTarget()}},
			})
			require.NoError(a, err)
		})

		t.Run("rejects order gaps", func(t *testing.T) {
			err := validateEscalationStepInputs([]*model.EscalationStepInput{
				{StepOrder: 1, DelayMinutes: 0, Targets: []*model.EscalationStepTargetInput{validUserTarget()}},
				{StepOrder: 3, DelayMinutes: 5, Targets: []*model.EscalationStepTargetInput{validUserTarget()}},
			})
			require.Error(a, err)

			var coded *gqlerr.CodedError
			require.ErrorAs(a, err, &coded)
			require.Equal(a, handlers.CodeValidation, coded.Code)
		})

		t.Run("rejects empty steps", func(t *testing.T) {
			err := validateEscalationStepInputs(nil)
			require.Error(a, err)

			var coded *gqlerr.CodedError
			require.ErrorAs(a, err, &coded)
			require.Equal(a, handlers.CodeValidation, coded.Code)
		})

		t.Run("rejects steps without targets", func(t *testing.T) {
			err := validateEscalationStepInputs([]*model.EscalationStepInput{
				{StepOrder: 1, DelayMinutes: 0, Targets: []*model.EscalationStepTargetInput{}},
			})
			require.Error(a, err)

			var coded *gqlerr.CodedError
			require.ErrorAs(a, err, &coded)
			require.Equal(a, handlers.CodeValidation, coded.Code)
		})

		t.Run("rejects incomplete targets", func(t *testing.T) {
			err := validateEscalationStepInputs([]*model.EscalationStepInput{
				{
					StepOrder:    1,
					DelayMinutes: 0,
					Targets: []*model.EscalationStepTargetInput{
						{TargetType: "webhook"},
					},
				},
			})
			require.Error(a, err)

			var coded *gqlerr.CodedError
			require.ErrorAs(a, err, &coded)
			require.Equal(a, handlers.CodeValidation, coded.Code)
		})
	})
}
