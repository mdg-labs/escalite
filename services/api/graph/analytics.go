package graph

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/api/graph/model"
	"github.com/mdg-labs/escalite/services/api/internal/auth"
	"github.com/mdg-labs/escalite/services/api/internal/authz"
	"github.com/mdg-labs/escalite/services/api/internal/db"
	"github.com/mdg-labs/escalite/services/api/internal/gqlerr"
	"github.com/mdg-labs/escalite/services/api/internal/handlers"
)

const (
	analyticsWindow7Days  = 7
	analyticsWindow30Days = 30
)

func analyticsSettingsFromDB(settings *db.OrganizationAnalyticsSetting) *model.AnalyticsSettings {
	if settings == nil {
		return &model.AnalyticsSettings{ExcludeMaintenanceWindowAlerts: false}
	}
	return &model.AnalyticsSettings{
		ExcludeMaintenanceWindowAlerts: settings.ExcludeMaintenanceWindowAlerts,
	}
}

func loadOrganizationAnalyticsSettings(
	ctx context.Context,
	queries *db.Queries,
	orgID uuid.UUID,
) (*db.OrganizationAnalyticsSetting, bool, error) {
	settings, err := queries.GetOrganizationAnalyticsSettings(ctx, orgID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &settings, true, nil
}

func excludeMaintenanceWindowAlertsConfigured(settings *db.OrganizationAnalyticsSetting, found bool) bool {
	if !found || settings == nil {
		return false
	}
	return settings.ExcludeMaintenanceWindowAlerts
}

func optionalMetricSeconds(value float64, count int32) *float64 {
	if count == 0 {
		return nil
	}
	v := value
	return &v
}

func rollupFromDB(windowDays int, row db.ComputeAlertAnalyticsRollupRow) *model.AlertAnalyticsRollup {
	return &model.AlertAnalyticsRollup{
		WindowDays:        windowDays,
		MttaSeconds:       optionalMetricSeconds(row.MttaSeconds, row.AcknowledgedCount),
		AcknowledgedCount: int(row.AcknowledgedCount),
		MttrSeconds:       optionalMetricSeconds(row.MttrSeconds, row.ResolvedCount),
		ResolvedCount:     int(row.ResolvedCount),
	}
}

func (r *Resolver) computeAnalyticsRollup(
	ctx context.Context,
	queries *db.Queries,
	orgID uuid.UUID,
	windowDays int,
	teamID pgtype.UUID,
	serviceID pgtype.UUID,
	excludeMaintenance bool,
) (*model.AlertAnalyticsRollup, error) {
	windowStart := pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, -windowDays), Valid: true}
	row, err := queries.ComputeAlertAnalyticsRollup(ctx, db.ComputeAlertAnalyticsRollupParams{
		OrganizationID:                 orgID,
		WindowStart:                    windowStart,
		TeamID:                         teamID,
		ServiceID:                      serviceID,
		ExcludeMaintenanceWindowAlerts: excludeMaintenance,
	})
	if err != nil {
		r.logger.Error("compute alert analytics rollup failed", "error", err, "windowDays", windowDays)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}
	return rollupFromDB(windowDays, row), nil
}

func (r *Resolver) resolveAlertAnalyticsScope(
	ctx context.Context,
	queries *db.Queries,
	sc auth.SessionContext,
	teamID *string,
	serviceID *string,
) (pgtype.UUID, pgtype.UUID, error) {
	var teamFilter pgtype.UUID
	var serviceFilter pgtype.UUID

	if serviceID != nil && strings.TrimSpace(*serviceID) != "" {
		parsedServiceID, err := parseUUIDField(*serviceID, "serviceId")
		if err != nil {
			return pgtype.UUID{}, pgtype.UUID{}, err
		}
		service, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
			ID:             parsedServiceID,
			OrganizationID: sc.User.OrganizationID,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return pgtype.UUID{}, pgtype.UUID{}, gqlerr.New(handlers.CodeNotFound, "service not found")
			}
			r.logger.Error("load service for analytics failed", "error", err)
			return pgtype.UUID{}, pgtype.UUID{}, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		if _, err := authz.CheckTeamAccess(ctx, queries, sc.User, service.TeamID); err != nil {
			if errors.Is(err, authz.ErrNotFound) || errors.Is(err, authz.ErrForbidden) {
				return pgtype.UUID{}, pgtype.UUID{}, gqlerr.New(handlers.CodeForbidden, "access denied")
			}
			r.logger.Error("check team access for analytics failed", "error", err)
			return pgtype.UUID{}, pgtype.UUID{}, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		serviceFilter = pgtype.UUID{Bytes: parsedServiceID, Valid: true}
		teamFilter = pgtype.UUID{Bytes: service.TeamID, Valid: true}
		return teamFilter, serviceFilter, nil
	}

	if teamID != nil && strings.TrimSpace(*teamID) != "" {
		parsedTeamID, err := parseUUIDField(*teamID, "teamId")
		if err != nil {
			return pgtype.UUID{}, pgtype.UUID{}, err
		}
		if _, err := authz.CheckTeamAccess(ctx, queries, sc.User, parsedTeamID); err != nil {
			if errors.Is(err, authz.ErrNotFound) || errors.Is(err, authz.ErrForbidden) {
				return pgtype.UUID{}, pgtype.UUID{}, gqlerr.New(handlers.CodeForbidden, "access denied")
			}
			r.logger.Error("check team access for analytics failed", "error", err)
			return pgtype.UUID{}, pgtype.UUID{}, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		teamFilter = pgtype.UUID{Bytes: parsedTeamID, Valid: true}
		return teamFilter, serviceFilter, nil
	}

	if !authz.IsAdmin(sc.User.Role) {
		return pgtype.UUID{}, pgtype.UUID{}, gqlerr.New(handlers.CodeForbidden, "teamId or serviceId is required")
	}

	return teamFilter, serviceFilter, nil
}
