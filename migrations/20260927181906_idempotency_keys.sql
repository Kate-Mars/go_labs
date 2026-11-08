-- +goose Up
CREATE TABLE idempotency_keys (
                                  key          UUID PRIMARY KEY,
                                  trip_id      UUID REFERENCES trips(id) ON DELETE CASCADE,
                                  request_hash BYTEA NOT NULL,
                                  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idempotency_keys_created_at_idx ON idempotency_keys (created_at);

-- +goose Down
DROP TABLE idempotency_keys;
