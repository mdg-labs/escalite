-- Realtime NOTIFY triggers for GraphQL fan-out (#96).

CREATE OR REPLACE FUNCTION notify_alert_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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
$$;

CREATE TRIGGER alerts_notify_insert
    AFTER INSERT ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION notify_alert_change();

CREATE TRIGGER alerts_notify_status_update
    AFTER UPDATE OF status ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION notify_alert_change();

CREATE OR REPLACE FUNCTION notify_schedule_row_change() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
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
$$;

CREATE TRIGGER schedules_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON schedules
    FOR EACH ROW
    EXECUTE FUNCTION notify_schedule_row_change();

CREATE TRIGGER rotations_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON rotations
    FOR EACH ROW
    EXECUTE FUNCTION notify_schedule_row_change();

CREATE TRIGGER overrides_notify_change
    AFTER INSERT OR UPDATE OR DELETE ON overrides
    FOR EACH ROW
    EXECUTE FUNCTION notify_schedule_row_change();
