package evercore

import "time"

// Optinionated version of aggregate
type Aggregate interface {
	GetId() int64
	SetId(int64)
	GetSequence() int64
	SetSequence(int64)
	GetAggregateType() string
	GetSnapshotFrequency() int64
	GetSnapshotState() (*string, error)
	// TODO: We want to remove this.
	// DecodeEvent(ev SerializedEvent) (EventState, error)
	ApplyEventState(eventState EventState, eventTime time.Time, reference string) error
	ApplySnapshot(snapshot *Snapshot) error
}
