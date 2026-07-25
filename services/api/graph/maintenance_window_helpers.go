package graph

import (
	"strings"
	"time"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func validateMaintenanceWindowInput(description string, startsAt, endsAt time.Time) error {
	if strings.TrimSpace(description) == "" {
		return gqlerr.New(handlers.CodeValidation, "description is required")
	}
	if !endsAt.After(startsAt) {
		return gqlerr.New(handlers.CodeValidation, "endsAt must be after startsAt")
	}
	return nil
}

func maintenanceWindowFromDB(window db.MaintenanceWindow) *model.MaintenanceWindow {
	return &model.MaintenanceWindow{
		ID:                    window.ID.String(),
		OrganizationID:        window.OrganizationID.String(),
		ServiceID:             window.ServiceID.String(),
		Description:           window.Description,
		StartsAt:              timeFromDB(window.StartsAt),
		EndsAt:                timeFromDB(window.EndsAt),
		SuppressNotifications: window.SuppressNotifications,
		SuppressIngestion:     window.SuppressIngestion,
		CreatedAt:             timeFromDB(window.CreatedAt),
		UpdatedAt:             timeFromDB(window.UpdatedAt),
	}
}

func maintenanceWindowsFromDB(windows []db.MaintenanceWindow) []*model.MaintenanceWindow {
	result := make([]*model.MaintenanceWindow, 0, len(windows))
	for _, window := range windows {
		result = append(result, maintenanceWindowFromDB(window))
	}
	return result
}
