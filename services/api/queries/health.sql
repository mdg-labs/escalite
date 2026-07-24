-- name: PingDatabase :one
SELECT 1::integer AS ok;

-- name: CoreSchemaReady :one
SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public'
      AND table_name = 'organizations'
) AS ready;

-- name: SessionsSchemaReady :one
SELECT EXISTS (
    SELECT 1
    FROM information_schema.tables
    WHERE table_schema = 'public'
      AND table_name = 'sessions'
) AS ready;
