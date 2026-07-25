-- Create "organizations" table
CREATE TABLE "organizations" (
  "id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id")
);
-- Create "users" table
CREATE TABLE "users" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "email" text NOT NULL,
  "password_hash" text NULL,
  "role" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "users_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "users_organization_id_email_key" UNIQUE ("organization_id", "email"),
  CONSTRAINT "users_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "users_role_check" CHECK (role = ANY (ARRAY['admin'::text, 'member'::text]))
);
-- Create index "users_organization_id_idx" to table: "users"
CREATE INDEX "users_organization_id_idx" ON "users" ("organization_id");
-- Create "teams" table
CREATE TABLE "teams" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "teams_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "teams_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "teams_organization_id_idx" to table: "teams"
CREATE INDEX "teams_organization_id_idx" ON "teams" ("organization_id");
-- Create "incidents" table
CREATE TABLE "incidents" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "title" text NOT NULL,
  "status" text NOT NULL DEFAULT 'investigating',
  "created_by_user_id" uuid NOT NULL,
  "resolved_at" timestamptz NULL,
  "slack_channel_id" text NULL,
  "slack_thread_ts" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "incidents_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "incidents_created_by_user_id_organization_id_fkey" FOREIGN KEY ("created_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "incidents_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incidents_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incidents_status_check" CHECK (status = ANY (ARRAY['investigating'::text, 'identified'::text, 'monitoring'::text, 'resolved'::text]))
);
-- Create index "incidents_organization_id_idx" to table: "incidents"
CREATE INDEX "incidents_organization_id_idx" ON "incidents" ("organization_id");
-- Create index "incidents_status_idx" to table: "incidents"
CREATE INDEX "incidents_status_idx" ON "incidents" ("status");
-- Create index "incidents_team_id_idx" to table: "incidents"
CREATE INDEX "incidents_team_id_idx" ON "incidents" ("team_id");
-- Create "services" table
CREATE TABLE "services" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "name" text NOT NULL,
  "dedup_window_seconds" integer NOT NULL DEFAULT 300,
  "auto_promote_enabled" boolean NOT NULL DEFAULT false,
  "auto_promote_alert_threshold" integer NOT NULL DEFAULT 3,
  "auto_promote_window_seconds" integer NOT NULL DEFAULT 300,
  "auto_promote_suppress_escalation_priorities" text[] NOT NULL DEFAULT ARRAY['high'::text, 'low'::text],
  "deleted_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "services_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "services_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "services_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "services_active_organization_id_idx" to table: "services"
CREATE INDEX "services_active_organization_id_idx" ON "services" ("organization_id") WHERE (deleted_at IS NULL);
-- Create index "services_organization_id_idx" to table: "services"
CREATE INDEX "services_organization_id_idx" ON "services" ("organization_id");
-- Create index "services_team_id_idx" to table: "services"
CREATE INDEX "services_team_id_idx" ON "services" ("team_id");
-- Create "integration_keys" table
CREATE TABLE "integration_keys" (
  "id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "token" text NOT NULL,
  "prefix" text NOT NULL,
  "plugin_name" text NOT NULL,
  "config" jsonb NOT NULL DEFAULT '{}',
  "revoked_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "integration_keys_token_key" UNIQUE ("token"),
  CONSTRAINT "integration_keys_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "integration_keys_plugin_name_idx" to table: "integration_keys"
CREATE INDEX "integration_keys_plugin_name_idx" ON "integration_keys" ("plugin_name");
-- Create index "integration_keys_service_id_idx" to table: "integration_keys"
CREATE INDEX "integration_keys_service_id_idx" ON "integration_keys" ("service_id");
-- Create "alerts" table
CREATE TABLE "alerts" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "integration_key_id" uuid NULL,
  "status" text NOT NULL,
  "dedup_key" text NOT NULL,
  "summary" text NOT NULL,
  "description" text NULL,
  "priority" text NOT NULL DEFAULT 'high',
  "event_count" integer NOT NULL DEFAULT 1,
  "escalation_state" jsonb NOT NULL DEFAULT '{}',
  "acknowledged_at" timestamptz NULL,
  "acknowledged_by_user_id" uuid NULL,
  "closed_at" timestamptz NULL,
  "resolved_at" timestamptz NULL,
  "resolved_integration" text NULL,
  "incident_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "alerts_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "alerts_acknowledged_by_user_id_organization_id_fkey" FOREIGN KEY ("acknowledged_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "alerts_incident_id_organization_id_fkey" FOREIGN KEY ("incident_id", "organization_id") REFERENCES "incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "alerts_integration_key_id_fkey" FOREIGN KEY ("integration_key_id") REFERENCES "integration_keys" ("id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "alerts_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "alerts_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "alerts_priority_check" CHECK (priority = ANY (ARRAY['low'::text, 'high'::text])),
  CONSTRAINT "alerts_status_check" CHECK (status = ANY (ARRAY['triggered'::text, 'acknowledged'::text, 'closed'::text]))
);
-- Create index "alerts_incident_id_idx" to table: "alerts"
CREATE INDEX "alerts_incident_id_idx" ON "alerts" ("incident_id");
-- Create index "alerts_organization_id_idx" to table: "alerts"
CREATE INDEX "alerts_organization_id_idx" ON "alerts" ("organization_id");
-- Create index "alerts_service_id_dedup_key_idx" to table: "alerts"
CREATE INDEX "alerts_service_id_dedup_key_idx" ON "alerts" ("service_id", "dedup_key");
-- Create index "alerts_service_id_idx" to table: "alerts"
CREATE INDEX "alerts_service_id_idx" ON "alerts" ("service_id");
-- Create index "alerts_status_idx" to table: "alerts"
CREATE INDEX "alerts_status_idx" ON "alerts" ("status");
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
-- Create "escalation_policies" table
CREATE TABLE "escalation_policies" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "name" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "escalation_policies_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "escalation_policies_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_policies_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "escalation_policies_organization_id_idx" to table: "escalation_policies"
CREATE INDEX "escalation_policies_organization_id_idx" ON "escalation_policies" ("organization_id");
-- Create index "escalation_policies_service_id_idx" to table: "escalation_policies"
CREATE INDEX "escalation_policies_service_id_idx" ON "escalation_policies" ("service_id");
-- Create "escalation_steps" table
CREATE TABLE "escalation_steps" (
  "id" uuid NOT NULL,
  "escalation_policy_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "step_order" integer NOT NULL,
  "delay_minutes" integer NOT NULL DEFAULT 0,
  "repeat_last_step" boolean NOT NULL DEFAULT false,
  "max_repeats" integer NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "escalation_steps_escalation_policy_id_step_order_key" UNIQUE ("escalation_policy_id", "step_order"),
  CONSTRAINT "escalation_steps_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "escalation_steps_escalation_policy_id_organization_id_fkey" FOREIGN KEY ("escalation_policy_id", "organization_id") REFERENCES "escalation_policies" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "escalation_steps_escalation_policy_id_idx" to table: "escalation_steps"
CREATE INDEX "escalation_steps_escalation_policy_id_idx" ON "escalation_steps" ("escalation_policy_id");
-- Create "schedules" table
CREATE TABLE "schedules" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "name" text NOT NULL,
  "timezone" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "schedules_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "schedules_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "schedules_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "schedules_timezone_check" CHECK (timezone(timezone, '2000-01-01 00:00:00+00'::timestamp with time zone) IS NOT NULL)
);
-- Create index "schedules_organization_id_idx" to table: "schedules"
CREATE INDEX "schedules_organization_id_idx" ON "schedules" ("organization_id");
-- Create index "schedules_team_id_idx" to table: "schedules"
CREATE INDEX "schedules_team_id_idx" ON "schedules" ("team_id");
-- Create "escalation_step_targets" table
CREATE TABLE "escalation_step_targets" (
  "id" uuid NOT NULL,
  "escalation_step_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "target_type" text NOT NULL,
  "user_id" uuid NULL,
  "schedule_id" uuid NULL,
  "webhook_url" text NULL,
  "channels" jsonb NOT NULL DEFAULT '["email"]',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "escalation_step_targets_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "escalation_step_targets_escalation_step_id_organization_id_fkey" FOREIGN KEY ("escalation_step_id", "organization_id") REFERENCES "escalation_steps" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_schedule_id_organization_id_fkey" FOREIGN KEY ("schedule_id", "organization_id") REFERENCES "schedules" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "escalation_step_targets_target_type_check" CHECK (target_type = ANY (ARRAY['user'::text, 'rotation'::text, 'webhook'::text]))
);
-- Create index "escalation_step_targets_escalation_step_id_idx" to table: "escalation_step_targets"
CREATE INDEX "escalation_step_targets_escalation_step_id_idx" ON "escalation_step_targets" ("escalation_step_id");
-- Create index "escalation_step_targets_organization_id_idx" to table: "escalation_step_targets"
CREATE INDEX "escalation_step_targets_organization_id_idx" ON "escalation_step_targets" ("organization_id");
-- Create "heartbeat_monitors" table
CREATE TABLE "heartbeat_monitors" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "service_id" uuid NOT NULL,
  "name" text NOT NULL,
  "interval_seconds" integer NOT NULL,
  "grace_seconds" integer NOT NULL,
  "token_hash" text NOT NULL,
  "prefix" text NOT NULL,
  "status" text NOT NULL DEFAULT 'healthy',
  "last_ping_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "heartbeat_monitors_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "heartbeat_monitors_token_hash_key" UNIQUE ("token_hash"),
  CONSTRAINT "heartbeat_monitors_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "heartbeat_monitors_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "heartbeat_monitors_grace_seconds_check" CHECK (grace_seconds > 0),
  CONSTRAINT "heartbeat_monitors_interval_seconds_check" CHECK (interval_seconds > 0),
  CONSTRAINT "heartbeat_monitors_status_check" CHECK (status = ANY (ARRAY['healthy'::text, 'overdue'::text, 'triggered'::text]))
);
-- Create index "heartbeat_monitors_organization_id_idx" to table: "heartbeat_monitors"
CREATE INDEX "heartbeat_monitors_organization_id_idx" ON "heartbeat_monitors" ("organization_id");
-- Create index "heartbeat_monitors_service_id_idx" to table: "heartbeat_monitors"
CREATE INDEX "heartbeat_monitors_service_id_idx" ON "heartbeat_monitors" ("service_id");
-- Create "incident_role_definitions" table
CREATE TABLE "incident_role_definitions" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" text NOT NULL,
  "sort_order" integer NOT NULL DEFAULT 0,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "incident_role_definitions_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "incident_role_definitions_organization_id_name_key" UNIQUE ("organization_id", "name"),
  CONSTRAINT "incident_role_definitions_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "incident_role_definitions_organization_id_idx" to table: "incident_role_definitions"
CREATE INDEX "incident_role_definitions_organization_id_idx" ON "incident_role_definitions" ("organization_id");
-- Create "incident_role_assignments" table
CREATE TABLE "incident_role_assignments" (
  "id" uuid NOT NULL,
  "incident_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "role_definition_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "assigned_by_user_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "incident_role_assignments_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "incident_role_assignments_incident_id_role_definition_id_key" UNIQUE ("incident_id", "role_definition_id"),
  CONSTRAINT "incident_role_assignments_assigned_by_user_id_organization_id_f" FOREIGN KEY ("assigned_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "incident_role_assignments_incident_id_organization_id_fkey" FOREIGN KEY ("incident_id", "organization_id") REFERENCES "incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incident_role_assignments_role_definition_id_organization_id_fk" FOREIGN KEY ("role_definition_id", "organization_id") REFERENCES "incident_role_definitions" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "incident_role_assignments_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "incident_role_assignments_incident_id_idx" to table: "incident_role_assignments"
CREATE INDEX "incident_role_assignments_incident_id_idx" ON "incident_role_assignments" ("incident_id");
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
-- Create "mobile_auth_codes" table
CREATE TABLE "mobile_auth_codes" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "code_hash" text NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "mobile_auth_codes_code_hash_key" UNIQUE ("code_hash"),
  CONSTRAINT "mobile_auth_codes_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "mobile_auth_codes_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "mobile_auth_codes_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "mobile_auth_codes_expires_at_idx" to table: "mobile_auth_codes"
CREATE INDEX "mobile_auth_codes_expires_at_idx" ON "mobile_auth_codes" ("expires_at");
-- Create index "mobile_auth_codes_user_id_idx" to table: "mobile_auth_codes"
CREATE INDEX "mobile_auth_codes_user_id_idx" ON "mobile_auth_codes" ("user_id");
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
-- Create "notification_attempts" table
CREATE TABLE "notification_attempts" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "alert_id" uuid NOT NULL,
  "escalation_step_id" uuid NULL,
  "channel" text NOT NULL,
  "status" text NOT NULL,
  "recipient" jsonb NOT NULL DEFAULT '{}',
  "error_message" text NULL,
  "sent_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "notification_attempts_alert_id_organization_id_fkey" FOREIGN KEY ("alert_id", "organization_id") REFERENCES "alerts" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "notification_attempts_escalation_step_id_organization_id_fkey" FOREIGN KEY ("escalation_step_id", "organization_id") REFERENCES "escalation_steps" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "notification_attempts_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "notification_attempts_status_check" CHECK (status = ANY (ARRAY['pending'::text, 'sent'::text, 'failed'::text]))
);
-- Create index "notification_attempts_alert_id_idx" to table: "notification_attempts"
CREATE INDEX "notification_attempts_alert_id_idx" ON "notification_attempts" ("alert_id");
-- Create index "notification_attempts_organization_id_idx" to table: "notification_attempts"
CREATE INDEX "notification_attempts_organization_id_idx" ON "notification_attempts" ("organization_id");
-- Create "organization_slack_settings" table
CREATE TABLE "organization_slack_settings" (
  "organization_id" uuid NOT NULL,
  "bot_token_ciphertext" bytea NOT NULL,
  "encryption_key_id" text NOT NULL,
  "token_hint" text NOT NULL,
  "workspace_id" text NULL,
  "workspace_name" text NULL,
  "bot_user_id" text NULL,
  "scope" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("organization_id"),
  CONSTRAINT "organization_slack_settings_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create "rotations" table
CREATE TABLE "rotations" (
  "id" uuid NOT NULL,
  "schedule_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" text NOT NULL,
  "layer" integer NOT NULL,
  "rrule" text NOT NULL,
  "participants" jsonb NOT NULL DEFAULT '[]',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "rotations_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "rotations_schedule_id_layer_key" UNIQUE ("schedule_id", "layer"),
  CONSTRAINT "rotations_schedule_id_organization_id_fkey" FOREIGN KEY ("schedule_id", "organization_id") REFERENCES "schedules" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "rotations_organization_id_idx" to table: "rotations"
CREATE INDEX "rotations_organization_id_idx" ON "rotations" ("organization_id");
-- Create index "rotations_schedule_id_idx" to table: "rotations"
CREATE INDEX "rotations_schedule_id_idx" ON "rotations" ("schedule_id");
-- Create "overrides" table
CREATE TABLE "overrides" (
  "id" uuid NOT NULL,
  "schedule_id" uuid NOT NULL,
  "rotation_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "replaced_user_id" uuid NULL,
  "starts_at" timestamptz NOT NULL,
  "ends_at" timestamptz NOT NULL,
  "created_by_user_id" uuid NOT NULL,
  "approved_by_user_id" uuid NULL,
  "deleted_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "overrides_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "overrides_approved_by_user_id_organization_id_fkey" FOREIGN KEY ("approved_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "overrides_created_by_user_id_organization_id_fkey" FOREIGN KEY ("created_by_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE RESTRICT,
  CONSTRAINT "overrides_replaced_user_id_organization_id_fkey" FOREIGN KEY ("replaced_user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "overrides_rotation_id_organization_id_fkey" FOREIGN KEY ("rotation_id", "organization_id") REFERENCES "rotations" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "overrides_schedule_id_organization_id_fkey" FOREIGN KEY ("schedule_id", "organization_id") REFERENCES "schedules" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "overrides_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "overrides_window_check" CHECK (ends_at > starts_at)
);
-- Create index "overrides_active_idx" to table: "overrides"
CREATE INDEX "overrides_active_idx" ON "overrides" ("schedule_id", "starts_at", "ends_at") WHERE (deleted_at IS NULL);
-- Create index "overrides_rotation_id_idx" to table: "overrides"
CREATE INDEX "overrides_rotation_id_idx" ON "overrides" ("rotation_id");
-- Create index "overrides_schedule_id_idx" to table: "overrides"
CREATE INDEX "overrides_schedule_id_idx" ON "overrides" ("schedule_id");
-- Create index "overrides_schedule_id_starts_at_ends_at_idx" to table: "overrides"
CREATE INDEX "overrides_schedule_id_starts_at_ends_at_idx" ON "overrides" ("schedule_id", "starts_at", "ends_at");
-- Create "password_reset_tokens" table
CREATE TABLE "password_reset_tokens" (
  "id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "token_hash" text NOT NULL,
  "expires_at" timestamptz NOT NULL,
  "used_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "password_reset_tokens_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "password_reset_tokens_token_hash_key" UNIQUE ("token_hash"),
  CONSTRAINT "password_reset_tokens_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "password_reset_tokens_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "password_reset_tokens_expires_at_idx" to table: "password_reset_tokens"
CREATE INDEX "password_reset_tokens_expires_at_idx" ON "password_reset_tokens" ("expires_at");
-- Create index "password_reset_tokens_user_id_idx" to table: "password_reset_tokens"
CREATE INDEX "password_reset_tokens_user_id_idx" ON "password_reset_tokens" ("user_id");
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
-- Create "team_memberships" table
CREATE TABLE "team_memberships" (
  "id" uuid NOT NULL,
  "team_id" uuid NOT NULL,
  "user_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "team_memberships_team_id_user_id_key" UNIQUE ("team_id", "user_id"),
  CONSTRAINT "team_memberships_team_id_organization_id_fkey" FOREIGN KEY ("team_id", "organization_id") REFERENCES "teams" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "team_memberships_user_id_organization_id_fkey" FOREIGN KEY ("user_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "team_memberships_team_id_idx" to table: "team_memberships"
CREATE INDEX "team_memberships_team_id_idx" ON "team_memberships" ("team_id");
-- Create index "team_memberships_user_id_idx" to table: "team_memberships"
CREATE INDEX "team_memberships_user_id_idx" ON "team_memberships" ("user_id");
-- Create "timeline_events" table
CREATE TABLE "timeline_events" (
  "id" uuid NOT NULL,
  "incident_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "actor_id" uuid NULL,
  "event_type" text NOT NULL,
  "body" text NOT NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}',
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "timeline_events_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "timeline_events_actor_id_organization_id_fkey" FOREIGN KEY ("actor_id", "organization_id") REFERENCES "users" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "timeline_events_incident_id_organization_id_fkey" FOREIGN KEY ("incident_id", "organization_id") REFERENCES "incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "timeline_events_event_type_check" CHECK (event_type = ANY (ARRAY['declared'::text, 'status_changed'::text, 'note'::text, 'role_assigned'::text, 'role_unassigned'::text]))
);
-- Create index "timeline_events_incident_id_created_at_idx" to table: "timeline_events"
CREATE INDEX "timeline_events_incident_id_created_at_idx" ON "timeline_events" ("incident_id", "created_at");
-- Create index "timeline_events_incident_id_idx" to table: "timeline_events"
CREATE INDEX "timeline_events_incident_id_idx" ON "timeline_events" ("incident_id");
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
