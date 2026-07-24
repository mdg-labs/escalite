-- name: BootstrapOrganizationWithAdmin :one
WITH new_org AS (
    INSERT INTO organizations (id, name)
    VALUES (sqlc.arg(org_id), sqlc.arg(org_name))
    RETURNING id, name
),
new_user AS (
    INSERT INTO users (id, organization_id, email, password_hash, role)
    VALUES (sqlc.arg(user_id), (SELECT id FROM new_org), sqlc.arg(email), sqlc.arg(password_hash), 'admin')
    RETURNING id, organization_id, email, role
)
SELECT
    new_org.id AS organization_id,
    new_org.name AS organization_name,
    new_user.id AS user_id,
    new_user.email AS user_email,
    new_user.role AS user_role
FROM new_org, new_user;

-- name: CreateUser :one
INSERT INTO users (id, organization_id, email, password_hash, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateTeam :one
INSERT INTO teams (id, organization_id, name)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateService :one
INSERT INTO services (id, organization_id, team_id, name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateEscalationPolicy :one
INSERT INTO escalation_policies (id, organization_id, service_id, name)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: CreateEscalationStep :one
INSERT INTO escalation_steps (
    id,
    escalation_policy_id,
    organization_id,
    step_order,
    delay_minutes,
    repeat_last_step,
    max_repeats
) VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: CreateEscalationStepTarget :one
INSERT INTO escalation_step_targets (
    id,
    escalation_step_id,
    organization_id,
    target_type,
    user_id,
    schedule_id,
    webhook_url,
    channels
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: CreateTriggeredAlert :one
INSERT INTO alerts (
    id,
    organization_id,
    service_id,
    integration_key_id,
    status,
    dedup_key,
    summary,
    description,
    priority,
    escalation_state
) VALUES (
    $1,
    $2,
    $3,
    $4,
    'triggered',
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING *;
