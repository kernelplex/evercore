
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

