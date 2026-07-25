package graph

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func validateEscalationStepInputs(steps []*model.EscalationStepInput) error {
	if len(steps) == 0 {
		return gqlerr.New(handlers.CodeValidation, "at least one step is required")
	}

	orders := make([]int, 0, len(steps))
	for _, step := range steps {
		if step == nil {
			return gqlerr.New(handlers.CodeValidation, "step is required")
		}
		if step.StepOrder < 1 {
			return gqlerr.New(handlers.CodeValidation, "stepOrder must be at least 1")
		}
		if step.DelayMinutes < 0 {
			return gqlerr.New(handlers.CodeValidation, "delayMinutes must be non-negative")
		}
		orders = append(orders, step.StepOrder)
	}

	sort.Ints(orders)
	for i, order := range orders {
		if order != i+1 {
			return gqlerr.New(handlers.CodeValidation, "step orders must be contiguous starting at 1")
		}
	}

	return nil
}

func requireAdminSession(ctx context.Context) (auth.SessionContext, error) {
	sc, ok := auth.SessionFromContext(ctx)
	if !ok {
		return auth.SessionContext{}, gqlerr.New(handlers.CodeUnauthenticated, "authentication required")
	}
	if !authz.IsAdmin(sc.User.Role) {
		return auth.SessionContext{}, gqlerr.New(handlers.CodeForbidden, "admin access required")
	}
	return sc, nil
}

func parseUUIDField(value, fieldName string) (uuid.UUID, error) {
	id, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil {
		return uuid.Nil, gqlerr.New(handlers.CodeValidation, fieldName+" is invalid")
	}
	return id, nil
}

func insertEscalationSteps(
	ctx context.Context,
	q db.Querier,
	orgID, policyID uuid.UUID,
	inputs []*model.EscalationStepInput,
) ([]db.EscalationStep, error) {
	steps := make([]db.EscalationStep, 0, len(inputs))
	for _, input := range inputs {
		repeatLastStep := false
		if input.RepeatLastStep != nil {
			repeatLastStep = *input.RepeatLastStep
		}

		var maxRepeats pgtype.Int4
		if input.MaxRepeats != nil {
			maxRepeats = pgtype.Int4{Int32: int32(*input.MaxRepeats), Valid: true}
		}

		step, err := q.CreateEscalationStep(ctx, db.CreateEscalationStepParams{
			ID:                 uuid.Must(uuid.NewV7()),
			EscalationPolicyID: policyID,
			OrganizationID:     orgID,
			StepOrder:          int32(input.StepOrder),
			DelayMinutes:       int32(input.DelayMinutes),
			RepeatLastStep:     repeatLastStep,
			MaxRepeats:         maxRepeats,
		})
		if err != nil {
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		steps = append(steps, step)
	}
	return steps, nil
}
