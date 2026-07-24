-- Create "services" table
CREATE TABLE "services" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "services_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "services_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "services_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "services_organization_id_idx" to table: "services"
CREATE INDEX "services_organization_id_idx" ON "services" ("organization_id");
-- Create index "services_team_id_idx" to table: "services"
CREATE INDEX "services_team_id_idx" ON "services" ("team_id");
-- Create "integration_keys" table
CREATE TABLE "integration_keys" (
  "id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "token" text NOT NULL,
  "prefix" text NOT NULL,
  "plugin_name" text NOT NULL,
  "config" jsonb NOT NULL DEFAULT '{}',
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "integration_keys_token_key" UNIQUE ("token"),
  CONSTRAINT "integration_keys_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "integration_keys_plugin_name_idx" to table: "integration_keys"
CREATE INDEX "integration_keys_plugin_name_idx" ON "integration_keys" ("plugin_name");
-- Create index "integration_keys_service_id_idx" to table: "integration_keys"
CREATE INDEX "integration_keys_service_id_idx" ON "integration_keys" ("service_id");
