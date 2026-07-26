-- Create "organization_saml_settings" table
CREATE TABLE "organization_saml_settings" (
  "organization_id" uuid NOT NULL,
  "enabled" boolean NOT NULL DEFAULT false,
  "idp_entity_id" text NOT NULL,
  "idp_sso_url" text NOT NULL,
  "idp_certificate_pem" text NOT NULL,
  "sp_certificate_pem" text NOT NULL,
  "sp_private_key_ciphertext" bytea NOT NULL,
  "sp_encryption_key_id" text NOT NULL,
  "certificate_hint" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("organization_id"),
  CONSTRAINT "organization_saml_settings_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
