-- Create "organizations" table
CREATE TABLE "organizations" (
  "id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create "teams" table
CREATE TABLE "teams" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "teams_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "teams_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "teams_organization_id_idx" to table: "teams"
CREATE INDEX "teams_organization_id_idx" ON "teams" ("organization_id");
-- Create "users" table
CREATE TABLE "users" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "email" text NOT NULL,
  "password_hash" text NULL,
  "role" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "users_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "users_organization_id_email_key" UNIQUE ("organization_id", "email"),
  CONSTRAINT "users_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "users_role_check" CHECK (role = ANY (ARRAY['admin'::text, 'member'::text]))
);
-- Create index "users_organization_id_idx" to table: "users"
CREATE INDEX "users_organization_id_idx" ON "users" ("organization_id");
-- Create "team_memberships" table
CREATE TABLE "team_memberships" (
  "id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "team_memberships_team_id_user_id_key" UNIQUE ("team_id", "user_id"),
  CONSTRAINT "team_memberships_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "team_memberships_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "team_memberships_team_id_idx" to table: "team_memberships"
CREATE INDEX "team_memberships_team_id_idx" ON "team_memberships" ("team_id");
-- Create index "team_memberships_user_id_idx" to table: "team_memberships"
CREATE INDEX "team_memberships_user_id_idx" ON "team_memberships" ("user_id");
