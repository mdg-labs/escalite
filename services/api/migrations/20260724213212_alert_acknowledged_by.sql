-- Modify "alerts" table
ALTER TABLE "alerts" ADD COLUMN "acknowledged_by_user_id" uuid NULL, ADD CONSTRAINT "alerts_acknowledged_by_user_id_organization_id_fkey" FOREIGN KEY ("acknowledged_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL;
