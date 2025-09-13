package evercore

import (
    "context"
    "fmt"
    "time"
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

	// Changes the natural key of an aggregate.
	ChangeAggregateNaturalKey(tx StorageEngineTxInfo, ctx context.Context, aggregateId int64, naturalKey string) error

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

    // -------- Durable Subscriptions --------
    // Create or update a subscription by name. If updating, last_event_id is not modified.
    UpsertSubscription(tx StorageEngineTxInfo, ctx context.Context, name string, aggregateTypeId *int64, eventTypeId *int64, aggregateKey *string, startFrom string, startEventId int64, startTimestamp *time.Time) (int64, error)
    // Optionally associate multiple event types to a subscription (enables multi-type filtering).
    AddSubscriptionEventType(tx StorageEngineTxInfo, ctx context.Context, subscriptionId int64, eventTypeId int64) error
    // Fetch a subscription by name.
    GetSubscriptionByName(tx StorageEngineTxInfo, ctx context.Context, name string) (*Subscription, error)
    // Activate/deactivate a subscription.
    SetSubscriptionActive(tx StorageEngineTxInfo, ctx context.Context, id int64, active bool) error

    // Cooperative lease management for distributed runners.
    ClaimSubscription(tx StorageEngineTxInfo, ctx context.Context, name string, owner string, lease time.Duration) (bool, error)
    RenewSubscription(tx StorageEngineTxInfo, ctx context.Context, name string, owner string, lease time.Duration) (bool, error)
    ReleaseSubscription(tx StorageEngineTxInfo, ctx context.Context, name string, owner string) error

    // Event streaming for a subscription and cursor advancement.
    GetEventsForSubscription(tx StorageEngineTxInfo, ctx context.Context, sub *Subscription, limit int) ([]SerializedEvent, error)
    AdvanceSubscriptionCursor(tx StorageEngineTxInfo, ctx context.Context, id int64, lastEventId int64) error

    // Helpers for initializing cursors.
    GetMaxEventId(tx StorageEngineTxInfo, ctx context.Context) (int64, error)
    GetFirstEventIdFromTimestamp(tx StorageEngineTxInfo, ctx context.Context, ts time.Time) (int64, error)
}
