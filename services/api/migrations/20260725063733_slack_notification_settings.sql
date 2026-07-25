-- Create "organization_slack_settings" table
CREATE TABLE "organization_slack_settings" (
  "organization_id" uuid NOT NULL,
  "bot_token_ciphertext" bytea NOT NULL,
  "encryption_key_id" text NOT NULL,
  "token_hint" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("organization_id"),
  CONSTRAINT "organization_slack_settings_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "user_contact_methods" table
CREATE TABLE "user_contact_methods" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "channel" text NOT NULL,
  "config" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "user_contact_methods_organization_id_user_id_channel_key" UNIQUE ("organization_id", "user_id", "channel"),
  CONSTRAINT "user_contact_methods_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "user_contact_methods_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "user_contact_methods_organization_id_idx" to table: "user_contact_methods"
CREATE INDEX "user_contact_methods_organization_id_idx" ON "user_contact_methods" ("organization_id");
-- Create index "user_contact_methods_user_id_idx" to table: "user_contact_methods"
CREATE INDEX "user_contact_methods_user_id_idx" ON "user_contact_methods" ("user_id");
