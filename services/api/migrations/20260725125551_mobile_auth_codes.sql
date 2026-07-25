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
