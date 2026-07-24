-- +goose Up
-- Bootstrap migration: establishes goose version tracking for Escalite schema v1.
-- Domain tables are added in subsequent migrations (issues #39–#43).
SELECT 1;

-- +goose Down
-- No-op: bootstrap has no schema objects to drop.
SELECT 1;
