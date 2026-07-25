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
		if len(step.Targets) == 0 {
			return gqlerr.New(handlers.CodeValidation, "each step must include at least one target")
		}
		for _, target := range step.Targets {
			if err := validateEscalationStepTargetInput(target); err != nil {
				return err
			}
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

func validateEscalationStepTargetInput(target *model.EscalationStepTargetInput) error {
	if target == nil {
		return gqlerr.New(handlers.CodeValidation, "target is required")
	}

	switch strings.TrimSpace(target.TargetType) {
	case "user":
		if target.UserID == nil || strings.TrimSpace(*target.UserID) == "" {
			return gqlerr.New(handlers.CodeValidation, "userId is required for user targets")
		}
		if _, err := parseUUIDField(*target.UserID, "userId"); err != nil {
			return err
		}
	case "rotation":
		if target.ScheduleID == nil || strings.TrimSpace(*target.ScheduleID) == "" {
			return gqlerr.New(handlers.CodeValidation, "scheduleId is required for rotation targets")
		}
		if _, err := parseUUIDField(*target.ScheduleID, "scheduleId"); err != nil {
			return err
		}
	case "webhook":
		if target.WebhookURL == nil || strings.TrimSpace(*target.WebhookURL) == "" {
			return gqlerr.New(handlers.CodeValidation, "webhookUrl is required for webhook targets")
		}
	default:
		return gqlerr.New(handlers.CodeValidation, "targetType must be user, rotation, or webhook")
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

		if err := insertEscalationStepTargets(ctx, q, orgID, step.ID, input.Targets); err != nil {
			return nil, err
		}

		steps = append(steps, step)
	}
	return steps, nil
}

func insertEscalationStepTargets(
	ctx context.Context,
	q db.Querier,
	orgID, stepID uuid.UUID,
	inputs []*model.EscalationStepTargetInput,
) error {
	for _, input := range inputs {
		userID := pgtype.UUID{}
		scheduleID := pgtype.UUID{}
		webhookURL := pgtype.Text{}

		switch strings.TrimSpace(input.TargetType) {
		case "user":
			parsedUserID, err := parseUUIDField(*input.UserID, "userId")
			if err != nil {
				return err
			}
			userID = pgtype.UUID{Bytes: parsedUserID, Valid: true}
		case "rotation":
			parsedScheduleID, err := parseUUIDField(*input.ScheduleID, "scheduleId")
			if err != nil {
				return err
			}
			scheduleID = pgtype.UUID{Bytes: parsedScheduleID, Valid: true}
		case "webhook":
			webhookURL = pgtype.Text{String: strings.TrimSpace(*input.WebhookURL), Valid: true}
		}

		if _, err := q.CreateEscalationStepTarget(ctx, db.CreateEscalationStepTargetParams{
			ID:               uuid.Must(uuid.NewV7()),
			EscalationStepID: stepID,
			OrganizationID:   orgID,
			TargetType:       strings.TrimSpace(input.TargetType),
			UserID:           userID,
			ScheduleID:       scheduleID,
			WebhookUrl:       webhookURL,
			Channels:         []byte("[]"),
		}); err != nil {
			return gqlerr.New(handlers.CodeInternal, "internal error")
		}
	}

	return nil
}
