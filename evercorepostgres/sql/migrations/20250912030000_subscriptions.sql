-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS subscriptions (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(128) NOT NULL UNIQUE,
  aggregate_type_id BIGINT NULL REFERENCES aggregate_types(id),
  event_type_id BIGINT NULL REFERENCES event_types(id),
  aggregate_key VARCHAR(64) NULL,
  start_from VARCHAR(16) NOT NULL DEFAULT 'beginning',
  start_event_id BIGINT NOT NULL DEFAULT 0,
  start_timestamp TIMESTAMP NULL,
  last_event_id BIGINT NOT NULL DEFAULT 0,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  lease_owner VARCHAR(128) NULL,
  lease_expires_at TIMESTAMP NULL,
  created_at TIMESTAMP NOT NULL DEFAULT now(),
  updated_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS subscription_event_types (
  subscription_id BIGINT NOT NULL REFERENCES subscriptions(id) ON DELETE CASCADE,
  event_type_id BIGINT NOT NULL REFERENCES event_types(id),
  PRIMARY KEY (subscription_id, event_type_id)
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_active ON subscriptions(active);
CREATE INDEX IF NOT EXISTS idx_subscriptions_lease_exp ON subscriptions(lease_expires_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS subscription_event_types;
DROP TABLE IF EXISTS subscriptions;
-- +goose StatementEnd

