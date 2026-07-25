-- Create "maintenance_windows" table
CREATE TABLE "maintenance_windows" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "description" text NOT NULL,
  "starts_at" timestamptz NOT NULL,
  "ends_at" timestamptz NOT NULL,
  "suppress_notifications" boolean NOT NULL DEFAULT true,
  "suppress_ingestion" boolean NOT NULL DEFAULT false,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "maintenance_windows_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "maintenance_windows_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "maintenance_windows_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "maintenance_windows_window_check" CHECK (ends_at > starts_at)
);
-- Create index "maintenance_windows_organization_id_idx" to table: "maintenance_windows"
CREATE INDEX "maintenance_windows_organization_id_idx" ON "maintenance_windows" ("organization_id");
-- Create index "maintenance_windows_service_id_active_idx" to table: "maintenance_windows"
CREATE INDEX "maintenance_windows_service_id_active_idx" ON "maintenance_windows" ("service_id", "starts_at", "ends_at");
-- Create index "maintenance_windows_service_id_idx" to table: "maintenance_windows"
CREATE INDEX "maintenance_windows_service_id_idx" ON "maintenance_windows" ("service_id");
