-- +goose Up
-- +goose StatementBegin

CREATE VIEW event_log AS
SELECT 
	e.id,
	e.aggregate_id, a.natural_key,
	e.sequence,
	a.aggregate_type_id, at.name AS aggregate_type,
	e.event_type_id, et.name AS event_type,
	e.event_time,
	e.state
FROM events e
JOIN aggregates a ON e.aggregate_id = a.id
JOIN aggregate_types at ON a.aggregate_type_id = at.id
JOIN event_types et ON e.event_type_id = et.id;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP VIEW IF EXISTS event_log;

-- +goose StatementEnd
