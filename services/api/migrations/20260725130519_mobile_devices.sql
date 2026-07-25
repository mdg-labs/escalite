-- Create "mobile_devices" table
CREATE TABLE "mobile_devices" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "refresh_token_id" uuid NULL,
  "expo_push_token" text NOT NULL,
  "push_token_prefix" text NOT NULL,
  "platform" text NULL,
  "device_label" text NULL,
  "revoked_at" timestamptz NULL,
  "last_registered_at" timestamptz NOT NULL DEFAULT now(),
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "mobile_devices_expo_push_token_key" UNIQUE ("expo_push_token"),
  CONSTRAINT "mobile_devices_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "mobile_devices_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "mobile_devices_refresh_token_id_organization_id_fkey" FOREIGN KEY ("refresh_token_id", "organization_id") REFERENCES "refresh_tokens" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "mobile_devices_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "mobile_devices_organization_id_idx" to table: "mobile_devices"
CREATE INDEX "mobile_devices_organization_id_idx" ON "mobile_devices" ("organization_id");
-- Create index "mobile_devices_user_id_idx" to table: "mobile_devices"
CREATE INDEX "mobile_devices_user_id_idx" ON "mobile_devices" ("user_id");
