-- Create "organization_scim_settings" table
CREATE TABLE "organization_scim_settings" (
  "organization_id" uuid NOT NULL,
  "token_hash" text NOT NULL,
  "token_prefix" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("organization_id"),
  CONSTRAINT "organization_scim_settings_token_hash_key" UNIQUE ("token_hash"),
  CONSTRAINT "organization_scim_settings_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "scim_groups" table
CREATE TABLE "scim_groups" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "external_id" text NOT NULL,
  "display_name" text NOT NULL,
  "team_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "scim_groups_organization_id_external_id_key" UNIQUE ("organization_id", "external_id"),
  CONSTRAINT "scim_groups_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "scim_groups_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "scim_groups_team_id_idx" to table: "scim_groups"
CREATE INDEX "scim_groups_team_id_idx" ON "scim_groups" ("team_id");
-- Modify "users" table
ALTER TABLE "users" ADD COLUMN "scim_external_id" text NULL, ADD COLUMN "deprovisioned_at" timestamptz NULL, ADD CONSTRAINT "users_organization_id_scim_external_id_key" UNIQUE ("organization_id", "scim_external_id");
-- Create index "users_deprovisioned_at_idx" to table: "users"
CREATE INDEX "users_deprovisioned_at_idx" ON "users" ("deprovisioned_at");
-- Create "scim_group_members" table
CREATE TABLE "scim_group_members" (
  "scim_group_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("scim_group_id", "user_id"),
  CONSTRAINT "scim_group_members_scim_group_id_fkey" FOREIGN KEY ("scim_group_id") REFERENCES "scim_groups" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "scim_group_members_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "scim_group_members_user_id_idx" to table: "scim_group_members"
CREATE INDEX "scim_group_members_user_id_idx" ON "scim_group_members" ("user_id");
