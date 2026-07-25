-- Modify "organization_slack_settings" table
ALTER TABLE "organization_slack_settings" ADD COLUMN "workspace_id" text NULL, ADD COLUMN "workspace_name" text NULL, ADD COLUMN "bot_user_id" text NULL, ADD COLUMN "scope" text NULL;
