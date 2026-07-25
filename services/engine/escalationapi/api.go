package escalationapi

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/mdg-labs/escalite/services/engine/internal/db"
	"github.com/mdg-labs/escalite/services/engine/internal/escalation"
)

// TriggeredAlertParams configures a new triggered alert.
type TriggeredAlertParams struct {
	AlertID        uuid.UUID
	OrganizationID uuid.UUID
	ServiceID      uuid.UUID
	DedupKey       string
	Summary        string
	Description    pgtype.Text
	Priority       string
}

// JobProducer inserts and cancels River jobs.
type JobProducer interface {
	Insert(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error)
	CancelJob(ctx context.Context, jobID int64) error
}

// SnoozeAlert delays the next escalation step for a triggered alert.
func SnoozeAlert(
	ctx context.Context,
	pool *pgxpool.Pool,
	jobs JobProducer,
	alertID, organizationID uuid.UUID,
	durationMinutes int32,
) error {
	queries := db.New(pool)
	_, err := escalation.SnoozeEscalation(ctx, queries, jobs, jobs, alertID, organizationID, durationMinutes)
	return err
}

// ReEscalateAlert resets escalation to step 1 and enqueues step-1 notifications.
func ReEscalateAlert(
	ctx context.Context,
	pool *pgxpool.Pool,
	jobs JobProducer,
	alertID, organizationID uuid.UUID,
) error {
	queries := db.New(pool)
	_, err := escalation.ReEscalateAlert(ctx, queries, jobs, jobs, alertID, organizationID)
	return err
}

// CreateTriggeredAlert inserts a triggered alert and schedules step-1 escalation.
func CreateTriggeredAlert(
	ctx context.Context,
	pool *pgxpool.Pool,
	jobs JobProducer,
	params TriggeredAlertParams,
) error {
	_, err := escalation.CreateTriggeredAlert(ctx, pool, jobs, escalation.CreateTriggeredAlertParams{
		AlertID:        params.AlertID,
		OrganizationID: params.OrganizationID,
		ServiceID:      params.ServiceID,
		DedupKey:       params.DedupKey,
		Summary:        params.Summary,
		Description:    params.Description,
		Priority:       params.Priority,
	})
	return err
}

// RenotifyCollapsedAlert schedules repeat notifications for a dedup-collapsed alert,
// skipping users who already acknowledged.
func RenotifyCollapsedAlert(
	ctx context.Context,
	pool *pgxpool.Pool,
	jobs JobProducer,
	alertID, organizationID uuid.UUID,
) error {
	queries := db.New(pool)
	alert, err := queries.GetAlertByID(ctx, db.GetAlertByIDParams{
		ID:             alertID,
		OrganizationID: organizationID,
	})
	if err != nil {
		return err
	}
	return escalation.RenotifyCollapsedAlert(ctx, queries, jobs, alert)
}

// AcknowledgeAlert marks an alert acknowledged and cancels pending escalation.
func AcknowledgeAlert(
	ctx context.Context,
	pool *pgxpool.Pool,
	jobs JobProducer,
	alertID, organizationID, acknowledgedBy uuid.UUID,
) error {
	queries := db.New(pool)
	_, err := escalation.AcknowledgeAlert(ctx, queries, jobs, alertID, organizationID, acknowledgedBy)
	return err
}
