package evercore

import (
	"context"
	"fmt"
)

const maxKeyLength = 64

// Ephemeral memory storage engine useful for testing without a database.
type MemoryStorageEngine struct {
	CapturedEvents    []StorageEngineEvent
	CapturedSnapshots []Snapshot

	AggregateTypes     map[string]int64
	AggregateTypesInv  map[int64]string
	EventTypes         map[string]int64
	EventTypesInv      map[int64]string
	CountAggregateType int64
	CountEventTypes    int64
	CountAggregates    int64
	AggregateToTypeId  map[int64]int64
	Aggregates         map[string]int64
	AggregateInv       map[int64]*string
}

// NewMemoryStorageEngine creates a new in-memory storage engine.
func NewMemoryStorageEngine() *MemoryStorageEngine {
	return &MemoryStorageEngine{
		CapturedEvents:    make([]StorageEngineEvent, 0),
		CapturedSnapshots: make([]Snapshot, 0),
		AggregateTypes:    make(map[string]int64),
		AggregateTypesInv: make(map[int64]string),
		EventTypes:        make(map[string]int64),
		EventTypesInv:     make(map[int64]string),
		Aggregates:        make(map[string]int64),
		AggregateToTypeId: make(map[int64]int64),
		AggregateInv:      make(map[int64]*string),
	}
}

// MemoryStorageEngineTransaction is a transaction for the in-memory storage engine.
type MemoryStorageEngineTransaction struct {
}

func (tx MemoryStorageEngineTransaction) Commit() error {
	// TODO: May want to implement memory transactions?
	return nil
}

func (tx MemoryStorageEngineTransaction) Rollback() error {
	// TODO: May want to implement memory transactions?
	return nil
}

var memoryStorageEngine StorageEngine = NewMemoryStorageEngine()

func (stor *MemoryStorageEngine) GetMaxKeyLength() int {
	return maxKeyLength
}

func (stor *MemoryStorageEngine) GetTransactionInfo() (StorageEngineTxInfo, error) {
	return MemoryStorageEngineTransaction{}, nil
}

func (store *MemoryStorageEngine) GetEventTypeId(tx StorageEngineTxInfo, ctx context.Context, name string) (int64, error) {
	eventType, exists := store.EventTypes[name]
	if exists {
		return eventType, nil
	}
	store.CountEventTypes++
	store.EventTypes[name] = store.CountEventTypes
	store.EventTypesInv[store.CountEventTypes] = name
	return store.CountEventTypes, nil
}

func (store *MemoryStorageEngine) GetAggregateTypeId(tx StorageEngineTxInfo, ctx context.Context, name string) (int64, error) {
	aggregateTypeId, exists := store.AggregateTypes[name]
	if exists {
		return aggregateTypeId, nil
	}

	store.CountAggregateType++
	store.AggregateTypes[name] = store.CountAggregateType
	store.AggregateTypesInv[store.CountAggregateType] = name
	return store.CountAggregateType, nil
}

func (store *MemoryStorageEngine) NewAggregate(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64) (int64, error) {
	store.CountAggregates++
	store.AggregateInv[store.CountAggregates] = nil
	store.AggregateToTypeId[store.CountAggregates] = aggregateTypeId
	return store.CountAggregates, nil
}

func (store *MemoryStorageEngine) NewAggregateWithKey(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error) {
	if len(naturalKey) > maxKeyLength {
		return 0, ErrorKeyExceedsMaximumLength
	}
	_, exists := store.Aggregates[naturalKey]
	if exists {
		return 0, fmt.Errorf("Duplicate key violation.")

	}
	store.CountAggregates++
	store.Aggregates[naturalKey] = store.CountAggregates
	store.AggregateToTypeId[store.CountAggregates] = aggregateTypeId
	store.AggregateInv[store.CountAggregates] = &naturalKey
	return store.CountAggregates, nil
}

func (store *MemoryStorageEngine) ChangeAggregateNaturalKey(tx StorageEngineTxInfo, ctx context.Context, aggregateId int64, naturalKey string) error {

	if len(naturalKey) > maxKeyLength {
		return ErrorKeyExceedsMaximumLength
	}

	// Try to get existing aggregate first
	aggregateKey, exists := store.AggregateInv[aggregateId]
	if !exists {
		return fmt.Errorf("No aggregate exists with Id of %d", aggregateId)
	}

	// Remove the existing key
	delete(store.Aggregates, *aggregateKey)
	delete(store.AggregateInv, aggregateId)

	// Add the new key
	store.Aggregates[naturalKey] = aggregateId
	store.AggregateInv[aggregateId] = &naturalKey

	return nil
}

func (store *MemoryStorageEngine) GetAggregateById(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, aggregateId int64) (int64, *string, error) {
	aggregateKey, exists := store.AggregateInv[aggregateId]
	if !exists {
		return 0, nil, fmt.Errorf("No aggregate exists with Id of %d", aggregateId)
	}
	if store.AggregateToTypeId[aggregateId] != aggregateTypeId {
		return 0, nil, fmt.Errorf("Aggregate exists, but does not match the type")
	}
	return aggregateId, aggregateKey, nil
}

func (store *MemoryStorageEngine) GetOrCreateAggregateByKey(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (bool, int64, error) {
	if len(naturalKey) > maxKeyLength {
		return false, 0, ErrorKeyExceedsMaximumLength
	}

	// Try to get existing aggregate first
	aggregateId, exists := store.Aggregates[naturalKey]
	if exists {
		if store.AggregateToTypeId[aggregateId] != aggregateTypeId {
			return false, 0, fmt.Errorf("aggregate exists but has different type")
		}
		return false, aggregateId, nil
	}

	// Create new aggregate if not found
	store.CountAggregates++
	store.Aggregates[naturalKey] = store.CountAggregates
	store.AggregateToTypeId[store.CountAggregates] = aggregateTypeId
	store.AggregateInv[store.CountAggregates] = &naturalKey
	return true, store.CountAggregates, nil
}

func (store *MemoryStorageEngine) GetAggregateByKey(tx StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error) {
	aggregateId, exists := store.Aggregates[naturalKey]
	if !exists {
		return 0, fmt.Errorf("No aggregate exists with key of %s", naturalKey)
	}
	if store.AggregateToTypeId[aggregateId] != aggregateTypeId {
		return 0, fmt.Errorf("Aggregate exists, but does not match the type")
	}
	return aggregateId, nil
}

func (store *MemoryStorageEngine) GetAggregateTypes(tx StorageEngineTxInfo, ctx context.Context) ([]IdNamePair, error) {
	aggregateTypes := make([]IdNamePair, 0, len(store.AggregateTypes))
	for name, id := range store.AggregateTypes {
		current := IdNamePair{
			Id:   id,
			Name: name,
		}
		aggregateTypes = append(aggregateTypes, current)
	}
	return aggregateTypes, nil
}

func (store *MemoryStorageEngine) GetEventTypes(tx StorageEngineTxInfo, ctx context.Context) ([]IdNamePair, error) {
	eventTypes := make([]IdNamePair, 0, len(store.EventTypes))
	for name, id := range store.EventTypes {
		current := IdNamePair{
			Id:   id,
			Name: name,
		}
		eventTypes = append(eventTypes, current)
	}
	return eventTypes, nil
}

func (store *MemoryStorageEngine) GetSnapshotForAggregate(tx StorageEngineTxInfo, ctx context.Context, aggregateId int64) (*Snapshot, error) {
	for i := len(store.CapturedSnapshots) - 1; i >= 0; i-- {
		if store.CapturedSnapshots[i].AggregateId == aggregateId {
			return &store.CapturedSnapshots[i], nil
		}
	}
	return nil, nil
}

func (store *MemoryStorageEngine) GetEventsForAggregate(tx StorageEngineTxInfo, ctx context.Context, aggregateId int64, afterSequence int64) ([]SerializedEvent, error) {
	aggregateEvents := make([]SerializedEvent, 0, 10)
	for _, storageEvent := range store.CapturedEvents {
		if storageEvent.AggregateID == aggregateId && storageEvent.Sequence > afterSequence {
			event := SerializedEvent{
				AggregateId: aggregateId,
				EventType:   store.EventTypesInv[storageEvent.EventTypeID],
				Sequence:    storageEvent.Sequence,
				State:       storageEvent.State,
				Reference:   storageEvent.Reference,
				EventTime:   storageEvent.EventTime,
			}
			aggregateEvents = append(aggregateEvents, event)
		}
	}
	return aggregateEvents, nil
}

func (store *MemoryStorageEngine) WriteState(tx StorageEngineTxInfo, ctx context.Context, events []StorageEngineEvent, snapshot SnapshotSlice) error {
	store.CapturedEvents = append(store.CapturedEvents, events...)
	store.CapturedSnapshots = append(store.CapturedSnapshots, snapshot...)
	return nil
}

func (store *MemoryStorageEngine) Close() error {
	// Nothing to do
	return nil
}
