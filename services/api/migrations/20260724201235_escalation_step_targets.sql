-- Create "escalation_step_targets" table
CREATE TABLE "escalation_step_targets" (
  "id" uuid NOT NULL,
  "escalation_step_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "target_type" text NOT NULL,
  "user_id" uuid NULL,
  "schedule_id" uuid NULL,
  "webhook_url" text NULL,
  "channels" jsonb NOT NULL DEFAULT '["email"]',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "escalation_step_targets_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "escalation_step_targets_escalation_step_id_organization_id_fkey" FOREIGN KEY ("escalation_step_id", "organization_id") REFERENCES "escalation_steps" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_schedule_id_organization_id_fkey" FOREIGN KEY ("schedule_id", "organization_id") REFERENCES "schedules" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_target_type_check" CHECK (target_type = ANY (ARRAY['user'::text, 'rotation'::text, 'webhook'::text]))
);
-- Create index "escalation_step_targets_escalation_step_id_idx" to table: "escalation_step_targets"
CREATE INDEX "escalation_step_targets_escalation_step_id_idx" ON "escalation_step_targets" ("escalation_step_id");
-- Create index "escalation_step_targets_organization_id_idx" to table: "escalation_step_targets"
CREATE INDEX "escalation_step_targets_organization_id_idx" ON "escalation_step_targets" ("organization_id");
