package orgbootstrap

import (
	"context"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/api/internal/db"
)

var defaultIncidentRoleDefinitions = []struct {
	Name      string
	SortOrder int32
}{
	{Name: "IC", SortOrder: 0},
	{Name: "Comms Lead", SortOrder: 1},
}

// SeedDefaultIncidentRoleDefinitions inserts the default incident role catalog for a new organization.
func SeedDefaultIncidentRoleDefinitions(ctx context.Context, queries db.Querier, orgID uuid.UUID) error {
	for _, role := range defaultIncidentRoleDefinitions {
		_, err := queries.CreateIncidentRoleDefinition(ctx, db.CreateIncidentRoleDefinitionParams{
			ID:             uuid.Must(uuid.NewV7()),
			OrganizationID: orgID,
			Name:           role.Name,
			SortOrder:      role.SortOrder,
		})
		if err != nil {
			return err
		}
	}
	return nil
}
