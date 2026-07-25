-- Modify "services" table
ALTER TABLE "services" ADD COLUMN "auto_promote_enabled" boolean NOT NULL DEFAULT false, ADD COLUMN "auto_promote_alert_threshold" integer NOT NULL DEFAULT 3, ADD COLUMN "auto_promote_window_seconds" integer NOT NULL DEFAULT 300, ADD COLUMN "auto_promote_suppress_escalation_priorities" text[] NOT NULL DEFAULT ARRAY['high'::text, 'low'::text];
