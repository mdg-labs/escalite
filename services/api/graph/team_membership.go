package graph

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

// AddTeamMember adds an organization user to a team (org admin only).
//
// Team membership scopes member-role users to the teams they belong to. Org-wide
// UserRole (admin vs member) is separate: admins already have access to every team
// without a team_memberships row, but membership is still recorded for roster and
// picker display.
func (r *mutationResolver) AddTeamMember(ctx context.Context, teamID string, userID string) (*model.TeamMembership, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	parsedTeamID, err := parseUUIDField(teamID, "teamId")
	if err != nil {
		return nil, err
	}
	parsedUserID, err := parseUUIDField(userID, "userId")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetTeamByID(ctx, db.GetTeamByIDParams{
		ID:             parsedTeamID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "team not found")
		}
		r.logger.Error("get team failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	targetUser, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             parsedUserID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "user not found")
		}
		r.logger.Error("get user failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	if !auth.IsUserActive(targetUser.DeprovisionedAt.Valid) {
		return nil, gqlerr.New(handlers.CodeValidation, "user is deprovisioned")
	}

	hasMembership, err := queries.HasTeamMembership(ctx, db.HasTeamMembershipParams{
		TeamID:         parsedTeamID,
		UserID:         parsedUserID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("check team membership failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	if hasMembership {
		return nil, gqlerr.New(handlers.CodeValidation, "user is already a team member")
	}

	membership, err := queries.CreateTeamMembership(ctx, db.CreateTeamMembershipParams{
		ID:             uuid.Must(uuid.NewV7()),
		TeamID:         parsedTeamID,
		UserID:         parsedUserID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("create team membership failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return teamMembershipFromDB(membership), nil
}

// RemoveTeamMember removes an organization user from a team (org admin only).
//
// Org UserRole (admin vs member) is unchanged by this mutation. Member-role users
// lose team-scoped access when removed; admin-role users retain org-wide team access
// regardless of team_memberships rows. Removal is rejected when the target user is
// the last org admin to prevent locking the organization without administrators.
func (r *mutationResolver) RemoveTeamMember(ctx context.Context, teamID string, userID string) (bool, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return false, err
	}

	parsedTeamID, err := parseUUIDField(teamID, "teamId")
	if err != nil {
		return false, err
	}
	parsedUserID, err := parseUUIDField(userID, "userId")
	if err != nil {
		return false, err
	}

	queries := db.New(r.pool)
	if _, err := queries.GetTeamByID(ctx, db.GetTeamByIDParams{
		ID:             parsedTeamID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, gqlerr.New(handlers.CodeNotFound, "team not found")
		}
		r.logger.Error("get team failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	targetUser, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             parsedUserID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, gqlerr.New(handlers.CodeNotFound, "user not found")
		}
		r.logger.Error("get user failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if authz.IsAdmin(targetUser.Role) {
		adminCount, err := queries.CountOrgAdmins(ctx, sc.User.OrganizationID)
		if err != nil {
			r.logger.Error("count org admins failed", "error", err)
			return false, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		if adminCount <= 1 {
			return false, gqlerr.New(handlers.CodeValidation, "cannot remove last admin from org")
		}
	}

	if _, err := queries.GetTeamMembership(ctx, db.GetTeamMembershipParams{
		TeamID:         parsedTeamID,
		UserID:         parsedUserID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, gqlerr.New(handlers.CodeNotFound, "team membership not found")
		}
		r.logger.Error("get team membership failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if err := queries.DeleteTeamMembership(ctx, db.DeleteTeamMembershipParams{
		TeamID:         parsedTeamID,
		UserID:         parsedUserID,
		OrganizationID: sc.User.OrganizationID,
	}); err != nil {
		r.logger.Error("delete team membership failed", "error", err)
		return false, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return true, nil
}
