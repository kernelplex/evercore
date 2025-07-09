package evercore

import (
	"context"
	"fmt"
)

var ErrorKeyExceedsMaximumLength error = fmt.Errorf("The specified key exceeds the maximum key length")

// A storage engine contains the raw methods for interacting with the event store database.
//
// The storage engine implementations should implemented to be safe for calling from
// multiple goroutines simultaneously.
type StorageEngine interface {

	// Gets the maximum length of a natural key
	GetMaxKeyLength() int

	// Gets the id for the event type string.  It is added if it does not already exist.
	GetEventTypeId(tx StorageEngineTxInfo, ctx context.Context, typeName string) (int64, error)

	// Gets the id for the aggregate type string.  It is added if it does not already exist.
	GetAggregateTypeId(tx StorageEngineTxInfo, ctx context.Context, typeName string) (int64, error)

	// Adds a new aggregate of the specified type to the store and returns the id.
	NewAggregate(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64) (int64, error)

	// Adds a new aggregate of the specified type with the natural key to the store and returns the id.
	NewAggregateWithKey(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, natrualKey string) (int64, error)

	// Gets aggregate id and corresponding natural key that corresponds to the type and id.
	GetAggregateById(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, aggregateId int64) (int64, *string, error)

	// Gets the aggregate id corresponding to the type and natural key.
	GetAggregateByKey(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error)

	// Gets or creates an aggregate id for the given type and natural key.
	// If an aggregate with the given natural key already exists, returns false and its id.
	// Otherwise creates a new aggregate with the key and returns true and its new id.
	// Returns error if:
	// - The natural key exceeds maximum length (ErrorKeyExceedsMaximumLength)
	// - The aggregate type doesn't exist
	// - There are database errors
	GetOrCreateAggregateByKey(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (bool, int64, error)

	// Loads all the aggregate types.
	GetAggregateTypes(tx StorageEngineTxInfo, ctx context.Context) ([]IdNamePair, error)

	// Loads all the event types.
	GetEventTypes(tx StorageEngineTxInfo, ctx context.Context) ([]IdNamePair, error)

	// Gets a snapshot (if any) for the specified aggregate.
	GetSnapshotForAggregate(tx StorageEngineTxInfo, ctx context.Context, aggregateId int64) (*Snapshot, error)

	// Retrieve events for an aggregate which are after a sequence.
	GetEventsForAggregate(tx StorageEngineTxInfo, ctx context.Context, aggregateId int64, afterSequence int64) ([]SerializedEvent, error)

	// Writes events to storage.
	WriteState(tx StorageEngineTxInfo, ctx context.Context, events []StorageEngineEvent, snapshot SnapshotSlice) error

	// Gets a transaction state to track multiple
	GetTransactionInfo() (StorageEngineTxInfo, error)

	// Closes the storage engine.
	Close() error
}
