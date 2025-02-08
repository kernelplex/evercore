-- +goose Up
-- +goose StatementBegin
CREATE TABLE aggregate_types (
	id INTEGER NOT NULL PRIMARY KEY,
	name VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE event_types (
	id INTEGER NOT NULL PRIMARY KEY,
	name VARCHAR(64) NOT NULL UNIQUE
);

CREATE TABLE aggregates (
	id INTEGER NOT NULL PRIMARY KEY,
	aggregate_type_id INTEGER NOT NULL,
	natural_key VARCHAR(64) UNIQUE
);

CREATE TABLE events (
	id INTEGER NOT NULL PRIMARY KEY,
	aggregate_id INTEGER NOT NULL,
	sequence INTEGER NOT NULL,
	event_type_id INTEGER NOT NULL,
	state TEXT NOT NULL,
	event_time DATETIME NOT NULL,
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
	id INTEGER NOT NULL PRIMARY KEY,
	aggregate_id INTEGER NOT NULL,
	sequence INTEGER NOT NULL,
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
