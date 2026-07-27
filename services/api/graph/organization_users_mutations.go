package graph

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func (r *mutationResolver) resolveInviteUser(
	ctx context.Context,
	input model.InviteUserInput,
) (*model.OrganizationUser, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	org, err := queries.GetOrganizationByID(ctx, sc.User.OrganizationID)
	if err != nil {
		r.logger.Error("load organization for invite failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	user, err := r.userInviteService().InviteUser(
		ctx,
		sc.User.OrganizationID,
		sc.User.ID,
		org.Name,
		input.Email,
		userRoleToDB(input.Role),
	)
	if err != nil {
		return nil, userInviteGraphQLError(err)
	}

	r.audit.UserInvited(ctx, queries, sc.User.OrganizationID, sc.User.ID, user.ID, user.Email, user.Role)
	return r.organizationUserByID(ctx, queries, sc.User.OrganizationID, user.ID)
}

func (r *mutationResolver) resolveUpdateUserRole(
	ctx context.Context,
	input model.UpdateUserRoleInput,
) (*model.OrganizationUser, error) {
	sc, err := requireAdminSession(ctx)
	if err != nil {
		return nil, err
	}

	userID, err := parseUUIDField(input.UserID, "userId")
	if err != nil {
		return nil, err
	}

	newRole := userRoleToDB(input.Role)
	queries := db.New(r.pool)

	targetUser, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             userID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "user not found")
		}
		r.logger.Error("get user for role update failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	if !auth.IsUserActive(targetUser.DeprovisionedAt.Valid) {
		return nil, gqlerr.New(handlers.CodeValidation, "user is deprovisioned")
	}
	if strings.EqualFold(targetUser.Role, newRole) {
		return r.organizationUserByID(ctx, queries, sc.User.OrganizationID, targetUser.ID)
	}

	if authz.IsAdmin(targetUser.Role) && !authz.IsAdmin(newRole) {
		adminCount, err := queries.CountOrgAdmins(ctx, sc.User.OrganizationID)
		if err != nil {
			r.logger.Error("count org admins failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		if adminCount <= 1 {
			return nil, gqlerr.New(handlers.CodeValidation, "cannot demote last admin")
		}
	}

	updatedUser, err := queries.UpdateUserRole(ctx, db.UpdateUserRoleParams{
		ID:             userID,
		OrganizationID: sc.User.OrganizationID,
		Role:           newRole,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "user not found")
		}
		r.logger.Error("update user role failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	r.audit.RoleChanged(ctx, queries, sc.User.OrganizationID, sc.User.ID, updatedUser.ID, targetUser.Role, updatedUser.Role)
	return r.organizationUserByID(ctx, queries, sc.User.OrganizationID, updatedUser.ID)
}

func (r *mutationResolver) organizationUserByID(
	ctx context.Context,
	queries *db.Queries,
	orgID, userID uuid.UUID,
) (*model.OrganizationUser, error) {
	user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
		ID:             userID,
		OrganizationID: orgID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, gqlerr.New(handlers.CodeNotFound, "user not found")
		}
		r.logger.Error("load organization user failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	memberships, err := queries.ListTeamMembershipsByUserIDs(ctx, db.ListTeamMembershipsByUserIDsParams{
		OrganizationID: orgID,
		UserIds:        []uuid.UUID{userID},
	})
	if err != nil {
		r.logger.Error("list team memberships for organization user failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return organizationUserFromDB(user, memberships), nil
}

func userInviteGraphQLError(err error) error {
	var inviteErr *handlers.UserInviteError
	if errors.As(err, &inviteErr) {
		return gqlerr.New(inviteErr.Code, inviteErr.Message)
	}
	return gqlerr.New(handlers.CodeInternal, "internal error")
}
