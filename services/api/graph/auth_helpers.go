package graph

import (
	"context"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/audit"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

const (
	invalidCredentialsMessage = "invalid credentials"
	minPasswordLength         = 8
)

func (r *mutationResolver) recordFailedLogin(
	ctx context.Context,
	queries db.Querier,
	user *db.User,
	meta audit.RequestMeta,
) {
	orgID := uuid.Nil
	var targetUserID *uuid.UUID
	if user != nil {
		orgID = user.OrganizationID
		targetUserID = &user.ID
	} else {
		org, err := queries.GetFirstOrganization(ctx)
		if err == nil {
			orgID = org.ID
		}
	}
	r.audit.LoginFailed(ctx, queries, orgID, targetUserID, meta)
}
