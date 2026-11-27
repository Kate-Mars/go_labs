-- +goose Up
-- +goose NO TRANSACTION
CREATE INDEX CONCURRENTLY trips_stale_active_idx
    ON trips (COALESCE(last_position_at, started_at))
    WHERE status = 'active';

-- +goose Down
-- +goose NO TRANSACTION
DROP INDEX CONCURRENTLY trips_stale_active_idx;