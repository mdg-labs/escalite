-- name: HasTeamMembership :one
SELECT EXISTS(
    SELECT 1
    FROM team_memberships
    WHERE team_id = $1
      AND user_id = $2
      AND organization_id = $3
) AS has_membership;

-- name: GetTeamMembership :one
SELECT *
FROM team_memberships
WHERE team_id = $1
  AND user_id = $2
  AND organization_id = $3
LIMIT 1;

-- name: CreateTeamMembership :one
INSERT INTO team_memberships (
    id,
    team_id,
    user_id,
    organization_id
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: DeleteTeamMembership :exec
DELETE FROM team_memberships
WHERE team_id = $1
  AND user_id = $2
  AND organization_id = $3;

-- name: ListTeamMembershipsByUserIDs :many
SELECT *
FROM team_memberships
WHERE organization_id = $1
  AND user_id = ANY(sqlc.arg(user_ids)::uuid[])
ORDER BY user_id ASC, team_id ASC;
