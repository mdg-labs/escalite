-- Create "status_pages" table
CREATE TABLE "status_pages" (
  "id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "slug" text NOT NULL,
  "title" text NOT NULL,
  "enabled" boolean NOT NULL DEFAULT false,
  "frame_ancestors_csp" text NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "status_pages_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "status_pages_organization_id_key" UNIQUE ("organization_id"),
  CONSTRAINT "status_pages_slug_key" UNIQUE ("slug"),
  CONSTRAINT "status_pages_organization_id_fkey" FOREIGN KEY ("organization_id") REFERENCES "organizations" ("id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "status_pages_organization_id_idx" to table: "status_pages"
CREATE INDEX "status_pages_organization_id_idx" ON "status_pages" ("organization_id");
-- Create "status_page_components" table
CREATE TABLE "status_page_components" (
  "id" uuid NOT NULL,
  "status_page_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "name" text NOT NULL,
  "description" text NULL,
  "status" text NOT NULL DEFAULT 'operational',
  "position" integer NOT NULL DEFAULT 0,
  "service_id" uuid NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "status_page_components_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "status_page_components_service_id_organization_id_fkey" FOREIGN KEY ("service_id", "organization_id") REFERENCES "services" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "status_page_components_status_page_id_organization_id_fkey" FOREIGN KEY ("status_page_id", "organization_id") REFERENCES "status_pages" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "status_page_components_status_check" CHECK (status = ANY (ARRAY['operational'::text, 'degraded'::text, 'partial_outage'::text, 'major_outage'::text]))
);
-- Create index "status_page_components_organization_id_idx" to table: "status_page_components"
CREATE INDEX "status_page_components_organization_id_idx" ON "status_page_components" ("organization_id");
-- Create index "status_page_components_status_page_id_idx" to table: "status_page_components"
CREATE INDEX "status_page_components_status_page_id_idx" ON "status_page_components" ("status_page_id");
-- Create "status_page_incidents" table
CREATE TABLE "status_page_incidents" (
  "id" uuid NOT NULL,
  "status_page_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "incident_id" uuid NULL,
  "title" text NOT NULL,
  "status" text NOT NULL DEFAULT 'investigating',
  "resolved_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  "updated_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "status_page_incidents_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "status_page_incidents_incident_id_organization_id_fkey" FOREIGN KEY ("incident_id", "organization_id") REFERENCES "incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE SET NULL,
  CONSTRAINT "status_page_incidents_status_page_id_organization_id_fkey" FOREIGN KEY ("status_page_id", "organization_id") REFERENCES "status_pages" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "status_page_incidents_status_check" CHECK (status = ANY (ARRAY['investigating'::text, 'identified'::text, 'monitoring'::text, 'resolved'::text]))
);
-- Create index "status_page_incidents_incident_id_idx" to table: "status_page_incidents"
CREATE INDEX "status_page_incidents_incident_id_idx" ON "status_page_incidents" ("incident_id");
-- Create index "status_page_incidents_status_page_id_idx" to table: "status_page_incidents"
CREATE INDEX "status_page_incidents_status_page_id_idx" ON "status_page_incidents" ("status_page_id");
-- Create "status_page_incident_components" table
CREATE TABLE "status_page_incident_components" (
  "status_page_incident_id" uuid NOT NULL,
  "status_page_component_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("status_page_incident_id", "status_page_component_id"),
  CONSTRAINT "status_page_incident_components_status_page_component_id_organi" FOREIGN KEY ("status_page_component_id", "organization_id") REFERENCES "status_page_components" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "status_page_incident_components_status_page_incident_id_organiz" FOREIGN KEY ("status_page_incident_id", "organization_id") REFERENCES "status_page_incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "status_page_incident_components_component_id_idx" to table: "status_page_incident_components"
CREATE INDEX "status_page_incident_components_component_id_idx" ON "status_page_incident_components" ("status_page_component_id");
-- Create "status_page_incident_updates" table
CREATE TABLE "status_page_incident_updates" (
  "id" uuid NOT NULL,
  "status_page_incident_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "body" text NOT NULL,
  "status" text NOT NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "status_page_incident_updates_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "status_page_incident_updates_status_page_incident_id_organizati" FOREIGN KEY ("status_page_incident_id", "organization_id") REFERENCES "status_page_incidents" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE,
  CONSTRAINT "status_page_incident_updates_status_check" CHECK (status = ANY (ARRAY['investigating'::text, 'identified'::text, 'monitoring'::text, 'resolved'::text]))
);
-- Create index "status_page_incident_updates_status_page_incident_id_idx" to table: "status_page_incident_updates"
CREATE INDEX "status_page_incident_updates_status_page_incident_id_idx" ON "status_page_incident_updates" ("status_page_incident_id");
-- Create "status_page_subscriptions" table
CREATE TABLE "status_page_subscriptions" (
  "id" uuid NOT NULL,
  "status_page_id" uuid NOT NULL,
  "organization_id" uuid NOT NULL,
  "email" text NOT NULL,
  "unsubscribed_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY ("id"),
  CONSTRAINT "status_page_subscriptions_id_organization_id_key" UNIQUE ("id", "organization_id"),
  CONSTRAINT "status_page_subscriptions_status_page_id_email_key" UNIQUE ("status_page_id", "email"),
  CONSTRAINT "status_page_subscriptions_status_page_id_organization_id_fkey" FOREIGN KEY ("status_page_id", "organization_id") REFERENCES "status_pages" ("id", "organization_id") ON UPDATE NO ACTION ON DELETE CASCADE
);
-- Create index "status_page_subscriptions_status_page_id_idx" to table: "status_page_subscriptions"
CREATE INDEX "status_page_subscriptions_status_page_id_idx" ON "status_page_subscriptions" ("status_page_id");
