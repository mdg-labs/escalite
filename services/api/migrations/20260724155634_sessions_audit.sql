-- Create "audit_events" table
CREATE TABLE "audit_events" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "actor_id" uuid NULL,
  "action" text NOT NULL,
  "target_type" text NULL,
  "target_id" uuid NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "audit_events_actor_id_organization_id_fkey" FOREIGN KEY ("actor_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "audit_events_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "audit_events_actor_id_idx" to table: "audit_events"
CREATE INDEX "audit_events_actor_id_idx" ON "audit_events" ("actor_id");
-- Create index "audit_events_created_at_idx" to table: "audit_events"
CREATE INDEX "audit_events_created_at_idx" ON "audit_events" ("created_at");
-- Create index "audit_events_organization_id_idx" to table: "audit_events"
CREATE INDEX "audit_events_organization_id_idx" ON "audit_events" ("organization_id");
-- Create index "audit_events_target_type_target_id_idx" to table: "audit_events"
CREATE INDEX "audit_events_target_type_target_id_idx" ON "audit_events" ("target_type", "target_id");
-- Create "refresh_tokens" table
CREATE TABLE "refresh_tokens" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "token_hash" text NOT NULL,
  "user_agent" text NULL,
  "revoked_at" timestamptz NULL,
  "expires_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "refresh_tokens_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "refresh_tokens_token_hash_key" UNIQUE ("token_hash"),
  CONSTRAINT "refresh_tokens_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "refresh_tokens_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "refresh_tokens_organization_id_idx" to table: "refresh_tokens"
CREATE INDEX "refresh_tokens_organization_id_idx" ON "refresh_tokens" ("organization_id");
-- Create index "refresh_tokens_user_id_idx" to table: "refresh_tokens"
CREATE INDEX "refresh_tokens_user_id_idx" ON "refresh_tokens" ("user_id");
-- Create "sessions" table
CREATE TABLE "sessions" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "user_agent" text NULL,
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "sessions_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "sessions_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "sessions_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "sessions_expires_at_idx" to table: "sessions"
CREATE INDEX "sessions_expires_at_idx" ON "sessions" ("expires_at");
-- Create index "sessions_organization_id_idx" to table: "sessions"
CREATE INDEX "sessions_organization_id_idx" ON "sessions" ("organization_id");
-- Create index "sessions_user_id_idx" to table: "sessions"
CREATE INDEX "sessions_user_id_idx" ON "sessions" ("user_id");
