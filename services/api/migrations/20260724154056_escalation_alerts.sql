-- Create "alerts" table
CREATE TABLE "alerts" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "integration_key_id" uuid NULL,
  "status" text NOT NULL,
  "dedup_key" text NOT NULL,
  "summary" text NOT NULL,
  "description" text NULL,
  "priority" text NOT NULL DEFAULT 'high',
  "event_count" integer NOT NULL DEFAULT 1,
  "escalation_state" jsonb NOT NULL DEFAULT '{}',
  "acknowledged_at" timestamptz NULL,
  "closed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "alerts_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "alerts_integration_key_id_fkey" FOREIGN KEY ("integration_key_id") REFERENCES "integration_keys" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "alerts_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "alerts_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "alerts_priority_check" CHECK (priority = ANY (ARRAY['low'::text, 'high'::text])),
  CONSTRAINT "alerts_status_check" CHECK (status = ANY (ARRAY['triggered'::text, 'acknowledged'::text, 'closed'::text]))
);
-- Create index "alerts_organization_id_idx" to table: "alerts"
CREATE INDEX "alerts_organization_id_idx" ON "alerts" ("organization_id");
-- Create index "alerts_service_id_dedup_key_idx" to table: "alerts"
CREATE INDEX "alerts_service_id_dedup_key_idx" ON "alerts" ("service_id", "dedup_key");
-- Create index "alerts_service_id_idx" to table: "alerts"
CREATE INDEX "alerts_service_id_idx" ON "alerts" ("service_id");
-- Create index "alerts_status_idx" to table: "alerts"
CREATE INDEX "alerts_status_idx" ON "alerts" ("status");
-- Create "escalation_policies" table
CREATE TABLE "escalation_policies" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "escalation_policies_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "escalation_policies_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_policies_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "escalation_policies_organization_id_idx" to table: "escalation_policies"
CREATE INDEX "escalation_policies_organization_id_idx" ON "escalation_policies" ("organization_id");
-- Create index "escalation_policies_service_id_idx" to table: "escalation_policies"
CREATE INDEX "escalation_policies_service_id_idx" ON "escalation_policies" ("service_id");
-- Create "escalation_steps" table
CREATE TABLE "escalation_steps" (
  "id" uuid NOT NULL,
  "escalation_policy_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "step_order" integer NOT NULL,
  "delay_minutes" integer NOT NULL DEFAULT 0,
  "repeat_last_step" boolean NOT NULL DEFAULT false,
  "max_repeats" integer NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "escalation_steps_escalation_policy_id_step_order_key" UNIQUE ("escalation_policy_id", "step_order"),
  CONSTRAINT "escalation_steps_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "escalation_steps_escalation_policy_id_organization_id_fkey" FOREIGN KEY ("escalation_policy_id", "organization_id") REFERENCES "escalation_policies" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "escalation_steps_escalation_policy_id_idx" to table: "escalation_steps"
CREATE INDEX "escalation_steps_escalation_policy_id_idx" ON "escalation_steps" ("escalation_policy_id");
-- Create "notification_attempts" table
CREATE TABLE "notification_attempts" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "alert_id" uuid NOT NULL,
  "escalation_step_id" uuid NULL,
  "channel" text NOT NULL,
  "status" text NOT NULL,
  "recipient" jsonb NOT NULL DEFAULT '{}',
  "error_message" text NULL,
  "sent_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "notification_attempts_alert_id_organization_id_fkey" FOREIGN KEY ("alert_id", "organization_id") REFERENCES "alerts" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "notification_attempts_escalation_step_id_organization_id_fkey" FOREIGN KEY ("escalation_step_id", "organization_id") REFERENCES "escalation_steps" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "notification_attempts_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "notification_attempts_status_check" CHECK (status = ANY (ARRAY['pending'::text, 'sent'::text, 'failed'::text]))
);
-- Create index "notification_attempts_alert_id_idx" to table: "notification_attempts"
CREATE INDEX "notification_attempts_alert_id_idx" ON "notification_attempts" ("alert_id");
-- Create index "notification_attempts_organization_id_idx" to table: "notification_attempts"
CREATE INDEX "notification_attempts_organization_id_idx" ON "notification_attempts" ("organization_id");
