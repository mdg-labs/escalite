// Canonical Escalite database schema (Atlas HCL).
// Core domain entities: organizations, teams, users, team_memberships (#39).
// Service + integration key tables (#40).
// Escalation policies, alerts, notification_attempts (#42).

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

table "escalation_policies" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "service_id" {
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

  foreign_key "escalation_policies_organization_id_fkey" {
    columns     = [column.organization_id]
    ref_columns = [table.organizations.column.id]
    on_delete   = CASCADE
  }

  foreign_key "escalation_policies_service_id_organization_id_fkey" {
    columns     = [column.service_id, column.organization_id]
    ref_columns = [table.services.column.id, table.services.column.organization_id]
    on_delete   = CASCADE
  }

  unique "escalation_policies_id_organization_id_key" {
    columns = [column.id, column.organization_id]
  }

  index "escalation_policies_organization_id_idx" {
    columns = [column.organization_id]
  }

  index "escalation_policies_service_id_idx" {
    columns = [column.service_id]
  }
}

table "escalation_steps" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "escalation_policy_id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "step_order" {
    null = false
    type = integer
  }
  column "delay_minutes" {
    null    = false
    type    = integer
    default = 0
  }
  column "repeat_last_step" {
    null    = false
    type    = boolean
    default = false
  }
  column "max_repeats" {
    null = true
    type = integer
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

  foreign_key "escalation_steps_escalation_policy_id_organization_id_fkey" {
    columns     = [column.escalation_policy_id, column.organization_id]
    ref_columns = [table.escalation_policies.column.id, table.escalation_policies.column.organization_id]
    on_delete   = CASCADE
  }

  unique "escalation_steps_escalation_policy_id_step_order_key" {
    columns = [column.escalation_policy_id, column.step_order]
  }

  unique "escalation_steps_id_organization_id_key" {
    columns = [column.id, column.organization_id]
  }

  index "escalation_steps_escalation_policy_id_idx" {
    columns = [column.escalation_policy_id]
  }
}

table "alerts" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "service_id" {
    null = false
    type = uuid
  }
  column "integration_key_id" {
    null = true
    type = uuid
  }
  column "status" {
    null = false
    type = text
  }
  column "dedup_key" {
    null = false
    type = text
  }
  column "summary" {
    null = false
    type = text
  }
  column "description" {
    null = true
    type = text
  }
  column "priority" {
    null    = false
    type    = text
    default = "high"
  }
  column "event_count" {
    null    = false
    type    = integer
    default = 1
  }
  column "escalation_state" {
    null    = false
    type    = jsonb
    default = sql("'{}'::jsonb")
  }
  column "acknowledged_at" {
    null = true
    type = timestamptz
  }
  column "closed_at" {
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

  foreign_key "alerts_organization_id_fkey" {
    columns     = [column.organization_id]
    ref_columns = [table.organizations.column.id]
    on_delete   = CASCADE
  }

  foreign_key "alerts_service_id_organization_id_fkey" {
    columns     = [column.service_id, column.organization_id]
    ref_columns = [table.services.column.id, table.services.column.organization_id]
    on_delete   = CASCADE
  }

  foreign_key "alerts_integration_key_id_fkey" {
    columns     = [column.integration_key_id]
    ref_columns = [table.integration_keys.column.id]
    on_delete   = SET_NULL
  }

  unique "alerts_id_organization_id_key" {
    columns = [column.id, column.organization_id]
  }

  index "alerts_organization_id_idx" {
    columns = [column.organization_id]
  }

  index "alerts_service_id_idx" {
    columns = [column.service_id]
  }

  index "alerts_service_id_dedup_key_idx" {
    columns = [column.service_id, column.dedup_key]
  }

  index "alerts_status_idx" {
    columns = [column.status]
  }

  check "alerts_status_check" {
    expr = "(status = ANY (ARRAY['triggered'::text, 'acknowledged'::text, 'closed'::text]))"
  }

  check "alerts_priority_check" {
    expr = "(priority = ANY (ARRAY['low'::text, 'high'::text]))"
  }
}

table "notification_attempts" {
  schema = schema.public

  column "id" {
    null = false
    type = uuid
  }
  column "organization_id" {
    null = false
    type = uuid
  }
  column "alert_id" {
    null = false
    type = uuid
  }
  column "escalation_step_id" {
    null = true
    type = uuid
  }
  column "channel" {
    null = false
    type = text
  }
  column "status" {
    null = false
    type = text
  }
  column "recipient" {
    null    = false
    type    = jsonb
    default = sql("'{}'::jsonb")
  }
  column "error_message" {
    null = true
    type = text
  }
  column "sent_at" {
    null = true
    type = timestamptz
  }
  column "created_at" {
    null    = false
    type    = timestamptz
    default = sql("now()")
  }

  primary_key {
    columns = [column.id]
  }

  foreign_key "notification_attempts_organization_id_fkey" {
    columns     = [column.organization_id]
    ref_columns = [table.organizations.column.id]
    on_delete   = CASCADE
  }

  foreign_key "notification_attempts_alert_id_organization_id_fkey" {
    columns     = [column.alert_id, column.organization_id]
    ref_columns = [table.alerts.column.id, table.alerts.column.organization_id]
    on_delete   = CASCADE
  }

  foreign_key "notification_attempts_escalation_step_id_organization_id_fkey" {
    columns     = [column.escalation_step_id, column.organization_id]
    ref_columns = [table.escalation_steps.column.id, table.escalation_steps.column.organization_id]
    on_delete   = SET_NULL
  }

  index "notification_attempts_alert_id_idx" {
    columns = [column.alert_id]
  }

  index "notification_attempts_organization_id_idx" {
    columns = [column.organization_id]
  }

  check "notification_attempts_status_check" {
    expr = "(status = ANY (ARRAY['pending'::text, 'sent'::text, 'failed'::text]))"
  }
}
