package evercore

import (
	"context"
	"testing"
)

const (
	AggregateTypeUser    = "user"
	AggregateTypeProfile = "profile"
)

const (
	EventTypeCreated = "created"
	EventTypeUpdated = "updated"
)

func TestEventStore(t *testing.T) {
	storage := NewMemoryStorageEngine()
	eventStore := NewEventStore(storage)
	testScenarios := NewEventStoreTestScenarios(eventStore)
	testScenarios.RunScenarios(t)
}

type EventStoreTestScenarios struct {
	iut          *EventStore
	newProfileId int64
}

func NewEventStoreTestScenarios(iut *EventStore) EventStoreTestScenarios {
	return EventStoreTestScenarios{
		iut: iut,
	}
}

func (s *EventStoreTestScenarios) RunScenarios(t *testing.T) {
	t.Run("", s.CreateProfileAggregate)
	t.Run("", s.CreateUserAggregate)
}

func (s *EventStoreTestScenarios) CreateProfileAggregate(t *testing.T) {
	ctx := context.Background()
	err := s.iut.WithContext(ctx, func(etx EventStoreContext) error {
		id, err := etx.NewAggregateId(AggregateTypeProfile)
		if err != nil {
			t.Errorf("Failed to create aggregate: %v", err)
			return err
		}
		s.newProfileId = id

		ev := SerializedEvent{
			AggregateId: id,
			EventType:   EventTypeCreated,
			State:       "{}",
			Reference:   "{}",
		}
		etx.Publish(&ev)

		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func (s *EventStoreTestScenarios) CreateUserAggregate(t *testing.T) {
	ctx := context.Background()
	err := s.iut.WithContext(ctx, func(etx EventStoreContext) error {
		id, err := etx.NewAggregateIdWithKey(AggregateTypeUser, "chavez@example.com")
		if err != nil {
			t.Errorf("Failed to create aggregate: %v", err)
			return err
		}
		s.newProfileId = id

		ev := SerializedEvent{
			AggregateId: id,
			EventType:   EventTypeCreated,
			State:       "{}",
			Reference:   "{}",
		}
		etx.Publish(&ev)
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}
