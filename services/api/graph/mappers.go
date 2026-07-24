package graph

import (
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
)

func userFromDB(user db.User) *model.User {
	return &model.User{
		ID:             user.ID.String(),
		Email:          user.Email,
		Role:           userRoleFromDB(user.Role),
		OrganizationID: user.OrganizationID.String(),
		CreatedAt:      timeFromDB(user.CreatedAt),
		UpdatedAt:      timeFromDB(user.UpdatedAt),
	}
}

func organizationFromDB(org db.Organization) *model.Organization {
	return &model.Organization{
		ID:        org.ID.String(),
		Name:      org.Name,
		CreatedAt: timeFromDB(org.CreatedAt),
		UpdatedAt: timeFromDB(org.UpdatedAt),
	}
}

func userRoleFromDB(role string) model.UserRole {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "admin":
		return model.UserRoleAdmin
	default:
		return model.UserRoleMember
	}
}

func timeFromDB(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time.UTC()
}
