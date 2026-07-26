-- name: GetOrganizationSamlSettings :one
SELECT *
FROM organization_saml_settings
WHERE organization_id = $1;

-- name: GetEnabledOrganizationSamlSettings :one
SELECT *
FROM organization_saml_settings
WHERE enabled = true
ORDER BY created_at
LIMIT 1;

-- name: UpsertOrganizationSamlSettings :one
INSERT INTO organization_saml_settings (
    organization_id,
    enabled,
    idp_entity_id,
    idp_sso_url,
    idp_certificate_pem,
    sp_certificate_pem,
    sp_private_key_ciphertext,
    sp_encryption_key_id,
    certificate_hint
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (organization_id) DO UPDATE SET
    enabled = EXCLUDED.enabled,
    idp_entity_id = EXCLUDED.idp_entity_id,
    idp_sso_url = EXCLUDED.idp_sso_url,
    idp_certificate_pem = EXCLUDED.idp_certificate_pem,
    sp_certificate_pem = EXCLUDED.sp_certificate_pem,
    sp_private_key_ciphertext = EXCLUDED.sp_private_key_ciphertext,
    sp_encryption_key_id = EXCLUDED.sp_encryption_key_id,
    certificate_hint = EXCLUDED.certificate_hint,
    updated_at = now()
RETURNING *;
