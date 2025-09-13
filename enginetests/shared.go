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

	t.Run("Get or create aggregate by key", s.testGetOrCreateAggregateByKey)
	t.Run("Get or create aggregate with max key length", s.testGetOrCreateAggregateWithMaxKeyLength)

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
	t.Run("Changing aggregate natural key", s.testChangeAggregateNaturalKey)
	t.Run("Changing aggregate natural key with new key being too long", s.testChangeAggregateNaturalKey_WithNewKeyBeingTooLong)

	// Durable subscription queries
	t.Run("Upsert and get subscription", s.testUpsertAndGetSubscription)
	t.Run("Subscription lease lifecycle", s.testSubscriptionLeaseLifecycle)
	t.Run("Subscription single-type filtering", s.testGetEventsForSubscription_SingleEventType)
	t.Run("Advance subscription cursor", s.testAdvanceSubscriptionCursor)
	t.Run("Subscription multi-type filtering", s.testGetEventsForSubscription_MultiEventTypes)
	t.Run("Subscription aggregate key filtering", s.testGetEventsForSubscription_AggregateKey)
	t.Run("Max and first event id helpers", s.testGetMaxAndFirstEventId)

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
	tx, err := s.iut.GetTransactionInfo()
	if err != nil {
		t.Errorf("Failed to get transaction info: %v", err)
		return
	}

	defer tx.Commit()

	userAggregateTypeId, err = s.iut.GetAggregateTypeId(tx, ctx, userAggregateType)
	if err != nil {
		t.Errorf("Failed to get aggregate type: %v", err)
		return
	}

	profileAggregateTypeId, err = s.iut.GetAggregateTypeId(tx, ctx, profileAggregateType)
	if err != nil {
		t.Errorf("Failed to get aggregate type: %v", err)
		return
	}

	s.aggregateTypeMap[userAggregateType] = userAggregateTypeId
	s.aggregateTypeMap[profileAggregateType] = profileAggregateTypeId
}

func (s *StorageEngineTestSuite) canGetExistingAggregateType(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	retrievedUserAggregateId, err := s.iut.GetAggregateTypeId(tx, ctx, userAggregateType)
	if err != nil {
		t.Errorf("Failed to get aggregate type: %v", err)
	}

	if retrievedUserAggregateId != userAggregateTypeId {
		t.Errorf("Retrieved aggregate type id does not match expected: %d != %d", retrievedUserAggregateId, userAggregateTypeId)
	}
}

func (s *StorageEngineTestSuite) getAllAggregateTypes(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	retrievedAggregateTypes, err := s.iut.GetAggregateTypes(tx, ctx)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Commit()

	userCreatedEventId, err = s.iut.GetEventTypeId(tx, ctx, userCreatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", userCreatedEvent, err)
	}

	userUpdatedEventId, err = s.iut.GetEventTypeId(tx, ctx, userUpdatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", userUpdatedEvent, err)
	}

	profileCreatedEventId, err = s.iut.GetEventTypeId(tx, ctx, profileCreatedEvent)
	if err != nil {
		t.Errorf("Failed to get event type id for '%s': %v", profileCreatedEvent, err)
	}

	profileUpdatedEventId, err = s.iut.GetEventTypeId(tx, ctx, profileUpdatedEvent)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	retrievedEventId, err := s.iut.GetEventTypeId(tx, ctx, profileCreatedEvent)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	retrievedEvents, err := s.iut.GetEventTypes(tx, ctx)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	id, key, err := s.iut.GetAggregateById(tx, ctx, userAggregateTypeId, s.aggregateNoKey)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	id, key, err := s.iut.GetAggregateById(tx, ctx, userAggregateTypeId, missingId)

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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	id, key, err := s.iut.GetAggregateById(tx, ctx, profileAggregateTypeId, s.aggregateNoKey)

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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	id, key, err := s.iut.GetAggregateById(tx, ctx, userAggregateTypeId, s.aggregateWithKey)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	id, err := s.iut.GetAggregateByKey(tx, ctx, userAggregateTypeId, "chavez")
	if err != nil {
		t.Errorf("Failed to get aggregate by id %d: %v", s.aggregateWithKey, err)
	}

	if id != s.aggregateWithKey {
		t.Errorf("Aggregate id received is incorrect - expected %d got: %d", s.aggregateWithKey, id)
	}
}
func (s *StorageEngineTestSuite) getAggregateByMissingKey(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	id, err := s.iut.GetAggregateByKey(tx, ctx, userAggregateTypeId, "jjjkkkooo")
	if err == nil {
		t.Errorf("Failed expedted error when retriving aggregate by missing key.")
	}

	if id != 0 {
		t.Errorf("Retrieving aggregate with missing key should return 0 for the id got: %d", id)
	}

}
func (s *StorageEngineTestSuite) getKeyedAggregateWithWrongAggregateType(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()
	id, err := s.iut.GetAggregateByKey(tx, ctx, profileAggregateTypeId, "chavez")

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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	snapshot, err := s.iut.GetSnapshotForAggregate(tx, ctx, s.aggregateNoKey)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	snapshot, err := s.iut.GetSnapshotForAggregate(tx, ctx, s.aggregateWithKey)
	if err != nil {
		t.Errorf("Failed to get snapshot for aggregate: %v", err)
	}

	if snapshot != nil {
		t.Errorf("Expected nil snapshot instead got: %+v", snapshot)
	}
}

func (s *StorageEngineTestSuite) getEvents(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()
	events, err := s.iut.GetEventsForAggregate(tx, ctx, s.aggregateNoKey, 0)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	events, err := s.iut.GetEventsForAggregate(tx, ctx, s.aggregateNoKey, 1)
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
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	events, err := s.iut.GetEventsForAggregate(tx, ctx, s.aggregateNoKey, 2)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) > 0 {
		t.Errorf("Expected no events got: %+v", events)
	}
}

func (s *StorageEngineTestSuite) getEventsForAggregateWithNoEvents(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	defer tx.Rollback()

	events, err := s.iut.GetEventsForAggregate(tx, ctx, s.aggregateWithKey, 0)
	if err != nil {
		t.Errorf("Failed to get events: %v", err)
	}

	if len(events) > 0 {
		t.Errorf("Expected no events got: %+v", events)
	}
}

func (s *StorageEngineTestSuite) testGetOrCreateAggregateWithMaxKeyLength(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}
	defer tx.Rollback()

	// Test max key length
	maxKey := strings.Repeat("L", s.iut.GetMaxKeyLength())
	created, aggregateId, err := s.iut.GetOrCreateAggregateByKey(tx, ctx, userAggregateTypeId, maxKey)
	if err != nil {
		t.Errorf("Failed with max length key: %v", err)
	}
	if !created {
		t.Error("Expected new aggregate to be created")
	}
	if aggregateId <= 0 {
		t.Errorf("Invalid aggregate id returned: %d", aggregateId)
	}
}

func (s *StorageEngineTestSuite) testGetOrCreateAggregateByKey(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}
	defer tx.Rollback()

	// Test creating new aggregate
	newKey := "test_key_" + time.Now().Format(time.RFC3339Nano)
	created, aggregateId, err := s.iut.GetOrCreateAggregateByKey(tx, ctx, userAggregateTypeId, newKey)
	if err != nil {
		t.Errorf("Failed to create aggregate with key: %v", err)
	}
	if !created {
		t.Error("Expected new aggregate to be created")
	}
	if aggregateId <= 0 {
		t.Errorf("Invalid aggregate id returned: %d", aggregateId)
	}

	// Test getting existing aggregate
	created, existingId, err := s.iut.GetOrCreateAggregateByKey(tx, ctx, userAggregateTypeId, newKey)
	if err != nil {
		t.Errorf("Failed to get existing aggregate: %v", err)
	}
	if created {
		t.Error("Expected existing aggregate to be returned, not created")
	}
	if existingId != aggregateId {
		t.Errorf("Existing aggregate id mismatch: expected %d got %d", aggregateId, existingId)
	}

	// Test wrong aggregate type
	_, _, err = s.iut.GetOrCreateAggregateByKey(tx, ctx, profileAggregateTypeId, newKey)
	if err == nil {
		t.Error("Expected error when using wrong aggregate type")
	}

	// Test exceeding max key length
	tooLongKey := strings.Repeat("J", s.iut.GetMaxKeyLength()+1)
	_, _, err = s.iut.GetOrCreateAggregateByKey(tx, ctx, userAggregateTypeId, tooLongKey)
	if !errors.Is(err, evercore.ErrorKeyExceedsMaximumLength) {
		t.Errorf("Expected ErrorKeyExceedsMaximumLength, got: %v", err)
	}
}

func (s *StorageEngineTestSuite) testChangeAggregateNaturalKey(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}
	defer tx.Rollback()

	userAggregateId := s.aggregateNoKey
	newKey := "test_key_" + time.Now().Format(time.RFC3339Nano)

	err = s.iut.ChangeAggregateNaturalKey(tx, ctx, userAggregateId, newKey)
	if err != nil {
		t.Errorf("Failed to change aggregate natural key: %v", err)
	}

	// Test getting existing aggregate
	existing, err := s.iut.GetAggregateByKey(tx, ctx, userAggregateTypeId, newKey)

	if err != nil {
		t.Errorf("Failed to get existing aggregate: %v", err)
	}

	if existing != userAggregateId {
		t.Errorf("Existing aggregate id mismatch: expected %d got %d", userAggregateId, existing)
	}

}

func (s *StorageEngineTestSuite) testChangeAggregateNaturalKey_WithNewKeyBeingTooLong(t *testing.T) {
	ctx := context.Background()
	tx, err := s.iut.GetTransactionInfo()
	if err != nil {
		t.Errorf("Failed to create transaction: %v", err)
	}
	defer tx.Rollback()

	userAggregateId := s.aggregateWithKey
	tooLongKey := strings.Repeat("M", s.iut.GetMaxKeyLength()+1)

	err = s.iut.ChangeAggregateNaturalKey(tx, ctx, userAggregateId, tooLongKey)
	if !errors.Is(err, evercore.ErrorKeyExceedsMaximumLength) {
		t.Errorf("Expected ErrorKeyExceedsMaximumLength, got: %v", err)
	}
}

// ============================================================================
// Durable Subscription Query Tests
// ============================================================================

// Helper to write events for the keyed aggregate without affecting earlier expectations
func (s *StorageEngineTestSuite) writeStateForKeyedAggregate(t *testing.T) {
    ctx := context.Background()

    tx, err := s.iut.GetTransactionInfo()
    if err != nil {
        t.Errorf("Failed to create transaction: %v", err)
        return
    }
    defer tx.Rollback()

    events := []evercore.StorageEngineEvent{
        {
            AggregateID: s.aggregateWithKey,
            Sequence:    1,
            EventTypeID: userCreatedEventId,
            State:       "{action: \"created\"}",
            EventTime:   time.Now().UTC(),
            Reference:   "test_suite_keyed",
        },
        {
            AggregateID: s.aggregateWithKey,
            Sequence:    2,
            EventTypeID: userUpdatedEventId,
            State:       "{action: \"updated\"}",
            EventTime:   time.Now().UTC(),
            Reference:   "test_suite_keyed",
        },
    }

    if err := s.iut.WriteState(tx, ctx, events, nil); err != nil {
        t.Errorf("Write state for keyed aggregate failed: %v", err)
        return
    }

    if err := tx.Commit(); err != nil {
        t.Errorf("Failed to commit transaction: %v", err)
    }
}

func (s *StorageEngineTestSuite) testUpsertAndGetSubscription(t *testing.T) {
    ctx := context.Background()

    name := "enginetest_sub_single"
    subID, err := s.iut.UpsertSubscription(nil, ctx, name, &userAggregateTypeId, &userCreatedEventId, nil, evercore.StartBeginning, 0, nil)
    if err != nil {
        t.Errorf("UpsertSubscription failed: %v", err)
        return
    }

    sub, err := s.iut.GetSubscriptionByName(nil, ctx, name)
    if err != nil {
        t.Errorf("GetSubscriptionByName failed: %v", err)
        return
    }
    if sub == nil {
        t.Errorf("Expected subscription, got nil")
        return
    }
    if sub.ID != subID {
        t.Errorf("Subscription id mismatch: expected %d got %d", subID, sub.ID)
    }
    if sub.Name != name {
        t.Errorf("Subscription name mismatch: expected %s got %s", name, sub.Name)
    }
    if sub.AggregateTypeID == nil || *sub.AggregateTypeID != userAggregateTypeId {
        t.Errorf("AggregateTypeID mismatch or nil")
    }
    if sub.EventTypeID == nil || *sub.EventTypeID != userCreatedEventId {
        t.Errorf("EventTypeID mismatch or nil")
    }
    if !sub.Active {
        t.Errorf("Expected subscription to be active")
    }
    if sub.StartFrom != evercore.StartBeginning {
        t.Errorf("StartFrom mismatch: expected %s got %s", evercore.StartBeginning, sub.StartFrom)
    }
}

func (s *StorageEngineTestSuite) testSubscriptionLeaseLifecycle(t *testing.T) {
    ctx := context.Background()
    name := "enginetest_sub_lease"

    // Create a simple subscription with no filters
    if _, err := s.iut.UpsertSubscription(nil, ctx, name, nil, nil, nil, evercore.StartBeginning, 0, nil); err != nil {
        t.Errorf("UpsertSubscription failed: %v", err)
        return
    }

    owner1 := "owner1"
    owner2 := "owner2"

    claimed, err := s.iut.ClaimSubscription(nil, ctx, name, owner1, 2*time.Second)
    if err != nil {
        t.Errorf("ClaimSubscription failed: %v", err)
        return
    }
    if !claimed {
        t.Errorf("Expected claim to succeed for owner1")
    }

    // Another owner should not be able to claim
    claimed, err = s.iut.ClaimSubscription(nil, ctx, name, owner2, 2*time.Second)
    if err != nil {
        t.Errorf("ClaimSubscription (owner2) failed: %v", err)
        return
    }
    if claimed {
        t.Errorf("Expected claim to fail for owner2 while leased")
    }

    // Renew with wrong owner should return false
    renewed, err := s.iut.RenewSubscription(nil, ctx, name, owner2, 2*time.Second)
    if err != nil {
        t.Errorf("RenewSubscription (owner2) error: %v", err)
        return
    }
    if renewed {
        t.Errorf("Expected renew to fail for non-owner")
    }

    // Renew with correct owner should succeed
    renewed, err = s.iut.RenewSubscription(nil, ctx, name, owner1, 2*time.Second)
    if err != nil {
        t.Errorf("RenewSubscription (owner1) error: %v", err)
        return
    }
    if !renewed {
        t.Errorf("Expected renew to succeed for owner1")
    }

    // Release and ensure another claim works
    if err := s.iut.ReleaseSubscription(nil, ctx, name, owner1); err != nil {
        t.Errorf("ReleaseSubscription error: %v", err)
        return
    }
    claimed, err = s.iut.ClaimSubscription(nil, ctx, name, owner2, 2*time.Second)
    if err != nil {
        t.Errorf("ClaimSubscription (owner2 after release) failed: %v", err)
        return
    }
    if !claimed {
        t.Errorf("Expected owner2 to claim after release")
    }
    _ = s.iut.ReleaseSubscription(nil, ctx, name, owner2)
}

func (s *StorageEngineTestSuite) testGetEventsForSubscription_SingleEventType(t *testing.T) {
    ctx := context.Background()

    name := "enginetest_sub_single"
    sub, err := s.iut.GetSubscriptionByName(nil, ctx, name)
    if err != nil {
        t.Errorf("GetSubscriptionByName failed: %v", err)
        return
    }

    events, err := s.iut.GetEventsForSubscription(nil, ctx, sub, 100)
    if err != nil {
        t.Errorf("GetEventsForSubscription failed: %v", err)
        return
    }

    // We expect only the userCreatedEvent for the aggregate without key
    expectedType := s.eventTypeMapInv[userCreatedEventId]
    foundOther := false
    for _, e := range events {
        if e.EventType != expectedType {
            foundOther = true
            break
        }
    }
    if foundOther {
        t.Errorf("Expected only events of type %s, got mixed types: %+v", expectedType, events)
    }
    if len(events) == 0 {
        t.Errorf("Expected at least one event for single-type subscription")
    }
}

func (s *StorageEngineTestSuite) testAdvanceSubscriptionCursor(t *testing.T) {
    ctx := context.Background()
    name := "enginetest_sub_single"
    sub, err := s.iut.GetSubscriptionByName(nil, ctx, name)
    if err != nil {
        t.Errorf("GetSubscriptionByName failed: %v", err)
        return
    }

    events, err := s.iut.GetEventsForSubscription(nil, ctx, sub, 100)
    if err != nil {
        t.Errorf("GetEventsForSubscription failed: %v", err)
        return
    }
    if len(events) == 0 {
        t.Skip("No events to advance cursor with; previous step likely failed")
        return
    }

    last := events[len(events)-1].EventID
    if err := s.iut.AdvanceSubscriptionCursor(nil, ctx, sub.ID, last); err != nil {
        t.Errorf("AdvanceSubscriptionCursor failed: %v", err)
        return
    }

    // After advancing, no further events for that type should be returned immediately
    sub, _ = s.iut.GetSubscriptionByName(nil, ctx, name)
    next, err := s.iut.GetEventsForSubscription(nil, ctx, sub, 100)
    if err != nil {
        t.Errorf("GetEventsForSubscription after advance failed: %v", err)
        return
    }
    if len(next) != 0 {
        t.Errorf("Expected zero events after advancing cursor, got %d", len(next))
    }
}

func (s *StorageEngineTestSuite) testGetEventsForSubscription_MultiEventTypes(t *testing.T) {
    ctx := context.Background()
    name := "enginetest_sub_multi"

    // Create multi-type subscription (use join table)
    subID, err := s.iut.UpsertSubscription(nil, ctx, name, &userAggregateTypeId, nil, nil, evercore.StartBeginning, 0, nil)
    if err != nil {
        t.Errorf("UpsertSubscription failed: %v", err)
        return
    }
    if err := s.iut.AddSubscriptionEventType(nil, ctx, subID, userCreatedEventId); err != nil {
        t.Errorf("AddSubscriptionEventType (created) failed: %v", err)
        return
    }
    if err := s.iut.AddSubscriptionEventType(nil, ctx, subID, userUpdatedEventId); err != nil {
        t.Errorf("AddSubscriptionEventType (updated) failed: %v", err)
        return
    }

    sub, err := s.iut.GetSubscriptionByName(nil, ctx, name)
    if err != nil {
        t.Errorf("GetSubscriptionByName failed: %v", err)
        return
    }

    events, err := s.iut.GetEventsForSubscription(nil, ctx, sub, 100)
    if err != nil {
        t.Errorf("GetEventsForSubscription failed: %v", err)
        return
    }

    // Expect both types to appear
    want := map[string]bool{
        s.eventTypeMapInv[userCreatedEventId]: true,
        s.eventTypeMapInv[userUpdatedEventId]: true,
    }
    got := map[string]bool{}
    for _, e := range events {
        got[e.EventType] = true
    }
    for k := range want {
        if !got[k] {
            t.Errorf("Expected event type %s in multi-type results; got %+v", k, events)
        }
    }
}

func (s *StorageEngineTestSuite) testGetEventsForSubscription_AggregateKey(t *testing.T) {
    // Ensure events exist for keyed aggregate
    s.writeStateForKeyedAggregate(t)

    ctx := context.Background()
    name := "enginetest_sub_key"
    key := "chavez"

    // Filter by aggregate key only
    if _, err := s.iut.UpsertSubscription(nil, ctx, name, nil, nil, &key, evercore.StartBeginning, 0, nil); err != nil {
        t.Errorf("UpsertSubscription failed: %v", err)
        return
    }
    sub, err := s.iut.GetSubscriptionByName(nil, ctx, name)
    if err != nil {
        t.Errorf("GetSubscriptionByName failed: %v", err)
        return
    }

    events, err := s.iut.GetEventsForSubscription(nil, ctx, sub, 100)
    if err != nil {
        t.Errorf("GetEventsForSubscription failed: %v", err)
        return
    }
    if len(events) < 2 {
        t.Errorf("Expected at least two events for keyed aggregate, got %d", len(events))
    }
    for _, e := range events {
        if e.AggregateId != s.aggregateWithKey {
            t.Errorf("Expected only events for aggregate %d, got %+v", s.aggregateWithKey, e)
        }
    }
}

func (s *StorageEngineTestSuite) testGetMaxAndFirstEventId(t *testing.T) {
    ctx := context.Background()
    name := "enginetest_sub_all"

    if _, err := s.iut.UpsertSubscription(nil, ctx, name, nil, nil, nil, evercore.StartBeginning, 0, nil); err != nil {
        t.Errorf("UpsertSubscription failed: %v", err)
        return
    }
    sub, err := s.iut.GetSubscriptionByName(nil, ctx, name)
    if err != nil {
        t.Errorf("GetSubscriptionByName failed: %v", err)
        return
    }

    events, err := s.iut.GetEventsForSubscription(nil, ctx, sub, 100)
    if err != nil {
        t.Errorf("GetEventsForSubscription failed: %v", err)
        return
    }
    if len(events) == 0 {
        t.Errorf("Expected events for max/first id checks")
        return
    }

    maxID := int64(0)
    firstTime := events[0].EventTime
    firstIDAtTime := events[0].EventID
    for _, e := range events {
        if e.EventID > maxID {
            maxID = e.EventID
        }
        if e.EventTime.Before(firstTime) || e.EventTime.Equal(firstTime) && e.EventID < firstIDAtTime {
            firstTime = e.EventTime
            firstIDAtTime = e.EventID
        }
    }

    gotMax, err := s.iut.GetMaxEventId(nil, ctx)
    if err != nil {
        t.Errorf("GetMaxEventId failed: %v", err)
        return
    }
    if gotMax != maxID {
        t.Errorf("GetMaxEventId mismatch: expected %d got %d", maxID, gotMax)
    }

    gotFirst, err := s.iut.GetFirstEventIdFromTimestamp(nil, ctx, firstTime)
    if err != nil {
        t.Errorf("GetFirstEventIdFromTimestamp failed: %v", err)
        return
    }
    if gotFirst != firstIDAtTime {
        t.Errorf("GetFirstEventIdFromTimestamp mismatch: expected %d got %d", firstIDAtTime, gotFirst)
    }
}
