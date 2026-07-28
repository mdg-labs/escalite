package statuspageapi

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mdg-labs/escalite/services/engine/internal/jobs"
)

// JobProducer inserts River jobs.
type JobProducer interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
}

// EnqueueIncidentNotify schedules subscriber email delivery for a status page incident update.
func EnqueueIncidentNotify(
	ctx context.Context,
	producer JobProducer,
	organizationID, statusPageIncidentID, updateID uuid.UUID,
) error {
	if producer == nil {
		return nil
	}
	if _, err := producer.Insert(ctx, jobs.StatusPageIncidentNotifyArgs{
		OrganizationID:       organizationID,
		StatusPageIncidentID: statusPageIncidentID,
		UpdateID:             updateID,
	}, nil); err != nil {
		return fmt.Errorf("enqueue status page incident notify: %w", err)
	}
	return nil
}
