-- Modify "alerts" table
ALTER TABLE "alerts" ADD COLUMN "resolved_at" timestamptz NULL, ADD COLUMN "resolved_integration" text NULL;
