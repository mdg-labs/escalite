-- Modify "services" table
ALTER TABLE "services" ADD COLUMN "dedup_window_seconds" integer NOT NULL DEFAULT 300;
