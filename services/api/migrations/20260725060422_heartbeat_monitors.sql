-- Create "heartbeat_monitors" table
CREATE TABLE "heartbeat_monitors" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "name" text NOT NULL,
  "interval_seconds" integer NOT NULL,
  "grace_seconds" integer NOT NULL,
  "token_hash" text NOT NULL,
  "prefix" text NOT NULL,
  "status" text NOT NULL DEFAULT 'healthy',
  "last_ping_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "heartbeat_monitors_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "heartbeat_monitors_token_hash_key" UNIQUE ("token_hash"),
  CONSTRAINT "heartbeat_monitors_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "heartbeat_monitors_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "heartbeat_monitors_grace_seconds_check" CHECK (grace_seconds > 0),
  CONSTRAINT "heartbeat_monitors_interval_seconds_check" CHECK (interval_seconds > 0),
  CONSTRAINT "heartbeat_monitors_status_check" CHECK (status = ANY (ARRAY['healthy'::text, 'overdue'::text, 'triggered'::text]))
);
-- Create index "heartbeat_monitors_organization_id_idx" to table: "heartbeat_monitors"
CREATE INDEX "heartbeat_monitors_organization_id_idx" ON "heartbeat_monitors" ("organization_id");
-- Create index "heartbeat_monitors_service_id_idx" to table: "heartbeat_monitors"
CREATE INDEX "heartbeat_monitors_service_id_idx" ON "heartbeat_monitors" ("service_id");
