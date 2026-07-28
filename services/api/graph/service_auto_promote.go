package graph

import (
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

func serviceAutoPromoteRuleFromDB(service db.Service) *model.ServiceAutoPromoteRule {
	priorities := make([]model.AlertPriority, 0, len(service.AutoPromoteSuppressEscalationPriorities))
	for _, priority := range service.AutoPromoteSuppressEscalationPriorities {
		priorities = append(priorities, alertPriorityFromDB(priority))
	}

	return &model.ServiceAutoPromoteRule{
		Enabled:                        service.AutoPromoteEnabled,
		AlertThreshold:                 int(service.AutoPromoteAlertThreshold),
		WindowSeconds:                  int(service.AutoPromoteWindowSeconds),
		SuppressEscalationPriorities: priorities,
	}
}

func buildUpdateServiceParams(input model.UpdateServiceInput) (db.UpdateServiceParams, error) {
	params := db.UpdateServiceParams{}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return db.UpdateServiceParams{}, gqlerr.New(handlers.CodeValidation, "name cannot be empty")
		}
		params.Name = pgtype.Text{String: name, Valid: true}
	}

	if input.TeamID != nil {
		teamID, err := parseUUIDField(*input.TeamID, "teamId")
		if err != nil {
			return db.UpdateServiceParams{}, err
		}
		params.TeamID = pgtype.UUID{Bytes: teamID, Valid: true}
	}

	if input.AutoPromoteEnabled != nil {
		params.AutoPromoteEnabled = pgtype.Bool{Bool: *input.AutoPromoteEnabled, Valid: true}
	}

	if input.AutoPromoteAlertThreshold != nil {
		if *input.AutoPromoteAlertThreshold < 1 {
			return db.UpdateServiceParams{}, gqlerr.New(handlers.CodeValidation, "autoPromoteAlertThreshold must be at least 1")
		}
		params.AutoPromoteAlertThreshold = pgtype.Int4{Int32: int32(*input.AutoPromoteAlertThreshold), Valid: true}
	}

	if input.AutoPromoteWindowSeconds != nil {
		if *input.AutoPromoteWindowSeconds < 1 {
			return db.UpdateServiceParams{}, gqlerr.New(handlers.CodeValidation, "autoPromoteWindowSeconds must be at least 1")
		}
		params.AutoPromoteWindowSeconds = pgtype.Int4{Int32: int32(*input.AutoPromoteWindowSeconds), Valid: true}
	}

	if input.AutoPromoteSuppressEscalationPriorities != nil {
		if len(input.AutoPromoteSuppressEscalationPriorities) == 0 {
			return db.UpdateServiceParams{}, gqlerr.New(handlers.CodeValidation, "autoPromoteSuppressEscalationPriorities cannot be empty")
		}
		priorities := make([]string, 0, len(input.AutoPromoteSuppressEscalationPriorities))
		for _, priority := range input.AutoPromoteSuppressEscalationPriorities {
			dbPriority, err := alertPriorityToDB(priority)
			if err != nil {
				return db.UpdateServiceParams{}, err
			}
			priorities = append(priorities, dbPriority)
		}
		params.AutoPromoteSuppressEscalationPriorities = priorities
	}

	return params, nil
}
