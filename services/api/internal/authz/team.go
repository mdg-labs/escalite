package authz

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

// CheckTeamAccess verifies the user may access a team in their organization.
// Org admins may access any team; members require a team_memberships row.
func CheckTeamAccess(ctx context.Context, q db.Querier, user db.User, teamID uuid.UUID) (db.Team, error) {
	team, err := q.GetTeamByID(ctx, db.GetTeamByIDParams{
		ID:             teamID,
		OrganizationID: user.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Team{}, ErrNotFound
		}
		return db.Team{}, err
	}

	if IsAdmin(user.Role) {
		return team, nil
	}

	hasMembership, err := q.HasTeamMembership(ctx, db.HasTeamMembershipParams{
		TeamID:         teamID,
		UserID:         user.ID,
		OrganizationID: user.OrganizationID,
	})
	if err != nil {
		return db.Team{}, err
	}
	if !hasMembership {
		return db.Team{}, ErrForbidden
	}

	return team, nil
}
