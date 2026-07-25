package graph

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func (r *incidentRoleAssignmentResolver) loadIncidentRoleAssignment(
	ctx context.Context,
	obj *model.IncidentRoleAssignment,
) (db.IncidentRoleAssignment, auth.SessionContext, *db.Queries, error) {
	if obj == nil {
		return db.IncidentRoleAssignment{}, auth.SessionContext{}, nil, nil
	}

	sc, err := requireAuthSession(ctx)
	if err != nil {
		return db.IncidentRoleAssignment{}, auth.SessionContext{}, nil, err
	}

	assignmentID, err := parseUUIDField(obj.ID, "id")
	if err != nil {
		return db.IncidentRoleAssignment{}, auth.SessionContext{}, nil, err
	}

	queries := db.New(r.pool)
	assignment, err := queries.GetIncidentRoleAssignmentByID(ctx, db.GetIncidentRoleAssignmentByIDParams{
		ID:             assignmentID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.IncidentRoleAssignment{}, auth.SessionContext{}, nil, gqlerr.New(handlers.CodeNotFound, "role assignment not found")
		}
		r.logger.Error("load incident role assignment failed", "error", err)
		return db.IncidentRoleAssignment{}, auth.SessionContext{}, nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if _, err := r.loadIncidentWithTeamAccess(ctx, queries, sc, assignment.IncidentID); err != nil {
		return db.IncidentRoleAssignment{}, auth.SessionContext{}, nil, err
	}

	return assignment, sc, queries, nil
}

func (r *incidentRoleAssignmentResolver) loadIncidentRoleAssignmentRole(ctx context.Context, obj *model.IncidentRoleAssignment) (*model.IncidentRoleDefinition, error) {
	assignment, sc, queries, err := r.loadIncidentRoleAssignment(ctx, obj)
	if err != nil || queries == nil {
		return nil, err
	}

	def, err := queries.GetIncidentRoleDefinitionByID(ctx, db.GetIncidentRoleDefinitionByIDParams{
		ID:             assignment.RoleDefinitionID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load incident role definition failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return incidentRoleDefinitionFromDB(def), nil
}

func (r *incidentRoleAssignmentResolver) loadIncidentRoleAssignmentUser(ctx context.Context, obj *model.IncidentRoleAssignment) (*model.User, error) {
	assignment, sc, queries, err := r.loadIncidentRoleAssignment(ctx, obj)
	if err != nil || queries == nil {
		return nil, err
	}

	user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             assignment.UserID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load incident role user failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return userFromDB(user), nil
}

func (r *incidentRoleAssignmentResolver) loadIncidentRoleAssignmentAssignedBy(ctx context.Context, obj *model.IncidentRoleAssignment) (*model.User, error) {
	assignment, sc, queries, err := r.loadIncidentRoleAssignment(ctx, obj)
	if err != nil || queries == nil {
		return nil, err
	}

	user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             assignment.AssignedByUserID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load incident role assigner failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return userFromDB(user), nil
}
