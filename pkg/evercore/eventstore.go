package evercore

import (
	"context"
	"time"
)

// Used internally to compare time passed in with zerotime.
var zeroTime time.Time

type EventStore struct {
	storageEngine       StorageEngine
	eventTypeLookup     map[string]int64
	aggregateTypeLookup map[string]int64
}

// Methods for the EventStoreContext uses.
type ContextOwner interface {
	newAggregate(stx *EventStoreContextType, aggregateType string) (int64, error)
	newAggregateWithKey(stx *EventStoreContextType, aggregateType string, naturalKey string) (int64, error)
	getAggregateIdByKey(stx *EventStoreContextType, aggregateType string, naturalKey string) (int64, error)
	getAggregateById(stx *EventStoreContextType, aggregateType string, aggregateId int64) (int64, *string, error)
	loadSnapshot(stx *EventStoreContextType, aggregateId int64) (*Snapshot, error)
	loadEvents(stx *EventStoreContextType, aggregateId int64, afterSequence int64) (EventSlice, error)
}

func NewEventStore(storageEngine StorageEngine) *EventStore {
	var eventStore = EventStore{
		storageEngine:       storageEngine,
		eventTypeLookup:     make(map[string]int64, 0),
		aggregateTypeLookup: make(map[string]int64, 0),
	}
	return &eventStore
}

type readonlyContextFunc func(ctx EventStoreReadonlyContext) error
type readonlyContextFuncReturns[T any] func(ctx EventStoreReadonlyContext) (T, error)
type contextFunc func(ctx EventStoreContext) error
type contextFuncReturns[T any] func(ctx EventStoreContext) (T, error)

// Executes a function within a readonly context returning a value.
func InReadonlyContext[T any](ctx context.Context, store *EventStore, exec readonlyContextFuncReturns[T]) (T, error) {

	// Initialize the context
	context := newEventStoreReadonlyContextType(store, ctx)

	// Execute the callback
	return exec(context)
}

// Executes a function callback that returns a value after the context
// complets.  This is the similar to the EventStore.Incontext method
// except it allows a return value after completion.
func InContext[T any](ctx context.Context, store *EventStore, exec contextFuncReturns[T]) (T, error) {

	transaction, err := store.storageEngine.GetTransactionInfo()
	if err != nil {
		var result T
		return result, err
	}

	// Initialize the context
	context := newEventStoreContextType(store, ctx, transaction)

	// Execute the callback
	result, err := exec(context)
	if err != nil {
		return result, err
	}

	// Write events to storage engine
	err = store.SaveState(context.context, transaction, context.capturedEvents, context.snapshots)
	if err != nil {
		return result, err
	}
	err = transaction.Commit()
	if err != nil {
		return result, err
	}
	return result, nil
}

// Executes a callback function within a readonly context
func (store *EventStore) WithReadonlyContext(ctx context.Context, exec readonlyContextFunc) error {

	// Initialize the context
	context := newEventStoreReadonlyContextType(store, ctx)

	// Execute the callback
	err := exec(context)
	if err != nil {
		return err
	}
	return nil
}

// Executes a function inside a context
// This is used to ensure the all events and snapshots are written after
// the context completes or that they are not written when an error
// happens.
func (store *EventStore) WithContext(ctx context.Context, exec contextFunc) error {
	transaction, err := store.storageEngine.GetTransactionInfo()
	if err != nil {
		return err
	}

	// Initialize the context
	context := newEventStoreContextType(store, ctx, transaction)

	// Execute the callback
	err = exec(context)
	if err != nil {
		return err
	}

	// Write events to storage engine
	err = store.SaveState(context.context, transaction, context.capturedEvents, context.snapshots)
	if err != nil {
		return err
	}
	return nil
}

// This is used at the end of a context to save all the events and snapshots
// that happened during the context.
func (store *EventStore) SaveState(ctx context.Context, tx StorageEngineTxInfo, events EventSlice, snapshots SnapshotSlice) error {

	var storageEvents = make([]StorageEngineEvent, 0, len(events))
	for _, event := range events {
		eventTypeId, err := store.getEventTypeId(ctx, event.EventType)
		if err != nil {
			return err
		}

		if event.EventTime == zeroTime {
			event.EventTime = time.Now().UTC()
		}

		storageEvent := StorageEngineEvent{
			AggregateID: event.AggregateId,
			Sequence:    event.Sequence,
			EventTypeID: eventTypeId,
			State:       event.State,
			EventTime:   event.EventTime,
			Reference:   event.Reference,
		}
		storageEvents = append(storageEvents, storageEvent)
	}

	return store.storageEngine.WriteState(tx, ctx, storageEvents, snapshots)
}

// Can be used to warm up the event store by loading existing aggregate types
// and event types.  This is intended to be called during startup of the
// calling application to ensure all known aggregate types and event types
// are pre-loaded into memory. This can save on database calls during
// contexts since these are memoized inside an internal map.
//
// When using Warmup, it should be called as early as possible.
func (store *EventStore) Warmup(ctx context.Context) error {

	// Early return if we already have event types and aggregate types
	if len(store.eventTypeLookup) > 0 && len(store.aggregateTypeLookup) > 0 {
		return nil
	}

	if len(store.eventTypeLookup) == 0 {
		eventTypes, err := store.storageEngine.GetEventTypes(ctx)
		if err != nil {
			return err
		}

		store.eventTypeLookup = MapNameToId(eventTypes)
	}

	if len(store.aggregateTypeLookup) == 0 {
		aggregateTypes, err := store.storageEngine.GetAggregateTypes(ctx)
		if err != nil {
			return err
		}
		store.aggregateTypeLookup = MapNameToId(aggregateTypes)
	}
	return nil
}
func (store *EventStore) newAggregate(stx *EventStoreContextType, aggregateType string) (int64, error) {
	aggregateTypeId, err := store.getAggregateTypeId(aggregateType)
	if err != nil {
		return 0, err
	}
	return store.storageEngine.NewAggregate(stx.Transaction, stx.context, aggregateTypeId)
}

func (store *EventStore) newAggregateWithKey(stx *EventStoreContextType, aggregateType string, naturalKey string) (int64, error) {
	aggregateTypeId, err := store.getAggregateTypeId(aggregateType)
	if err != nil {
		return 0, err
	}
	return store.storageEngine.NewAggregateWithKey(stx.Transaction, stx.context, aggregateTypeId, naturalKey)
}

// Gets the aggregate type id by name.
// This first checks our local map to see if the name already exists, if so,
// we can avoid the database call.
func (store *EventStore) getAggregateTypeId(aggregateTypeName string) (int64, error) {
	aggregateId, exists := store.aggregateTypeLookup[aggregateTypeName]
	if exists {
		return aggregateId, nil
	}

	ctx := context.Background()
	aggregateId, err := store.storageEngine.GetAggregateTypeId(ctx, aggregateTypeName)
	if err != nil {
		return 0, err
	}
	store.aggregateTypeLookup[aggregateTypeName] = aggregateId
	return aggregateId, nil
}

// Gets the event type id by name
// This first checks the local map to see if the name already exists, if so,
// we can avoid the database call.
func (store *EventStore) getEventTypeId(ctx context.Context, eventTypeName string) (int64, error) {
	eventTypeId, exists := store.eventTypeLookup[eventTypeName]
	if exists {
		return eventTypeId, nil
	}

	eventTypeId, err := store.storageEngine.GetEventTypeId(ctx, eventTypeName)
	if err != nil {
		return 0, err
	}

	store.eventTypeLookup[eventTypeName] = eventTypeId
	return eventTypeId, nil
}

// Retrieves an aggregate id by its natural key instead of its id.
func (store *EventStore) getAggregateIdByKey(stx *EventStoreContextType, aggregateTypeName string, naturalKey string) (int64, error) {

	aggregateTypeId, err := store.getAggregateTypeId(aggregateTypeName)
	if err != nil {
		return 0, err
	}

	aggregateId, err := store.storageEngine.GetAggregateByKey(stx.context, aggregateTypeId, naturalKey)
	return aggregateId, err
}

func (store *EventStore) getAggregateById(stx *EventStoreContextType, aggregateTypeName string, aggregateId int64) (int64, *string, error) {

	aggregateTypeId, err := store.getAggregateTypeId(aggregateTypeName)
	if err != nil {
		return 0, nil, err
	}

	id, key, err := store.storageEngine.GetAggregateById(stx.context, aggregateTypeId, aggregateId)
	return id, key, err
}

// Retrieves the most recent snapshot for the aggregate.
func (store *EventStore) loadSnapshot(stx *EventStoreContextType, aggregateId int64) (*Snapshot, error) {
	snapshot, err := store.storageEngine.GetSnapshotForAggregate(stx.context, aggregateId)
	return snapshot, err
}

// Using the afterSequence allows us to only pick events going forward from a
// particular sequence.  This is so if we have a snapshot, we only have to load
// events after that snapshot's sequence.
func (store *EventStore) loadEvents(stx *EventStoreContextType, aggregateId int64, afterSequence int64) (EventSlice, error) {
	return store.storageEngine.GetEventsForAggregate(stx.context, aggregateId, afterSequence)
}
