-- +goose Up
-- +goose StatementBegin


CREATE TABLE aggregate_types (
	id BIGSERIAL NOT NULL PRIMARY KEY,
	name VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE event_types (
	id BIGSERIAL NOT NULL PRIMARY KEY,
	name VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE aggregates (
	id BIGSERIAL NOT NULL PRIMARY KEY,
	aggregate_type_id BIGINT NOT NULL,
	natural_key VARCHAR(64) UNIQUE
);


CREATE TABLE events (
	id BIGSERIAL NOT NULL PRIMARY KEY,
	aggregate_id BIGINT NOT NULL,
	sequence BIGINT NOT NULL,
	event_type_id BIGINT NOT NULL,
	state TEXT NOT NULL,
	event_time TIMESTAMP NOT NULL,
	reference TEXT NOT NULL DEFAULT '',

	FOREIGN KEY(aggregate_id) REFERENCES
		aggregates(id),
	FOREIGN KEY(event_type_id) REFERENCES
		event_types(id),
	UNIQUE(aggregate_id, sequence)
);
CREATE INDEX event_event_time ON events(aggregate_id, event_time);
CREATE INDEX event_reference ON events(reference);

CREATE TABLE snapshots (
	id BIGSERIAL NOT NULL PRIMARY KEY,
	aggregate_id BIGINT NOT NULL,
	sequence BIGINT NOT NULL,
	state TEXT NOT NULL,
	UNIQUE(aggregate_id, sequence)
);

-- +goose StatementEnd




-- +goose Down
-- +goose StatementBegin

DROP TABLE events;
DROP TABLE aggregates;
DROP TABLE event_types;
DROP TABLE aggregate_types;
DROP TABLE snapshots;

-- +goose StatementEnd
