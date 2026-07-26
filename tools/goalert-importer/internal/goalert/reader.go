package goalert

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// User row from GoAlert public.users.
type User struct {
	ID    uuid.UUID
	Name  string
	Email string
	Role  string
}

// Schedule row from GoAlert public.schedules.
type Schedule struct {
	ID          uuid.UUID
	Name        string
	Description string
	TimeZone    string
}

// Rotation row from GoAlert public.rotations with schedule link via schedule_rules.
type Rotation struct {
	ID           uuid.UUID
	ScheduleID   uuid.UUID
	Name         string
	Description  string
	Type         string
	ShiftLength  int64
	StartTime    time.Time
	Participants []uuid.UUID
}

// Service row from GoAlert public.services.
type Service struct {
	ID                 string
	Name               string
	Description        string
	EscalationPolicyID string
}

// EscalationPolicy row from GoAlert public.escalation_policies.
type EscalationPolicy struct {
	ID          string
	Name        string
	Description string
	Repeat      int32
}

// EscalationStep row from GoAlert public.escalation_policy_steps.
type EscalationStep struct {
	ID                 string
	EscalationPolicyID string
	Delay              int32
	StepNumber         int32
}

// EscalationAction row from GoAlert public.escalation_policy_actions.
type EscalationAction struct {
	StepID      string
	UserID      *uuid.UUID
	ScheduleID  *uuid.UUID
	RotationID  *uuid.UUID
	ChannelDest *string
}

// Alert row from GoAlert public.alerts (historical import).
type Alert struct {
	ID        int64
	Summary   string
	Details   string
	ServiceID string
	Source    string
	Status    string
	DedupKey  string
	CreatedAt time.Time
}

// Counts summarizes source row counts for dry-run output.
type Counts struct {
	Users              int
	Schedules          int
	Rotations          int
	Services           int
	EscalationPolicies int
	EscalationSteps    int
	EscalationActions  int
	Alerts             int
}

// Reader loads entities from a GoAlert Postgres database.
type Reader struct {
	pool *pgxpool.Pool
}

func NewReader(pool *pgxpool.Pool) *Reader {
	return &Reader{pool: pool}
}

func (r *Reader) Counts(ctx context.Context, includeAlerts bool) (Counts, error) {
	var counts Counts
	queries := []struct {
		sql    string
		target *int
	}{
		{"SELECT count(*) FROM users", &counts.Users},
		{"SELECT count(*) FROM schedules", &counts.Schedules},
		{"SELECT count(*) FROM rotations", &counts.Rotations},
		{"SELECT count(*) FROM services", &counts.Services},
		{"SELECT count(*) FROM escalation_policies", &counts.EscalationPolicies},
		{"SELECT count(*) FROM escalation_policy_steps", &counts.EscalationSteps},
		{"SELECT count(*) FROM escalation_policy_actions", &counts.EscalationActions},
	}
	for _, q := range queries {
		if err := r.pool.QueryRow(ctx, q.sql).Scan(q.target); err != nil {
			return Counts{}, fmt.Errorf("count query failed: %w", err)
		}
	}
	if includeAlerts {
		if err := r.pool.QueryRow(ctx, "SELECT count(*) FROM alerts").Scan(&counts.Alerts); err != nil {
			return Counts{}, fmt.Errorf("count alerts failed: %w", err)
		}
	}
	return counts, nil
}

func (r *Reader) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, email, role::text
		FROM users
		ORDER BY email
	`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Role); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *Reader) ListSchedules(ctx context.Context) ([]Schedule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, time_zone
		FROM schedules
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.TimeZone); err != nil {
			return nil, fmt.Errorf("scan schedule: %w", err)
		}
		schedules = append(schedules, s)
	}
	return schedules, rows.Err()
}

func (r *Reader) ListRotations(ctx context.Context) ([]Rotation, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			rot.id,
			rule.schedule_id,
			rot.name,
			rot.description,
			rot.type::text,
			rot.shift_length,
			rot.start_time,
			COALESCE(
				ARRAY(
					SELECT p.user_id
					FROM rotation_participants p
					WHERE p.rotation_id = rot.id
					ORDER BY p.position
				),
				'{}'::uuid[]
			) AS participants
		FROM rotations rot
		INNER JOIN schedule_rules rule ON rule.tgt_rotation_id = rot.id
		ORDER BY rule.schedule_id, rot.name
	`)
	if err != nil {
		return nil, fmt.Errorf("list rotations: %w", err)
	}
	defer rows.Close()

	var rotations []Rotation
	for rows.Next() {
		var rot Rotation
		if err := rows.Scan(
			&rot.ID,
			&rot.ScheduleID,
			&rot.Name,
			&rot.Description,
			&rot.Type,
			&rot.ShiftLength,
			&rot.StartTime,
			&rot.Participants,
		); err != nil {
			return nil, fmt.Errorf("scan rotation: %w", err)
		}
		rotations = append(rotations, rot)
	}
	return rotations, rows.Err()
}

func (r *Reader) ListServices(ctx context.Context) ([]Service, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, escalation_policy_id
		FROM services
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list services: %w", err)
	}
	defer rows.Close()

	var services []Service
	for rows.Next() {
		var s Service
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.EscalationPolicyID); err != nil {
			return nil, fmt.Errorf("scan service: %w", err)
		}
		services = append(services, s)
	}
	return services, rows.Err()
}

func (r *Reader) ListEscalationPolicies(ctx context.Context) ([]EscalationPolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, description, repeat
		FROM escalation_policies
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list escalation policies: %w", err)
	}
	defer rows.Close()

	var policies []EscalationPolicy
	for rows.Next() {
		var p EscalationPolicy
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Repeat); err != nil {
			return nil, fmt.Errorf("scan escalation policy: %w", err)
		}
		policies = append(policies, p)
	}
	return policies, rows.Err()
}

func (r *Reader) ListEscalationSteps(ctx context.Context) ([]EscalationStep, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, escalation_policy_id, delay, step_number
		FROM escalation_policy_steps
		ORDER BY escalation_policy_id, step_number
	`)
	if err != nil {
		return nil, fmt.Errorf("list escalation steps: %w", err)
	}
	defer rows.Close()

	var steps []EscalationStep
	for rows.Next() {
		var s EscalationStep
		if err := rows.Scan(&s.ID, &s.EscalationPolicyID, &s.Delay, &s.StepNumber); err != nil {
			return nil, fmt.Errorf("scan escalation step: %w", err)
		}
		steps = append(steps, s)
	}
	return steps, rows.Err()
}

func (r *Reader) ListEscalationActions(ctx context.Context) ([]EscalationAction, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			a.escalation_policy_step_id,
			a.user_id,
			a.schedule_id,
			a.rotation_id,
			ch.dest
		FROM escalation_policy_actions a
		LEFT JOIN notification_channels ch ON a.channel_id = ch.id
		ORDER BY a.escalation_policy_step_id
	`)
	if err != nil {
		return nil, fmt.Errorf("list escalation actions: %w", err)
	}
	defer rows.Close()

	var actions []EscalationAction
	for rows.Next() {
		var a EscalationAction
		if err := rows.Scan(&a.StepID, &a.UserID, &a.ScheduleID, &a.RotationID, &a.ChannelDest); err != nil {
			return nil, fmt.Errorf("scan escalation action: %w", err)
		}
		actions = append(actions, a)
	}
	return actions, rows.Err()
}

func (r *Reader) ListAlerts(ctx context.Context) ([]Alert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, summary, details, service_id, source, status, dedup_key, created_at
		FROM alerts
		ORDER BY created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []Alert
	for rows.Next() {
		var a Alert
		if err := rows.Scan(&a.ID, &a.Summary, &a.Details, &a.ServiceID, &a.Source, &a.Status, &a.DedupKey, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// ScheduleIDForRotation returns the schedule linked to a rotation via schedule_rules.
func (r *Reader) ScheduleIDForRotation(ctx context.Context, rotationID uuid.UUID) (uuid.UUID, error) {
	var scheduleID uuid.UUID
	err := r.pool.QueryRow(ctx, `
		SELECT schedule_id FROM schedule_rules WHERE tgt_rotation_id = $1 LIMIT 1
	`, rotationID).Scan(&scheduleID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("schedule for rotation %s: %w", rotationID, err)
	}
	return scheduleID, nil
}
