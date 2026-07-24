package graph

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *mutationResolver) CreateEscalationPolicy(ctx context.Context, input model.CreateEscalationPolicyInput) (*model.EscalationPolicy, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, gqlerr.New(handlers.CodeValidation, "name is required")
	}
	if err := validateEscalationStepInputs(input.Steps); err != nil {
		return nil, err
	}

	serviceID, err := parseUUIDField(input.ServiceID, "serviceId")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             serviceID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "service not found")
		}
		r.logger.Error("load service failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	policyID := uuid.Must(uuid.NewV7())

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("begin transaction failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := queries.WithTx(tx)

	policy, err := txQueries.CreateEscalationPolicy(ctx, db.CreateEscalationPolicyParams{
		ID:             policyID,
		OrganizationID: sc.User.OrganizationID,
		ServiceID:      serviceID,
		Name:           name,
	})
	if err != nil {
		r.logger.Error("create escalation policy failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	steps, err := insertEscalationSteps(ctx, txQueries, sc.User.OrganizationID, policyID, input.Steps)
	if err != nil {
		return nil, err
	}

	r.audit.EscalationPolicyCreated(ctx, txQueries, sc.User.OrganizationID, sc.User.ID, policyID)

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("commit transaction failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return escalationPolicyFromDB(policy, steps), nil
}

func (r *mutationResolver) UpdateEscalationPolicy(ctx context.Context, input model.UpdateEscalationPolicyInput) (*model.EscalationPolicy, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, gqlerr.New(handlers.CodeValidation, "name is required")
	}
	if err := validateEscalationStepInputs(input.Steps); err != nil {
		return nil, err
	}

	policyID, err := parseUUIDField(input.ID, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetEscalationPolicyByID(ctx, db.GetEscalationPolicyByIDParams{
		ID:             policyID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "escalation policy not found")
		}
		r.logger.Error("load escalation policy failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		r.logger.Error("begin transaction failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	txQueries := queries.WithTx(tx)

	policy, err := txQueries.UpdateEscalationPolicy(ctx, db.UpdateEscalationPolicyParams{
		ID:             policyID,
		OrganizationID: sc.User.OrganizationID,
		Name:           name,
	})
	if err != nil {
		r.logger.Error("update escalation policy failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if err := txQueries.DeleteEscalationStepsByPolicyID(ctx, db.DeleteEscalationStepsByPolicyIDParams{
		EscalationPolicyID: policyID,
		OrganizationID:     sc.User.OrganizationID,
	}); err != nil {
		r.logger.Error("delete escalation steps failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	steps, err := insertEscalationSteps(ctx, txQueries, sc.User.OrganizationID, policyID, input.Steps)
	if err != nil {
		return nil, err
	}

	r.audit.EscalationPolicyUpdated(ctx, txQueries, sc.User.OrganizationID, sc.User.ID, policyID)

	if err := tx.Commit(ctx); err != nil {
		r.logger.Error("commit transaction failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return escalationPolicyFromDB(policy, steps), nil
}

func (r *mutationResolver) DeleteEscalationPolicy(ctx context.Context, id string) (bool, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return false, err
	}

	policyID, err := parseUUIDField(id, "id")
	if err != nil {
		return false, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetEscalationPolicyByID(ctx, db.GetEscalationPolicyByIDParams{
		ID:             policyID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, gqlerr.New(handlers.CodeNotFound, "escalation policy not found")
		}
		r.logger.Error("load escalation policy failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if err := queries.DeleteEscalationPolicy(ctx, db.DeleteEscalationPolicyParams{
		ID:             policyID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		r.logger.Error("delete escalation policy failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	r.audit.EscalationPolicyDeleted(ctx, queries, sc.User.OrganizationID, sc.User.ID, policyID)
	return true, nil
}

func (r *queryResolver) EscalationPolicy(ctx context.Context, id string) (*model.EscalationPolicy, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	policyID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	policy, err := queries.GetEscalationPolicyByID(ctx, db.GetEscalationPolicyByIDParams{
		ID:             policyID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		r.logger.Error("load escalation policy failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	steps, err := queries.ListEscalationStepsByPolicyID(ctx, db.ListEscalationStepsByPolicyIDParams{
		EscalationPolicyID: policyID,
		OrganizationID:     sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list escalation steps failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return escalationPolicyFromDB(policy, steps), nil
}

func (r *queryResolver) EscalationPolicies(ctx context.Context, serviceID string) ([]*model.EscalationPolicy, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	svcID, err := parseUUIDField(serviceID, "serviceId")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             svcID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "service not found")
		}
		r.logger.Error("load service failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	policies, err := queries.ListEscalationPoliciesByServiceID(ctx, db.ListEscalationPoliciesByServiceIDParams{
		ServiceID:      svcID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("list escalation policies failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	result := make([]*model.EscalationPolicy, 0, len(policies))
	for _, policy := range policies {
		steps, err := queries.ListEscalationStepsByPolicyID(ctx, db.ListEscalationStepsByPolicyIDParams{
			EscalationPolicyID: policy.ID,
			OrganizationID:     sc.User.OrganizationID,
		})
		if err != nil {
			r.logger.Error("list escalation steps failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		result = append(result, escalationPolicyFromDB(policy, steps))
	}

	return result, nil
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
