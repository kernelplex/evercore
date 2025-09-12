
--
-- Aggregate Type Queries
--

-- name: AddAggregateType :one 
INSERT INTO aggregate_types (name) VALUES (sqlc.arg(aggregate_type_name)) 
	RETURNING id;

-- name: GetAggregateTypes :many
SELECT id, name FROM aggregate_types;

-- name: GetAggregateTypeIdByName :one
SELECT id FROM aggregate_types WHERE name=sqlc.arg(aggregate_type_name);

--
-- Event Type Queries
--

-- name: AddEventType :one
INSERT INTO event_types (name) VALUES(sqlc.arg(event_name))
	RETURNING id;

-- name: GetEventTypes :many
SELECT id, name FROM event_types;

-- name: GetEventTypeIdByName :one
SELECT id FROM event_types WHERE name=sqlc.arg(event_name);


--
-- Aggregate Queries
-- 

-- name: AddAggregate :one
INSERT INTO aggregates (aggregate_type_id) VALUES (sqlc.arg(aggregate_type_id)) 
	RETURNING id;

-- name: AddAggregateWithNaturalKey :one
INSERT INTO aggregates (aggregate_type_id, natural_key) VALUES(sqlc.arg(aggregate_type_id), sqlc.arg(natural_key)) 
	RETURNING id;

-- name: ChangeAggregateNaturalKey :exec
UPDATE aggregates SET natural_key=sqlc.arg(natural_key) WHERE id=sqlc.arg(aggregate_id);

-- name: GetAggregateIdByNaturalKey :one
SELECT id FROM aggregates WHERE aggregate_type_id=sqlc.arg(aggregate_type_id) and natural_key=sqlc.arg(natural_key);

-- name: GetAggregateById :one
SELECT id, natural_key FROM aggregates WHERE aggregate_type_id=sqlc.arg(aggregate_type_id) AND id=sqlc.arg(aggregate_id);

--
-- Event Queries
--

-- name: AddEvent :exec
INSERT INTO events (aggregate_id, sequence, event_type_id, state, event_time, reference)
	VALUES(
		sqlc.arg(aggregate_id), 
		sqlc.arg(sequence),
		sqlc.arg(event_type_id), 
		sqlc.arg(state),
		sqlc.arg(event_time),
		sqlc.arg(reference)
	);


-- name: GetEventsForAggregate :many
SELECT sequence, et.name event_type, state, event_time, reference
FROM events e
JOIN event_types AS et ON e.event_type_id = et.id
WHERE e.aggregate_id = sqlc.arg(aggregate_id) AND sequence > sqlc.arg(after_sequence) 
ORDER BY sequence;

-- name: AddSnapshot :exec
INSERT INTO snapshots (aggregate_id, sequence, state) VALUES(sqlc.arg(aggregate_id), sqlc.arg(sequence), sqlc.arg(state));

-- name: GetMostRecentSnapshot :one
SELECT aggregate_id, sequence, state
FROM snapshots
WHERE aggregate_id=sqlc.arg(aggregate_id) 
ORDER BY sequence DESC
LIMIT 1;


--
-- Subscription Queries
--

-- name: UpsertSubscription :one
INSERT INTO subscriptions (
  name, aggregate_type_id, event_type_id, aggregate_key,
  start_from, start_event_id, start_timestamp
) VALUES (
  sqlc.arg(name), sqlc.arg(aggregate_type_id), sqlc.arg(event_type_id), sqlc.arg(aggregate_key),
  sqlc.arg(start_from), sqlc.arg(start_event_id), sqlc.arg(start_timestamp)
)
ON CONFLICT(name) DO UPDATE SET
  aggregate_type_id=EXCLUDED.aggregate_type_id,
  event_type_id=EXCLUDED.event_type_id,
  aggregate_key=EXCLUDED.aggregate_key,
  start_from=EXCLUDED.start_from,
  start_event_id=EXCLUDED.start_event_id,
  start_timestamp=EXCLUDED.start_timestamp,
  updated_at=now()
RETURNING id;

-- name: AddSubscriptionEventType :exec
INSERT INTO subscription_event_types (subscription_id, event_type_id)
VALUES (sqlc.arg(subscription_id), sqlc.arg(event_type_id))
ON CONFLICT DO NOTHING;

-- name: GetSubscriptionByName :one
SELECT id, name, aggregate_type_id, event_type_id, aggregate_key,
       start_from, start_event_id, start_timestamp,
       last_event_id, active, lease_owner, lease_expires_at
FROM subscriptions WHERE name = sqlc.arg(name);

-- name: SetSubscriptionActive :exec
UPDATE subscriptions SET active = sqlc.arg(active), updated_at=now() WHERE id = sqlc.arg(id);

-- name: ClaimSubscription :execrows
UPDATE subscriptions
SET lease_owner = sqlc.arg(owner), lease_expires_at = sqlc.arg(lease_expires_at), updated_at=now()
WHERE name = sqlc.arg(name) AND active = TRUE AND (lease_owner IS NULL OR lease_expires_at < sqlc.arg(now));

-- name: RenewSubscription :execrows
UPDATE subscriptions
SET lease_expires_at = sqlc.arg(lease_expires_at), updated_at=now()
WHERE name = sqlc.arg(name) AND lease_owner = sqlc.arg(owner) AND active = TRUE;

-- name: ReleaseSubscription :exec
UPDATE subscriptions SET lease_owner = NULL, lease_expires_at = NULL, updated_at=now()
WHERE name = sqlc.arg(name) AND lease_owner = sqlc.arg(owner);

-- name: GetEventsForSubscription :many
SELECT id, aggregate_id, natural_key, sequence, aggregate_type_id, aggregate_type, event_type_id, event_type, event_time, state
FROM event_log
WHERE id > sqlc.arg(after_id)
  AND (sqlc.arg(aggregate_type_id) IS NULL OR aggregate_type_id = sqlc.arg(aggregate_type_id))
  AND (sqlc.arg(aggregate_key) IS NULL OR natural_key = sqlc.arg(aggregate_key))
  AND (
    (sqlc.arg(use_multi) = TRUE AND event_type_id IN (SELECT event_type_id FROM subscription_event_types WHERE subscription_id = sqlc.arg(subscription_id)))
    OR
    (sqlc.arg(use_multi) = FALSE AND (sqlc.arg(event_type_id) IS NULL OR event_type_id = sqlc.arg(event_type_id)))
  )
ORDER BY id ASC
LIMIT sqlc.arg(limit);

-- name: AdvanceSubscriptionCursor :exec
UPDATE subscriptions SET last_event_id = sqlc.arg(last_event_id), updated_at=now() WHERE id = sqlc.arg(id);

-- name: GetMaxEventId :one
SELECT COALESCE(MAX(id), 0) AS id FROM events;

-- name: GetFirstEventIdFromTimestamp :one
SELECT id FROM events WHERE event_time >= sqlc.arg(ts) ORDER BY id ASC LIMIT 1;
