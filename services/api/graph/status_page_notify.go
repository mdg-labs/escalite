package graph

import (
	"context"

	"github.com/google/uuid"

	"github.com/mdg-labs/escalite/services/engine/statuspageapi"
)

func (r *Resolver) enqueueStatusPageIncidentNotify(
	ctx context.Context,
	organizationID, statusPageIncidentID, updateID uuid.UUID,
) {
	if r.jobs == nil {
		return
	}
	if err := statuspageapi.EnqueueIncidentNotify(ctx, r.jobs, organizationID, statusPageIncidentID, updateID); err != nil {
		r.logger.Error(
			"enqueue status page incident notify failed",
			"error", err,
			"status_page_incident_id", statusPageIncidentID,
			"update_id", updateID,
			"organization_id", organizationID,
		)
	}
}
