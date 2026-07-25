-- Create "user_notification_rules" table
CREATE TABLE "user_notification_rules" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "priority" text NOT NULL,
  "steps" jsonb NOT NULL DEFAULT '[]',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "user_notification_rules_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "user_notification_rules_organization_id_user_id_priority_key" UNIQUE ("organization_id", "user_id", "priority"),
  CONSTRAINT "user_notification_rules_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "user_notification_rules_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "user_notification_rules_priority_check" CHECK (priority = ANY (ARRAY['low'::text, 'high'::text]))
);
-- Create index "user_notification_rules_organization_id_idx" to table: "user_notification_rules"
CREATE INDEX "user_notification_rules_organization_id_idx" ON "user_notification_rules" ("organization_id");
-- Create index "user_notification_rules_user_id_idx" to table: "user_notification_rules"
CREATE INDEX "user_notification_rules_user_id_idx" ON "user_notification_rules" ("user_id");
