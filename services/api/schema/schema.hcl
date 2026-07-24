// Canonical Escalite database schema (Atlas HCL).
// Core domain entities: organizations, teams, users, team_memberships (#39).
// Service + integration key tables (#40).

schema "public" {
}

table "organizations" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "name" {
    null = false
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }
}

table "teams" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "name" {
    null = false
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "teams_organization_id_fkey" {
    columns     = [column.organization_id]
    ref_columns = [table.organizations.column.id]
    on_delete   = CASCADE
  }

  unique "teams_id_organization_id_key" {
    columns = [column.id, column.organization_id]
  }

  index "teams_organization_id_idx" {
    columns = [column.organization_id]
  }
}

table "users" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "email" {
    null = false
    type = text
  }
  column "password_hash" {
    null = true
    type = text
  }
  column "role" {
    null = false
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "users_organization_id_fkey" {
    columns     = [column.organization_id]
    ref_columns = [table.organizations.column.id]
    on_delete   = CASCADE
  }

  unique "users_organization_id_email_key" {
    columns = [column.organization_id, column.email]
  }

  unique "users_id_organization_id_key" {
    columns = [column.id, column.organization_id]
  }

  index "users_organization_id_idx" {
    columns = [column.organization_id]
  }

  check "users_role_check" {
    expr = "(role = ANY (ARRAY['admin'::text, 'member'::text]))"
  }
}

table "team_memberships" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "team_id" {
    null = false
    type = uuid
  }
  column "user_id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "team_memberships_team_id_organization_id_fkey" {
    columns     = [column.team_id, column.organization_id]
    ref_columns = [table.teams.column.id, table.teams.column.organization_id]
    on_delete   = CASCADE
  }

  foreign_key "team_memberships_user_id_organization_id_fkey" {
    columns     = [column.user_id, column.organization_id]
    ref_columns = [table.users.column.id, table.users.column.organization_id]
    on_delete   = CASCADE
  }

  unique "team_memberships_team_id_user_id_key" {
    columns = [column.team_id, column.user_id]
  }

  index "team_memberships_team_id_idx" {
    columns = [column.team_id]
  }

  index "team_memberships_user_id_idx" {
    columns = [column.user_id]
  }
}

table "services" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "team_id" {
    null = false
    type = uuid
  }
  column "name" {
    null = false
    type = text
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "services_organization_id_fkey" {
    columns     = [column.organization_id]
    ref_columns = [table.organizations.column.id]
    on_delete   = CASCADE
  }

  foreign_key "services_team_id_organization_id_fkey" {
    columns     = [column.team_id, column.organization_id]
    ref_columns = [table.teams.column.id, table.teams.column.organization_id]
    on_delete   = CASCADE
  }

  unique "services_id_organization_id_key" {
    columns = [column.id, column.organization_id]
  }

  index "services_organization_id_idx" {
    columns = [column.organization_id]
  }

  index "services_team_id_idx" {
    columns = [column.team_id]
  }
}

table "integration_keys" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "service_id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "token" {
    null = false
    type = text
  }
  column "prefix" {
    null = false
    type = text
  }
  column "plugin_name" {
    null = false
    type = text
  }
  column "config" {
    null    = false
    type    = jsonb
    default = sql("'{}'::jsonb")
  }
  column "revoked_at" {
    null = true
    type = timestamptz
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }
  column "updated_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "integration_keys_service_id_organization_id_fkey" {
    columns     = [column.service_id, column.organization_id]
    ref_columns = [table.services.column.id, table.services.column.organization_id]
    on_delete   = CASCADE
  }

  unique "integration_keys_token_key" {
    columns = [column.token]
  }

  index "integration_keys_service_id_idx" {
    columns = [column.service_id]
  }

  index "integration_keys_plugin_name_idx" {
    columns = [column.plugin_name]
  }
}
