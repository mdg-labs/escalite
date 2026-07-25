// Postgres NOTIFY triggers for realtime GraphQL fan-out (#96).
// See also schema/sql/realtime_notify.sql (sqlc/export mirror).

function "notify_alert_change" {
  schema = schema.public
  lang   = PLpgSQL
  return = trigger
  as = <<-SQL
  BEGIN
    PERFORM pg_notify(
      'escalite_alerts',
      json_build_object(
        'alert_id', NEW.id,
        'organization_id', NEW.organization_id,
        'status', NEW.status,
        'op', TG_OP
      )::text
    );
    RETURN NEW;
  END;
  SQL
}

trigger "alerts_notify_insert" {
  on = table.alerts
  after {
    insert = true
  }
  foreach = ROW
  execute {
    function = function.notify_alert_change
  }
}

trigger "alerts_notify_status_update" {
  on = table.alerts
  after {
    update_of = [table.alerts.column.status]
  }
  foreach = ROW
  execute {
    function = function.notify_alert_change
  }
}

function "notify_schedule_row_change" {
  schema = schema.public
  lang   = PLpgSQL
  return = trigger
  as = <<-SQL
  DECLARE
    v_schedule_id uuid;
    v_organization_id uuid;
  BEGIN
    IF TG_TABLE_NAME = 'schedules' THEN
      IF TG_OP = 'DELETE' THEN
        v_schedule_id := OLD.id;
        v_organization_id := OLD.organization_id;
      ELSE
        v_schedule_id := NEW.id;
        v_organization_id := NEW.organization_id;
      END IF;
    ELSIF TG_TABLE_NAME = 'rotations' THEN
      IF TG_OP = 'DELETE' THEN
        v_schedule_id := OLD.schedule_id;
        v_organization_id := OLD.organization_id;
      ELSE
        v_schedule_id := NEW.schedule_id;
        v_organization_id := NEW.organization_id;
      END IF;
    ELSIF TG_TABLE_NAME = 'overrides' THEN
      IF TG_OP = 'DELETE' THEN
        v_schedule_id := OLD.schedule_id;
        v_organization_id := OLD.organization_id;
      ELSE
        v_schedule_id := NEW.schedule_id;
        v_organization_id := NEW.organization_id;
      END IF;
    END IF;

    PERFORM pg_notify(
      'escalite_schedules',
      json_build_object(
        'schedule_id', v_schedule_id,
        'organization_id', v_organization_id,
        'table', TG_TABLE_NAME,
        'op', TG_OP
      )::text
    );

    IF TG_OP = 'DELETE' THEN
      RETURN OLD;
    END IF;
    RETURN NEW;
  END;
  SQL
}

trigger "schedules_notify_change" {
  on = table.schedules
  after {
    insert = true
    update = true
    delete = true
  }
  foreach = ROW
  execute {
    function = function.notify_schedule_row_change
  }
}

trigger "rotations_notify_change" {
  on = table.rotations
  after {
    insert = true
    update = true
    delete = true
  }
  foreach = ROW
  execute {
    function = function.notify_schedule_row_change
  }
}

trigger "overrides_notify_change" {
  on = table.overrides
  after {
    insert = true
    update = true
    delete = true
  }
  foreach = ROW
  execute {
    function = function.notify_schedule_row_change
  }
}
