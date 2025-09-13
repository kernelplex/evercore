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
    // EventID is the global, monotonically increasing id from the events table.
    // It may be zero when events are loaded by-aggregate (not via event_log).
    EventID    int64
    AggregateId int64
    EventType   string
    State       string
    Sequence    int64
    Reference   string
    EventTime   time.Time
}

// Subscription describes a durable reader over the event log with optional filters
// and a durable cursor stored as last_event_id.
type Subscription struct {
    ID               int64
    Name             string
    AggregateTypeID  *int64
    EventTypeID      *int64
    AggregateKey     *string
    StartFrom        string
    StartEventID     int64
    StartTimestamp   *time.Time
    LastEventID      int64
    Active           bool
    LeaseOwner       *string
    LeaseExpiresAt   *time.Time
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

// StorageEngineError represents an error from the storage engine.
type StorageEngineError struct {
	// The underlying error that triggered this error
	Err error
	// A human-readable message describing the error
	Message string
	// The type of error that occurred
	ErrorType string
}

// Error returns the string representation of the error.
func (e *StorageEngineError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *StorageEngineError) Unwrap() error {
	return e.Err
}

// NewStorageEngineError creates a new StorageEngineError.
func NewStorageEngineError(message string, err error) *StorageEngineError {
	return &StorageEngineError{
		Err:     err,
		Message: message,
	}
}

// Common storage engine error types
const (
	ErrorNotFound                = "not_found"
	ErrorTypeConstraintViolation = "constraint_violation"
)
