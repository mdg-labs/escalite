package escalation

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
)

func notificationsSuppressed(
	ctx context.Context,
	q db.Querier,
	serviceID, organizationID uuid.UUID,
	at time.Time,
) (bool, error) {
	active, err := q.ServiceHasActiveNotificationSuppression(ctx, db.ServiceHasActiveNotificationSuppressionParams{
		ServiceID:      serviceID,
		OrganizationID: organizationID,
		StartsAt:       pgtype.Timestamptz{Time: at.UTC(), Valid: true},
	})
	if err != nil {
		return false, err
	}
	return active, nil
}
