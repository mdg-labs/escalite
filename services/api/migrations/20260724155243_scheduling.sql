-- Create "schedules" table
CREATE TABLE "schedules" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "name" text NOT NULL,
  "timezone" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "schedules_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "schedules_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "schedules_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "schedules_timezone_check" CHECK (timezone(timezone, '2000-01-01 00:00:00+00'::timestamp with time zone) IS NOT NULL)
);
-- Create index "schedules_organization_id_idx" to table: "schedules"
CREATE INDEX "schedules_organization_id_idx" ON "schedules" ("organization_id");
-- Create index "schedules_team_id_idx" to table: "schedules"
CREATE INDEX "schedules_team_id_idx" ON "schedules" ("team_id");
-- Create "rotations" table
CREATE TABLE "rotations" (
  "id" uuid NOT NULL,
  "schedule_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" text NOT NULL,
  "layer" integer NOT NULL,
  "rrule" text NOT NULL,
  "participants" jsonb NOT NULL DEFAULT '[]',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "rotations_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "rotations_schedule_id_layer_key" UNIQUE ("schedule_id", "layer"),
  CONSTRAINT "rotations_schedule_id_organization_id_fkey" FOREIGN KEY ("schedule_id", "organization_id") REFERENCES "schedules" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "rotations_organization_id_idx" to table: "rotations"
CREATE INDEX "rotations_organization_id_idx" ON "rotations" ("organization_id");
-- Create index "rotations_schedule_id_idx" to table: "rotations"
CREATE INDEX "rotations_schedule_id_idx" ON "rotations" ("schedule_id");
-- Create "overrides" table
CREATE TABLE "overrides" (
  "id" uuid NOT NULL,
  "schedule_id" uuid NOT NULL,
  "rotation_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "replaced_user_id" uuid NULL,
  "starts_at" timestamptz NOT NULL,
  "ends_at" timestamptz NOT NULL,
  "created_by_user_id" uuid NOT NULL,
  "approved_by_user_id" uuid NULL,
  "deleted_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "overrides_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "overrides_approved_by_user_id_organization_id_fkey" FOREIGN KEY ("approved_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "overrides_created_by_user_id_organization_id_fkey" FOREIGN KEY ("created_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "overrides_replaced_user_id_organization_id_fkey" FOREIGN KEY ("replaced_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "overrides_rotation_id_organization_id_fkey" FOREIGN KEY ("rotation_id", "organization_id") REFERENCES "rotations" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "overrides_schedule_id_organization_id_fkey" FOREIGN KEY ("schedule_id", "organization_id") REFERENCES "schedules" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "overrides_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "overrides_window_check" CHECK (ends_at > starts_at)
);
-- Create index "overrides_active_idx" to table: "overrides"
CREATE INDEX "overrides_active_idx" ON "overrides" ("schedule_id", "starts_at", "ends_at") WHERE (deleted_at IS NULL);
-- Create index "overrides_rotation_id_idx" to table: "overrides"
CREATE INDEX "overrides_rotation_id_idx" ON "overrides" ("rotation_id");
-- Create index "overrides_schedule_id_idx" to table: "overrides"
CREATE INDEX "overrides_schedule_id_idx" ON "overrides" ("schedule_id");
-- Create index "overrides_schedule_id_starts_at_ends_at_idx" to table: "overrides"
CREATE INDEX "overrides_schedule_id_starts_at_ends_at_idx" ON "overrides" ("schedule_id", "starts_at", "ends_at");
