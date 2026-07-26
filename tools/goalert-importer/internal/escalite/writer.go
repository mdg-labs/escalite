package escalite

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Writer inserts mapped rows into an Escalite Postgres database.
type Writer struct {
	pool *pgxpool.Pool
}

func NewWriter(pool *pgxpool.Pool) *Writer {
	return &Writer{pool: pool}
}

func (w *Writer) VerifyOrganization(ctx context.Context, orgID uuid.UUID) error {
	var exists bool
	err := w.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id = $1)`, orgID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("verify organization: %w", err)
	}
	if !exists {
		return fmt.Errorf("organization %s not found", orgID)
	}
	return nil
}

func (w *Writer) VerifyTeam(ctx context.Context, orgID, teamID uuid.UUID) error {
	var exists bool
	err := w.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM teams WHERE id = $1 AND organization_id = $2)
	`, teamID, orgID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("verify team: %w", err)
	}
	if !exists {
		return fmt.Errorf("team %s not found in organization %s", teamID, orgID)
	}
	return nil
}

func (w *Writer) CreateTeam(ctx context.Context, orgID uuid.UUID, name string) (uuid.UUID, error) {
	teamID, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate team id: %w", err)
	}
	_, err = w.pool.Exec(ctx, `
		INSERT INTO teams (id, organization_id, name) VALUES ($1, $2, $3)
	`, teamID, orgID, name)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create team: %w", err)
	}
	return teamID, nil
}

func (w *Writer) InsertUser(ctx context.Context, id, orgID uuid.UUID, email, role string) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO users (id, organization_id, email, password_hash, role)
		VALUES ($1, $2, $3, NULL, $4)
	`, id, orgID, email, role)
	if err != nil {
		return fmt.Errorf("insert user %s: %w", email, err)
	}
	return nil
}

func (w *Writer) InsertTeamMembership(ctx context.Context, id, teamID, userID, orgID uuid.UUID) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO team_memberships (id, team_id, user_id, organization_id)
		VALUES ($1, $2, $3, $4)
	`, id, teamID, userID, orgID)
	if err != nil {
		return fmt.Errorf("insert team membership: %w", err)
	}
	return nil
}

func (w *Writer) InsertSchedule(ctx context.Context, id, orgID, teamID uuid.UUID, name, timezone string) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO schedules (id, organization_id, team_id, name, timezone)
		VALUES ($1, $2, $3, $4, $5)
	`, id, orgID, teamID, name, timezone)
	if err != nil {
		return fmt.Errorf("insert schedule %s: %w", name, err)
	}
	return nil
}

func (w *Writer) InsertRotation(
	ctx context.Context,
	id, scheduleID, orgID uuid.UUID,
	name string,
	layer int,
	rrule string,
	participants []byte,
) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO rotations (id, schedule_id, organization_id, name, layer, rrule, participants)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, scheduleID, orgID, name, layer, rrule, participants)
	if err != nil {
		return fmt.Errorf("insert rotation %s: %w", name, err)
	}
	return nil
}

func (w *Writer) InsertService(ctx context.Context, id, orgID, teamID uuid.UUID, name string) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO services (id, organization_id, team_id, name)
		VALUES ($1, $2, $3, $4)
	`, id, orgID, teamID, name)
	if err != nil {
		return fmt.Errorf("insert service %s: %w", name, err)
	}
	return nil
}

func (w *Writer) InsertEscalationPolicy(ctx context.Context, id, orgID, serviceID uuid.UUID, name string) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO escalation_policies (id, organization_id, service_id, name)
		VALUES ($1, $2, $3, $4)
	`, id, orgID, serviceID, name)
	if err != nil {
		return fmt.Errorf("insert escalation policy %s: %w", name, err)
	}
	return nil
}

func (w *Writer) InsertEscalationStep(
	ctx context.Context,
	id, policyID, orgID uuid.UUID,
	stepOrder int,
	delayMinutes int32,
	repeatLast bool,
	maxRepeats *int32,
) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO escalation_steps (
			id, escalation_policy_id, organization_id, step_order, delay_minutes, repeat_last_step, max_repeats
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, id, policyID, orgID, stepOrder, delayMinutes, repeatLast, maxRepeats)
	if err != nil {
		return fmt.Errorf("insert escalation step: %w", err)
	}
	return nil
}

func (w *Writer) InsertEscalationStepTarget(
	ctx context.Context,
	id, stepID, orgID uuid.UUID,
	targetType string,
	userID, scheduleID *uuid.UUID,
	webhookURL *string,
	channels []byte,
) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO escalation_step_targets (
			id, escalation_step_id, organization_id, target_type, user_id, schedule_id, webhook_url, channels
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, id, stepID, orgID, targetType, userID, scheduleID, webhookURL, channels)
	if err != nil {
		return fmt.Errorf("insert escalation step target: %w", err)
	}
	return nil
}

func (w *Writer) InsertAlert(
	ctx context.Context,
	id, orgID, serviceID uuid.UUID,
	status, dedupKey, summary, description string,
	createdAt interface{},
) error {
	_, err := w.pool.Exec(ctx, `
		INSERT INTO alerts (
			id, organization_id, service_id, status, dedup_key, summary, description, priority, escalation_state, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, 'high', '{}'::jsonb, $8, $8)
	`, id, orgID, serviceID, status, dedupKey, summary, description, createdAt)
	if err != nil {
		return fmt.Errorf("insert alert: %w", err)
	}
	return nil
}

// WithTx runs fn inside a transaction.
func (w *Writer) WithTx(ctx context.Context, fn func(pgx.Tx) error) error {
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
