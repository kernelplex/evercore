package evercoreenginetests

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/kernelplex/evercore/base"
)

type StorageEngineTestSuite struct {
	iut evercore.StorageEngine

	aggregateNoKey   int64
	aggregateWithKey int64

	aggregateTypeMap      evercore.NameIdMap
	eventTypeMap          evercore.NameIdMap
	eventTypeMapInv       evercore.IdNameMap
	capturedStorageEvents []evercore.StorageEngineEvent
	capturedEvents        []evercore.SerializedEvent
	capturedSnapshots     []evercore.Snapshot
}

func NewStorageEngineTestSuite(iut evercore.StorageEngine) *StorageEngineTestSuite {
	return &StorageEngineTestSuite{
		iut:              iut,
		aggregateTypeMap: make(evercore.NameIdMap, knownAggregateTypes),
		eventTypeMap:     make(evercore.NameIdMap, knownAggregateTypes),
		eventTypeMapInv:  make(evercore.IdNameMap, knownAggregateTypes),
	}
}

func (s *StorageEngineTestSuite) RunTests(t *testing.T) {
	t.Run("Get a new AggregateType", s.canGetNewAggregateType)
	t.Run("Get an existing AggregateType", s.canGetExistingAggregateType)
	t.Run("Get all aggregate types", s.getAllAggregateTypes)

	t.Run("Get a new EventType", s.canGetNewEventType)
	t.Run("Get an existing EventType", s.canGetExistingEventType)
	t.Run("Get all event types", s.getAllEventTypes)

	t.Run("Creating a new aggregate by id, no key", s.createNewAggregate)
	t.Run("Retrieve an existing aggregate by id", s.getAggregate)
	t.Run("Retrieve a missing aggregate", s.getAggregateMissingId)
	t.Run("Retrieve an aggregate with incorrect aggregate type", s.getAggregateIncorrectType)

	t.Run("Creating a new keyed aggregate", s.createAggregateWithKey)
	t.Run("Creating an aggregate with duplicate key", s.createAggregateWithDuplicateKey)
	t.Run("Creting an aggregate with the maximum key length", s.createAggregateWithMaximumKeyLength)
	t.Run("Creting an aggregate exceeding the maximum key length", s.createAggregateExceedingMaximumKeyLength)
	t.Run("Retrieving a keyed aggregate by id", s.getKeyedAggregateById)
	t.Run("Retrieving an existing aggregate by key", s.getKeyedAggregateByKey)
	t.Run("Retrieving a missing aggregate by key", s.getAggregateByMissingKey)
	t.Run("Retrieving an existing aggregate by key with wrong aggregate type", s.getKeyedAggregateWithWrongAggregateType)

	t.Run("Writing events and snapshots for an aggregate", s.writeState)
	t.Run("Reading existing snapshot for an aggregate", s.getExistingSnapshot)
	t.Run("Reading missing snapshot for an aggregate", s.getMissingSnapshot)

	t.Run("Reading existing events for aggregate", s.getEvents)
	t.Run("Reading existing events for aggregate with no events", s.getEventsForAggregateWithNoEvents)
	t.Run("Reading existing events for aggregate after a specified sequence", s.getEventsAfterSequence)
	t.Run("Reading events after last sequence", s.getEventsAfterLastSequence)

}

const knownAggregateTypes = 2

const userAggregateType = "aggt_user"

var userAggregateTypeId int64

const profileAggregateType = "aggt_profile"

var profileAggregateTypeId int64

const knownEventTypes = 4

const userCreatedEvent = "eventt_user_created"

var userCreatedEventId int64

const userUpdatedEvent = "eventt_user_created"

var userUpdatedEventId int64

const profileCreatedEvent = "eventt_profile_created"

var profileCreatedEventId int64

const profileUpdatedEvent = "eventt_profile_created"

var profileUpdatedEventId int64

func (s *StorageEngineTestSuite) TODO(t *testing.T) {
	t.Errorf("Not implemented.")
}

// ============================================================================
// Testing Aggregate Types
// ============================================================================

func (s *StorageEngineTestSuite) canGetNewAggregateType(t *testing.T) {
	ctx := context.Background()
	var err error

	userAggregateTypeId, err = s.iut.GetAggregateTypeId(ctx, userAggregateType)
	if err != nil {
		t.Errorf("Failed to get aggregate type: %v", err)
	}

	profileAggregateTypeId, err = s.iut.GetAggregateTypeId(ctx, profileAggregateType)
	if err != nil {
		t.Errorf("Failed to get aggregate type: %v", err)
	}

	s.aggregateTypeMap[userAggregateType] = userAggregateTypeId
	s.aggregateTypeMap[profileAggregateType] = profileAggregateTypeId
}

func (s *StorageEngineTestSuite) canGetExistingAggregateType(t *testing.T) {
	ctx := context.Background()

	retrievedUserAggregateId, err := s.iut.GetAggregateTypeId(ctx, userAggregateType)
	if err != nil {
		t.Errorf("Failed to get aggregate type: %v", err)
	}

	if retrievedUserAggregateId != userAggregateTypeId {
		t.Errorf("Retrieved aggregate type id does not match expected: %d != %d", retrievedUserAggregateId, userAggregateTypeId)
	}
}

func (s *StorageEngineTestSuite) getAllAggregateTypes(t *testing.T) {
	ctx := context.Background()
	retrievedAggregateTypes, err := s.iut.GetAggregateTypes(ctx)
	if err != nil {
		t.Errorf("Failed to get all aggregate types: %v", err)
	}

	retrievedMap := evercore.MapNameToId(retrievedAggregateTypes)
	if !reflect.DeepEqual(s.aggregateTypeMap, retrievedMap) {
		t.Errorf("Retrieved aggregates do not match expected")
		t.Errorf("Expected %+v", s.aggregateTypeMap)
		t.Errorf("Received %+v", retrievedMap)
	}
}

// ============================================================================
// Testing Event Types
// ============================================================================

func (s *StorageEngineTestSuite) canGetNewEventType(t *testing.T) {
	ctx := context.Background()
	var err error

	userCreatedEventId, err = s.iut.GetEventTypeId(ctx, userCreatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", userCreatedEvent, err)
	}

	userUpdatedEventId, err = s.iut.GetEventTypeId(ctx, userUpdatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", userUpdatedEvent, err)
	}

	profileCreatedEventId, err = s.iut.GetEventTypeId(ctx, profileCreatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", profileCreatedEvent, err)
	}

	profileUpdatedEventId, err = s.iut.GetEventTypeId(ctx, profileUpdatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", profileUpdatedEvent, err)
	}

	s.eventTypeMap[userCreatedEvent] = userCreatedEventId
	s.eventTypeMapInv[userCreatedEventId] = userCreatedEvent
	s.eventTypeMap[userUpdatedEvent] = userUpdatedEventId
	s.eventTypeMapInv[userUpdatedEventId] = userUpdatedEvent
	s.eventTypeMap[profileCreatedEvent] = profileCreatedEventId
	s.eventTypeMap[profileUpdatedEvent] = profileUpdatedEventId

}

func (s *StorageEngineTestSuite) canGetExistingEventType(t *testing.T) {
	ctx := context.Background()

	retrievedEventId, err := s.iut.GetEventTypeId(ctx, profileCreatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", profileCreatedEvent, err)
	}

	if retrievedEventId != profileCreatedEventId {
		t.Errorf("Received incorrect id for '%s' expected %d got %d",
			profileCreatedEvent, profileCreatedEventId, retrievedEventId)
	}
}

func (s *StorageEngineTestSuite) getAllEventTypes(t *testing.T) {
	ctx := context.Background()

	retrievedEvents, err := s.iut.GetEventTypes(ctx)
	if err != nil {
		t.Errorf("Failed to get event types: %v", err)
	}
	retrievedMap := evercore.MapNameToId(retrievedEvents)
	if !reflect.DeepEqual(s.eventTypeMap, retrievedMap) {
		t.Errorf("Retrieved events does not match expected")
		t.Errorf("Expected %+v", s.eventTypeMap)
		t.Errorf("Received %+v", retrievedMap)
	}
}

// ============================================================================
// Testing Aggregate creation and retrieval
// ============================================================================
func (s *StorageEngineTestSuite) createNewAggregate(t *testing.T) {
	ctx := context.Background()

	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}

	id, err := s.iut.NewAggregate(tx, ctx, userAggregateTypeId)
	if err != nil {
		t.Errorf("Failed to create aggregate without key: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}
	s.aggregateNoKey = id
}

func (s *StorageEngineTestSuite) getAggregate(t *testing.T) {
	ctx := context.Background()

	id, key, err := s.iut.GetAggregateById(ctx, userAggregateTypeId, s.aggregateNoKey)
	if err != nil {
		t.Errorf("Failed to get aggregate by id %d: %v", s.aggregateNoKey, err)
	}

	if key != nil {
		t.Errorf("Aggregate key should be nil for id %d got: %s", s.aggregateNoKey, *key)
	}

	if id != s.aggregateNoKey {
		t.Errorf("Aggregate id received is incorrect - expected %d got: %d", s.aggregateNoKey, id)
	}
}

func (s *StorageEngineTestSuite) getAggregateMissingId(t *testing.T) {
	ctx := context.Background()
	const missingId int64 = 99999999

	id, key, err := s.iut.GetAggregateById(ctx, userAggregateTypeId, missingId)

	if err == nil {
		t.Error("Retrieving aggregate missing an id should an error")
	}

	if id != 0 {
		t.Errorf("Retrieving aggregate missing an id should return 0 for the id got: %d", id)
	}

	if key != nil {
		t.Errorf("Retrieving aggregate missing an id should return nil for the key got: %s", *key)
	}
}

func (s *StorageEngineTestSuite) getAggregateIncorrectType(t *testing.T) {
	ctx := context.Background()
	id, key, err := s.iut.GetAggregateById(ctx, profileAggregateTypeId, s.aggregateNoKey)

	if err == nil {
		t.Error("Retrieving aggregate with incorrect aggregate type should an error")
	}

	if id != 0 {
		t.Errorf("Retrieving aggregate with incorrect aggregate type should return 0 for the id got: %d", id)
	}

	if key != nil {
		t.Errorf("Retrieving aggregate with incorrect aggregate type should return nil for the key got: %s", *key)
	}
}

func (s *StorageEngineTestSuite) createAggregateWithKey(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}

	s.aggregateWithKey, err = s.iut.NewAggregateWithKey(tx, ctx, userAggregateTypeId, "chavez")
	if err != nil {
		t.Errorf("Failed to create aggregate with key: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}
}

func (s *StorageEngineTestSuite) createAggregateWithDuplicateKey(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}

	id, err := s.iut.NewAggregateWithKey(tx, ctx, userAggregateTypeId, "chavez")
	if err == nil {
		t.Errorf("Creating an aggregate with an existing key for the same type should return an error.")
	}
	if id != 0 {
		t.Errorf("Creating an aggregate with an existing key for the same type should return id of 0, got %d", id)
	}

}

// t.Run("Creting an aggregate with the maximum key length", s.createAggregateWithMaximumKeyLength)
// t.Run("Creting an aggregate exceeding the maximum key length", s.createAggregateExceedingMaximumKeyLength)
func (s *StorageEngineTestSuite) createAggregateWithMaximumKeyLength(t *testing.T) {
	ctx := context.Background()
	maxKeyLength := s.iut.GetMaxKeyLength()
	maxKey := strings.Repeat("K", maxKeyLength)

	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}

	id, err := s.iut.NewAggregateWithKey(tx, ctx, userAggregateTypeId, maxKey)
	if err != nil {
		t.Errorf("Failed create an aggregate with the maximum key length.")
	}

	if id <= 0 {
		t.Errorf("Got invalid aggregate id: %d", id)
	}

	err = tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}
}

func (s *StorageEngineTestSuite) createAggregateExceedingMaximumKeyLength(t *testing.T) {
	ctx := context.Background()
	maxKeyLength := s.iut.GetMaxKeyLength()
	maxKey := strings.Repeat("K", maxKeyLength+1)

	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}

	id, err := s.iut.NewAggregateWithKey(tx, ctx, userAggregateTypeId, maxKey)

	if !errors.Is(err, evercore.ErrorKeyExceedsMaximumLength) {
		t.Errorf("Received incorrect error for maximum key length. Expected %v got %v",
			evercore.ErrorKeyExceedsMaximumLength, err)
	}

	if id != 0 {
		t.Errorf("Id expected to be 0 got: %d", id)
	}

	err = tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}
}

func (s *StorageEngineTestSuite) getKeyedAggregateById(t *testing.T) {
	ctx := context.Background()
	id, key, err := s.iut.GetAggregateById(ctx, userAggregateTypeId, s.aggregateWithKey)
	if err != nil {
		t.Errorf("Failed to get aggregate by id %d: %v", s.aggregateWithKey, err)
	}

	if *key != "chavez" {
		t.Errorf("Aggregate key should be 'chavez' for id %d got '%s'", s.aggregateWithKey, *key)
	}

	if id != s.aggregateWithKey {
		t.Errorf("Aggregate id received is incorrect - expected %d got: %d", s.aggregateWithKey, id)
	}
}

func (s *StorageEngineTestSuite) getKeyedAggregateByKey(t *testing.T) {
	ctx := context.Background()
	id, err := s.iut.GetAggregateByKey(ctx, userAggregateTypeId, "chavez")
	if err != nil {
		t.Errorf("Failed to get aggregate by id %d: %v", s.aggregateWithKey, err)
	}

	if id != s.aggregateWithKey {
		t.Errorf("Aggregate id received is incorrect - expected %d got: %d", s.aggregateWithKey, id)
	}
}
func (s *StorageEngineTestSuite) getAggregateByMissingKey(t *testing.T) {
	ctx := context.Background()
	id, err := s.iut.GetAggregateByKey(ctx, userAggregateTypeId, "jjjkkkooo")
	if err == nil {
		t.Errorf("Failed expedted error when retriving aggregate by missing key.")
	}

	if id != 0 {
		t.Errorf("Retrieving aggregate with missing key should return 0 for the id got: %d", id)
	}

}
func (s *StorageEngineTestSuite) getKeyedAggregateWithWrongAggregateType(t *testing.T) {
	ctx := context.Background()
	id, err := s.iut.GetAggregateByKey(ctx, profileAggregateTypeId, "chavez")

	if err == nil {
		t.Error("Retrieving aggregate with incorrect aggregate type should an error")
	}

	if id != 0 {
		t.Errorf("Retrieving aggregate with incorrect aggregate type should return 0 for the id got: %d", id)
	}
}

func (s *StorageEngineTestSuite) writeState(t *testing.T) {
	ctx := context.Background()

	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}

	storageEvents := make([]evercore.StorageEngineEvent, 0, 2)
	storageEvents = append(storageEvents, evercore.StorageEngineEvent{
		AggregateID: s.aggregateNoKey,
		Sequence:    1,
		EventTypeID: userCreatedEventId,
		State:       "{action: \"created\"}",
		EventTime:   time.Now().UTC().Truncate(1000),
		Reference:   "test_suite",
	})

	storageEvents = append(storageEvents, evercore.StorageEngineEvent{
		AggregateID: s.aggregateNoKey,
		Sequence:    2,
		EventTypeID: userUpdatedEventId,
		State:       "{action: \"updated\"}",
		EventTime:   time.Now().UTC().Truncate(1000),
		Reference:   "test_suite",
	})
	s.capturedStorageEvents = storageEvents

	// Make the event retrieval view of the events.
	for _, storageEvent := range storageEvents {
		event := evercore.SerializedEvent{
			AggregateId: storageEvent.AggregateID,
			Sequence:    storageEvent.Sequence,
			EventType:   s.eventTypeMapInv[storageEvent.EventTypeID],
			State:       storageEvent.State,
			Reference:   storageEvent.Reference,
			EventTime:   storageEvent.EventTime,
		}
		s.capturedEvents = append(s.capturedEvents, event)
	}

	snapshots := make([]evercore.Snapshot, 0, 2)
	snapshots = append(snapshots, evercore.Snapshot{
		AggregateId: s.aggregateNoKey,
		State:       "{initial}",
		Sequence:    1,
	})
	snapshots = append(snapshots, evercore.Snapshot{
		AggregateId: s.aggregateNoKey,
		State:       "{initial}",
		Sequence:    2,
	})
	s.capturedSnapshots = snapshots

	err = s.iut.WriteState(tx, ctx, storageEvents, snapshots)
	if err != nil {
		t.Errorf("Write state failed: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		t.Errorf("Failed to commit transaction: %v", err)
	}
}

func (s *StorageEngineTestSuite) getExistingSnapshot(t *testing.T) {
	ctx := context.Background()
	snapshot, err := s.iut.GetSnapshotForAggregate(ctx, s.aggregateNoKey)
	if err != nil {
		t.Errorf("Failed to get snapshot for aggregate: %v", err)
	}

	// We should always get the latest snapshot
	if !reflect.DeepEqual(s.capturedSnapshots[1], *snapshot) {
		t.Errorf("Received incorrect snapshot")
		t.Errorf("Expected: %+v", s.capturedSnapshots[1])
		t.Errorf("Got: %+v", snapshot)
	}
}

func (s *StorageEngineTestSuite) getMissingSnapshot(t *testing.T) {
	ctx := context.Background()
	snapshot, err := s.iut.GetSnapshotForAggregate(ctx, s.aggregateWithKey)
	if err != nil {
		t.Errorf("Failed to get snapshot for aggregate: %v", err)
	}

	if snapshot != nil {
		t.Errorf("Expected nil snapshot instead got: %+v", snapshot)
	}
}

func (s *StorageEngineTestSuite) getEvents(t *testing.T) {
	ctx := context.Background()
	events, err := s.iut.GetEventsForAggregate(ctx, s.aggregateNoKey, 0)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if !reflect.DeepEqual(s.capturedEvents, events) {
		t.Errorf("Incorrect events received")
		t.Errorf("Expected: %+v", s.capturedEvents)
		t.Errorf("Got: %+v", events)
	}
}

func (s *StorageEngineTestSuite) getEventsAfterSequence(t *testing.T) {
	ctx := context.Background()
	events, err := s.iut.GetEventsForAggregate(ctx, s.aggregateNoKey, 1)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	expectedEvents := s.capturedEvents[1:]

	if !reflect.DeepEqual(expectedEvents, events) {
		t.Errorf("Incorrect events received")
		t.Errorf("Expected: %+v", s.capturedEvents)
		t.Errorf("Got: %+v", events)
	}
}

func (s *StorageEngineTestSuite) getEventsAfterLastSequence(t *testing.T) {
	ctx := context.Background()
	events, err := s.iut.GetEventsForAggregate(ctx, s.aggregateNoKey, 2)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) > 0 {
		t.Errorf("Expected no events got: %+v", events)
	}
}

func (s *StorageEngineTestSuite) getEventsForAggregateWithNoEvents(t *testing.T) {
	ctx := context.Background()
	events, err := s.iut.GetEventsForAggregate(ctx, s.aggregateWithKey, 0)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) > 0 {
		t.Errorf("Expected no events got: %+v", events)
	}
}
