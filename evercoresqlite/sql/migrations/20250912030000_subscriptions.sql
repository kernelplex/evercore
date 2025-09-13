-- +goose Up
-- +goose StatementBegin

CREATE TABLE IF NOT EXISTS subscriptions (
  id INTEGER NOT NULL PRIMARY KEY,
  name VARCHAR(128) NOT NULL UNIQUE,
  aggregate_type_id INTEGER,
  event_type_id INTEGER,
  aggregate_key VARCHAR(64),
  start_from VARCHAR(16) NOT NULL DEFAULT 'beginning',
  start_event_id INTEGER NOT NULL DEFAULT 0,
  start_timestamp DATETIME,
  last_event_id INTEGER NOT NULL DEFAULT 0,
  active BOOLEAN NOT NULL DEFAULT 1,
  lease_owner VARCHAR(128),
  lease_expires_at DATETIME,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS subscription_event_types (
  subscription_id INTEGER NOT NULL,
  event_type_id INTEGER NOT NULL,
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
