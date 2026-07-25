-- Modify "services" table
ALTER TABLE "services" ADD COLUMN "deleted_at" timestamptz NULL;
-- Create index "services_active_organization_id_idx" to table: "services"
CREATE INDEX "services_active_organization_id_idx" ON "services" ("organization_id") WHERE (deleted_at IS NULL);
