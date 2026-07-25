package graph

import (
	"context"
	"errors"
	"strings"

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
	defaultAlertsListLimit = 100
	maxAlertsListLimit     = 200
)

func requireAuthSession(ctx context.Context) (auth.SessionContext, error) {
	sc, ok := auth.SessionFromContext(ctx)
	if !ok {
		return auth.SessionContext{}, gqlerr.New(handlers.CodeUnauthenticated, "authentication required")
	}
	return sc, nil
}

func (r *Resolver) loadAlertWithTeamAccess(
	ctx context.Context,
	queries *db.Queries,
	sc auth.SessionContext,
	alertID uuid.UUID,
) (db.Alert, db.Service, error) {
	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeNotFound, "alert not found")
		}
		r.logger.Error("load alert failed", "error", err)
		return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	service, err := queries.GetServiceByID(ctx, db.GetServiceByIDParams{
		ID:             alert.ServiceID,
		OrganizationID: sc.User.OrganizationID,
	})
	if err != nil {
		r.logger.Error("load service failed", "error", err)
		return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	if _, err := authz.CheckTeamAccess(ctx, queries, sc.User, service.TeamID); err != nil {
		if errors.Is(err, authz.ErrNotFound) || errors.Is(err, authz.ErrForbidden) {
			return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeForbidden, "access denied")
		}
		r.logger.Error("check team access failed", "error", err)
		return db.Alert{}, db.Service{}, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return alert, service, nil
}

func isEscalationValidationError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "must be positive") ||
		strings.Contains(msg, "exceeds maximum") ||
		strings.Contains(msg, "only triggered alerts can be snoozed") ||
		strings.Contains(msg, "escalation is exhausted") ||
		strings.Contains(msg, "no active escalation step") ||
		strings.Contains(msg, "cannot re-escalate a closed alert")
}

func alertsListLimit(limit *int) int32 {
	if limit == nil || *limit <= 0 {
		return defaultAlertsListLimit
	}
	if *limit > maxAlertsListLimit {
		return maxAlertsListLimit
	}
	return int32(*limit)
}

func alertStatusFilter(status *model.AlertStatus) pgtype.Text {
	if status == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: alertStatusToDB(*status), Valid: true}
}

func alertStatusToDB(status model.AlertStatus) string {
	switch status {
	case model.AlertStatusAcknowledged:
		return "acknowledged"
	case model.AlertStatusClosed:
		return "closed"
	default:
		return "triggered"
	}
}

func alertsFromDB(rows []db.Alert) []*model.Alert {
	result := make([]*model.Alert, 0, len(rows))
	for _, row := range rows {
		result = append(result, alertFromDB(row, nil))
	}
	return result
}

func (r *queryResolver) resolveAlert(ctx context.Context, id string) (*model.Alert, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	alertID, err := parseUUIDField(id, "id")
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	alert, service, err := r.loadAlertWithTeamAccess(ctx, queries, sc, alertID)
	if err != nil {
		return nil, err
	}
	_ = service

	var acknowledgedBy *db.User
	if alert.AcknowledgedByUserID.Valid {
		user, err := queries.GetUserByID(ctx, db.GetUserByIDParams{
			ID:             alert.AcknowledgedByUserID.Bytes,
			OrganizationID: sc.User.OrganizationID,
		})
		if err != nil {
			r.logger.Error("load alert acknowledger failed", "error", err)
			return nil, gqlerr.New(handlers.CodeInternal, "internal error")
		}
		acknowledgedBy = &user
	}

	return alertFromDB(alert, acknowledgedBy), nil
}

func (r *queryResolver) resolveAlerts(
	ctx context.Context,
	status *model.AlertStatus,
	limit *int,
) ([]*model.Alert, error) {
	sc, err := requireAuthSession(ctx)
	if err != nil {
		return nil, err
	}

	queries := db.New(r.pool)
	listLimit := alertsListLimit(limit)
	statusFilter := alertStatusFilter(status)

	var rows []db.Alert
	if authz.IsAdmin(sc.User.Role) {
		rows, err = queries.ListAlertsForOrgAdmin(ctx, db.ListAlertsForOrgAdminParams{
			OrganizationID: sc.User.OrganizationID,
			Limit:          listLimit,
			StatusFilter:   statusFilter,
		})
	} else {
		rows, err = queries.ListAlertsForTeamMember(ctx, db.ListAlertsForTeamMemberParams{
			OrganizationID: sc.User.OrganizationID,
			UserID:         sc.User.ID,
			Limit:          listLimit,
			StatusFilter:   statusFilter,
		})
	}
	if err != nil {
		r.logger.Error("list alerts failed", "error", err)
		return nil, gqlerr.New(handlers.CodeInternal, "internal error")
	}

	return alertsFromDB(rows), nil
}
