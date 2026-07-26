-- name: ListScimGroupMemberUserIDs :many
SELECT user_id
FROM scim_group_members
WHERE scim_group_id = $1
  AND organization_id = $2
ORDER BY created_at ASC;

-- name: AddScimGroupMember :exec
INSERT INTO scim_group_members (
    scim_group_id,
    user_id,
    organization_id
) VALUES (
    $1, $2, $3
)
ON CONFLICT DO NOTHING;

-- name: RemoveScimGroupMember :exec
DELETE FROM scim_group_members
WHERE scim_group_id = $1
  AND user_id = $2
  AND organization_id = $3;

-- name: RemoveAllScimGroupMembers :exec
DELETE FROM scim_group_members
WHERE scim_group_id = $1
  AND organization_id = $2;
