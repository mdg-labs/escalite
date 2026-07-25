-- Create "incidents" table
CREATE TABLE "incidents" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "title" text NOT NULL,
  "status" text NOT NULL DEFAULT 'investigating',
  "created_by_user_id" uuid NOT NULL,
  "resolved_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "incidents_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "incidents_created_by_user_id_organization_id_fkey" FOREIGN KEY ("created_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "incidents_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incidents_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incidents_status_check" CHECK (status = ANY (ARRAY['investigating'::text, 'identified'::text, 'monitoring'::text, 'resolved'::text]))
);
-- Create index "incidents_organization_id_idx" to table: "incidents"
CREATE INDEX "incidents_organization_id_idx" ON "incidents" ("organization_id");
-- Create index "incidents_status_idx" to table: "incidents"
CREATE INDEX "incidents_status_idx" ON "incidents" ("status");
-- Create index "incidents_team_id_idx" to table: "incidents"
CREATE INDEX "incidents_team_id_idx" ON "incidents" ("team_id");
-- Modify "alerts" table
ALTER TABLE "alerts" ADD COLUMN "incident_id" uuid NULL, ADD CONSTRAINT "alerts_incident_id_organization_id_fkey" FOREIGN KEY ("incident_id", "organization_id") REFERENCES "incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;
-- Create index "alerts_incident_id_idx" to table: "alerts"
CREATE INDEX "alerts_incident_id_idx" ON "alerts" ("incident_id");
-- Create "incident_role_definitions" table
CREATE TABLE "incident_role_definitions" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" text NOT NULL,
  "sort_order" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "incident_role_definitions_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "incident_role_definitions_organization_id_name_key" UNIQUE ("organization_id", "name"),
  CONSTRAINT "incident_role_definitions_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "incident_role_definitions_organization_id_idx" to table: "incident_role_definitions"
CREATE INDEX "incident_role_definitions_organization_id_idx" ON "incident_role_definitions" ("organization_id");
-- Create "incident_role_assignments" table
CREATE TABLE "incident_role_assignments" (
  "id" uuid NOT NULL,
  "incident_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "role_definition_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "assigned_by_user_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "incident_role_assignments_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "incident_role_assignments_incident_id_role_definition_id_key" UNIQUE ("incident_id", "role_definition_id"),
  CONSTRAINT "incident_role_assignments_assigned_by_user_id_organization_id_f" FOREIGN KEY ("assigned_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "incident_role_assignments_incident_id_organization_id_fkey" FOREIGN KEY ("incident_id", "organization_id") REFERENCES "incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incident_role_assignments_role_definition_id_organization_id_fk" FOREIGN KEY ("role_definition_id", "organization_id") REFERENCES "incident_role_definitions" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incident_role_assignments_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "incident_role_assignments_incident_id_idx" to table: "incident_role_assignments"
CREATE INDEX "incident_role_assignments_incident_id_idx" ON "incident_role_assignments" ("incident_id");
-- Create "timeline_events" table
CREATE TABLE "timeline_events" (
  "id" uuid NOT NULL,
  "incident_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "actor_id" uuid NULL,
  "event_type" text NOT NULL,
  "body" text NOT NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "timeline_events_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "timeline_events_actor_id_organization_id_fkey" FOREIGN KEY ("actor_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "timeline_events_incident_id_organization_id_fkey" FOREIGN KEY ("incident_id", "organization_id") REFERENCES "incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "timeline_events_event_type_check" CHECK (event_type = ANY (ARRAY['declared'::text, 'status_changed'::text, 'note'::text, 'role_assigned'::text, 'role_unassigned'::text]))
);
-- Create index "timeline_events_incident_id_created_at_idx" to table: "timeline_events"
CREATE INDEX "timeline_events_incident_id_created_at_idx" ON "timeline_events" ("incident_id", "created_at");
-- Create index "timeline_events_incident_id_idx" to table: "timeline_events"
CREATE INDEX "timeline_events_incident_id_idx" ON "timeline_events" ("incident_id");
