package graph

import (
	"context"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

const (
	defaultOrganizationUsersLimit = 500
	maxOrganizationUsersLimit     = 500
)

func organizationUsersListLimit(limit *int) int32 {
	if limit == nil || *limit <= 0 {
		return defaultOrganizationUsersLimit
	}
	if *limit > maxOrganizationUsersLimit {
		return maxOrganizationUsersLimit
	}
	return int32(*limit)
}

func (r *queryResolver) resolveOrganizationUsers(
	ctx context.Context,
	limit *int,
) ([]*model.OrganizationUser, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	listLimit := organizationUsersListLimit(limit)

	var users []db.User
	if authz.IsAdmin(sc.User.Role) {
		users, err = queries.ListOrganizationUsersForOrgAdmin(ctx, db.ListOrganizationUsersForOrgAdminParams{
			OrganizationID: sc.User.OrganizationID,
			Limit:          listLimit,
		})
	} else {
		users, err = queries.ListOrganizationUsersForTeamMember(ctx, db.ListOrganizationUsersForTeamMemberParams{
			OrganizationID: sc.User.OrganizationID,
			UserID:         sc.User.ID,
			Limit:          listLimit,
		})
	}
	if err != nil {
		r.logger.Error("list organization users failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if len(users) == 0 {
		return []*model.OrganizationUser{}, nil
	}

	userIDs := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		userIDs = append(userIDs, user.ID)
	}

	memberships, err := queries.ListTeamMembershipsByUserIDs(ctx, db.ListTeamMembershipsByUserIDsParams{
		OrganizationID: sc.User.OrganizationID,
		UserIds:        userIDs,
	})
	if err != nil {
		r.logger.Error("list team memberships for organization users failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	membershipsByUserID := groupTeamMembershipsByUserID(memberships)
	result := make([]*model.OrganizationUser, 0, len(users))
	for _, user := range users {
		result = append(result, organizationUserFromDB(user, membershipsByUserID[user.ID]))
	}

	return result, nil
}

func groupTeamMembershipsByUserID(memberships []db.TeamMembership) map[uuid.UUID][]db.TeamMembership {
	grouped := make(map[uuid.UUID][]db.TeamMembership, len(memberships))
	for _, membership := range memberships {
		grouped[membership.UserID] = append(grouped[membership.UserID], membership)
	}
	return grouped
}

func organizationUserFromDB(user db.User, memberships []db.TeamMembership) *model.OrganizationUser {
	teamMemberships := make([]*model.TeamMembership, 0, len(memberships))
	for _, membership := range memberships {
		teamMemberships = append(teamMemberships, teamMembershipFromDB(membership))
	}

	return &model.OrganizationUser{
		ID:              user.ID.String(),
		Email:           user.Email,
		Role:            userRoleFromDB(user.Role),
		TeamMemberships: teamMemberships,
	}
}
