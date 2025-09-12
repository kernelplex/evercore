-- +goose Up
-- +goose StatementBegin

CREATE OR REPLACE FUNCTION get_events_for_subscription(
  p_after_id BIGINT,
  p_aggregate_type_id BIGINT,
  p_aggregate_key VARCHAR,
  p_use_multi BOOLEAN,
  p_subscription_id BIGINT,
  p_event_type_id BIGINT,
  p_limit INTEGER
) RETURNS TABLE (
  id BIGINT,
  aggregate_id BIGINT,
  natural_key VARCHAR,
  sequence BIGINT,
  aggregate_type_id BIGINT,
  aggregate_type VARCHAR,
  event_type_id BIGINT,
  event_type VARCHAR,
  event_time TIMESTAMP,
  state TEXT
) AS $$
  SELECT el.id, el.aggregate_id, el.natural_key, el.sequence,
         el.aggregate_type_id, el.aggregate_type,
         el.event_type_id, el.event_type,
         el.event_time, el.state
  FROM event_log el
  WHERE el.id > p_after_id
    AND (p_aggregate_type_id IS NULL OR el.aggregate_type_id = p_aggregate_type_id)
    AND (p_aggregate_key IS NULL OR el.natural_key = p_aggregate_key)
    AND (
      (p_use_multi IS TRUE AND el.event_type_id IN (SELECT event_type_id FROM subscription_event_types WHERE subscription_id = p_subscription_id))
      OR
      (p_use_multi IS FALSE AND (p_event_type_id IS NULL OR el.event_type_id = p_event_type_id))
    )
  ORDER BY el.id ASC
  LIMIT p_limit;
$$ LANGUAGE sql STABLE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP FUNCTION IF EXISTS get_events_for_subscription(BIGINT, BIGINT, VARCHAR, BOOLEAN, BIGINT, BIGINT, INTEGER);
-- +goose StatementEnd

