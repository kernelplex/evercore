package evercore

import (
	"context"
	"fmt"
	"time"
)

// Represents the state of the aggregate including the latest snapshot (if it
// exists) and the events required to reconstitute its current state.
type AggregateState struct {
	AggregateId int64
	NaturalKey  *string
	Snapshot    *Snapshot
	Events      EventSlice
}

// EventState represents the state of an event.
type EventState interface {
	GetEventType() string
	Serialize() string
}

// Interface represents transaction info needed to track the transaction.
type StorageEngineTxInfo interface {
	Commit() error
	Rollback() error
}

// Represents a readonly context access to the event store.
type EventStoreReadonlyContext interface {
	LoadStateInto(aggregate Aggregate, id int64) error
	LoadStateByKeyInto(aggregate Aggregate, naturalKey string) error

	LoadAggregateState(aggregateType string, aggregateId int64) (*AggregateState, error)
	LoadAggregateStateByKey(aggregateType string, naturalKey string) (*AggregateState, error)
}

// Represents a read/write context access to the event store.
type EventStoreContext interface {
	EventStoreReadonlyContext
	NewAggregateId(aggregateType string) (int64, error)
	NewAggregateIdWithKey(aggregateType string, naturalKey string) (int64, error)

	LoadOrCreateAggregate(agg Aggregate, naturalKey string) (bool, error)
	CreateAggregateInto(agg Aggregate) error
	CreateAggregateWithKeyInto(agg Aggregate, naturalKey string) error
	ChangeAggregateNaturalKey(id int64, naturalKey string) error
	ApplyEventTo(agg Aggregate, event EventState, time time.Time, reference string) error

	Publish(*SerializedEvent)
	SaveSnapshot(snapshot Snapshot)
}

// Tracks information related to the current event store context.
type EventStoreContextType struct {
	capturedEvents EventSlice
	snapshots      SnapshotSlice
	Transaction    StorageEngineTxInfo
	store          ContextOwner
	context        context.Context
}

func newEventStoreReadonlyContextType(store ContextOwner, ctx context.Context) *EventStoreContextType {
	return &EventStoreContextType{
		capturedEvents: make(EventSlice, 0, 5),
		snapshots:      make(SnapshotSlice, 0, 5),
		Transaction:    nil,
		store:          store,
		context:        ctx,
	}

}

func newEventStoreContextType(store ContextOwner,
	ctx context.Context,
	transaction StorageEngineTxInfo) *EventStoreContextType {
	return &EventStoreContextType{
		capturedEvents: make(EventSlice, 0, 5),
		snapshots:      make(SnapshotSlice, 0, 5),
		Transaction:    transaction,
		store:          store,
		context:        ctx,
	}
}

// Publishes an event for storage once the context completes.
func (ctx *EventStoreContextType) Publish(event *SerializedEvent) {
	ctx.capturedEvents = append(ctx.capturedEvents, *event)
}

// Adds a new aggregate stream and returns the resulting id.
func (ctx EventStoreContextType) NewAggregateId(aggregateType string) (int64, error) {
	return ctx.store.newAggregate(&ctx, aggregateType)
}

// Adds a new aggregate stream with the specified natural key and returns the resulting id.
func (ctx EventStoreContextType) NewAggregateIdWithKey(
	aggregateType string,
	naturalKey string) (int64, error) {
	return ctx.store.newAggregateWithKey(&ctx, aggregateType, naturalKey)
}

// Loads the most recent snapshot and events from the event store
func (ctx *EventStoreContextType) LoadAggregateState(
	aggregateType string,
	aggregateId int64) (*AggregateState, error) {
	_, key, err := ctx.store.getAggregateById(ctx, aggregateType, aggregateId)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregate by ID in LoadAggregateState: %w", err)
	}

	return ctx.loadState(aggregateId, key)
}

// Loads the aggregate state using the natural key.
func (ctx *EventStoreContextType) LoadAggregateStateByKey(
	aggregateType string,
	naturalKey string) (*AggregateState, error) {
	aggregateId, err := ctx.store.getAggregateIdByKey(ctx, aggregateType, naturalKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get aggregate by key in LoadAggregateStateByKey: %w", err)
	}

	return ctx.loadState(aggregateId, &naturalKey)
}

func (ctx *EventStoreContextType) loadState(
	aggregateId int64,
	naturalKey *string) (*AggregateState, error) {
	snapshot, err := ctx.store.loadSnapshot(ctx, aggregateId)
	if err != nil {
		return nil, fmt.Errorf("failed to load snapshot in loadState: %w", err)
	}

	var sequence int64 = 0
	if snapshot != nil {
		sequence = snapshot.Sequence
	}

	events, err := ctx.store.loadEvents(ctx, aggregateId, sequence)
	if err != nil {
		return nil, fmt.Errorf("failed to load events in loadState: %w", err)
	}

	state := AggregateState{
		AggregateId: aggregateId,
		NaturalKey:  naturalKey,
		Snapshot:    snapshot,
		Events:      events,
	}

	return &state, nil
}

// Stages a snapshot to save once the context completes.
func (etx *EventStoreContextType) SaveSnapshot(snapshot Snapshot) {
	etx.snapshots = append(etx.snapshots, snapshot)
}

// Loads the state of an aggregate into the aggregate.
func (etx *EventStoreContextType) LoadStateInto(
	agg Aggregate,
	aggregateId int64) error {
	aggregateType := agg.GetAggregateType()

	state, err := etx.LoadAggregateState(aggregateType, aggregateId)
	if err != nil {
		return err
	}
	err = applyState(state, agg)
	if err != nil {
		return fmt.Errorf("failed to apply state in LoadStateInto: %w", err)
	}
	return nil
}

// Loads the state of an aggregate into the aggregate using the natural key.
func (etx *EventStoreContextType) LoadStateByKeyInto(
	agg Aggregate,
	naturalKey string) error {
	aggregateType := agg.GetAggregateType()

	state, err := etx.LoadAggregateStateByKey(aggregateType, naturalKey)
	if err != nil {
		return err
	}

	err = applyState(state, agg)
	if err != nil {
		return fmt.Errorf("failed to apply state in LoadStateByKeyInto: %w", err)
	}
	return nil
}

// Creates a new aggregate of the specified type.
func (etx *EventStoreContextType) CreateAggregateInto(agg Aggregate) error {
	aggregateType := agg.GetAggregateType()
	id, err := etx.NewAggregateId(aggregateType)
	if err != nil {
		return err
	}
	agg.SetId(id)
	return nil
}

// LoadOrCreateAggregate loads an existing aggregate by natural key if it exists,
// otherwise creates a new one. Returns true if the aggregate was created, false if loaded.
func (etx *EventStoreContextType) LoadOrCreateAggregate(
	agg Aggregate,
	naturalKey string) (bool, error) {
	aggregateType := agg.GetAggregateType()

	created, aggregateId, err := etx.store.getOrCreateAggregateByKey(etx, aggregateType, naturalKey)
	if err != nil {
		return false, err
	}

	if created {
		agg.SetId(aggregateId)
		return true, nil
	}

	err = etx.LoadStateByKeyInto(agg, naturalKey)
	if err != nil {
		return false, err
	}

	return false, nil
}

// Creates a new aggregate of the specified type with the specified natural key.
func (etx *EventStoreContextType) CreateAggregateWithKeyInto(
	agg Aggregate,
	naturalKey string) error {
	aggregateType := agg.GetAggregateType()
	id, err := etx.NewAggregateIdWithKey(aggregateType, naturalKey)
	if err != nil {
		return err
	}
	agg.SetId(id)
	return nil
}

// Changes the natural key of an aggregate.
func (etx *EventStoreContextType) ChangeAggregateNaturalKey(
	id int64,
	naturalKey string) error {
	err := etx.store.changeAggregateNaturalKey(etx, id, naturalKey)
	if err != nil {
		return fmt.Errorf("failed to change aggregate natural key in ChangeAggregateNaturalKey: %w", err)
	}
	return nil
}

func applyState(state *AggregateState, agg Aggregate) error {
	var sequence int64 = 0
	if state.Snapshot != nil {
		err := agg.ApplySnapshot(state.Snapshot)
		if err != nil {
			return err
		}
		sequence = state.Snapshot.Sequence
	}

	for _, event := range state.Events {
		_, decoded, err := DecodeEvent(event)
		if err != nil {
			return fmt.Errorf("failed to decode event in applyState: %w", err)
		}
		err = agg.ApplyEventState(decoded, event.EventTime, event.Reference)
		if err != nil {
			return fmt.Errorf("failed to apply event state in applyState: %w", err)
		}
		sequence = event.Sequence
	}
	agg.SetId(state.AggregateId)
	agg.SetSequence(sequence)
	return nil
}

// Applies an event to the aggregate.
func (etx *EventStoreContextType) ApplyEventTo(
	agg Aggregate,
	eventState EventState,
	time time.Time,
	reference string) error {
	err := agg.ApplyEventState(eventState, time, reference)
	if err != nil {
		return err
	}
	newSequence := agg.GetSequence() + 1
	event := SerializedEvent{
		AggregateId: agg.GetId(),
		EventType:   eventState.GetEventType(),
		State:       eventState.Serialize(),
		Sequence:    newSequence,
		Reference:   reference,
		EventTime:   time,
	}
	agg.SetSequence(newSequence)
	etx.Publish(&event)

	snapshotFrequency := agg.GetSnapshotFrequency()
	if snapshotFrequency > 0 && newSequence%snapshotFrequency == 0 {
		snapshotState, err := agg.GetSnapshotState()
		if err != nil {
			return err
		}
		snapshot := Snapshot{
			AggregateId: agg.GetId(),
			State:       *snapshotState,
			Sequence:    newSequence,
		}
		etx.SaveSnapshot(snapshot)
	}
	return nil
}
