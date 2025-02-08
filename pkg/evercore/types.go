package evercore

import (
	"encoding/json"
	"time"
)

// Represents a pair of id and name.
type IdNamePair struct {
	Id   int64
	Name string
}

// Represents a map of name to id.
type NameIdMap map[string]int64

// Represents a map of id to name.
type IdNameMap map[int64]string

// Represents an event to be published with the state serialized.
type SerializedEvent struct {
	AggregateId int64
	EventType   string
	State       string
	Sequence    int64
	Reference   string
	EventTime   time.Time
}

// Decodes the event state into the specified type.
func DecodeEventStateTo[U any](e SerializedEvent, state *U) error {
	err := json.Unmarshal([]byte(e.State), state)
	if err != nil {
		return err
	}
	return nil
}

// Represents an event mapping to the storage engine.
type StorageEngineEvent struct {
	AggregateID int64
	Sequence    int64
	EventTypeID int64
	State       string
	EventTime   time.Time
	Reference   string
}

// Slice of serialized events.
type EventSlice []SerializedEvent

// Represents a snapshot of an aggregate.
type Snapshot struct {
	AggregateId int64
	State       string
	Sequence    int64
}

// Slice of snapshots.
type SnapshotSlice []Snapshot
