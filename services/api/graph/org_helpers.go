package graph

import (
	"context"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func organizationMembershipFromDB(row db.ListOrganizationMembershipsByAccountIDRow) *model.OrganizationMembership {
	return &model.OrganizationMembership{
		Organization: organizationFromDB(row.Organization),
		Role:         userRoleFromDB(row.User.Role),
	}
}

func accountIDFromSession(ctx context.Context, q db.Querier, sc auth.SessionContext) (uuid.UUID, error) {
	if sc.User.AccountID != uuid.Nil {
		return sc.User.AccountID, nil
	}
	return auth.AccountIDFromUser(ctx, q, sc.User)
}
