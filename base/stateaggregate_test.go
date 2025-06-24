package evercore

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type SampleState struct {
	FirstName string
	LastName  string
	Active    bool
}

type SampleCreatedEvent struct {
	FirstName *string
	LastName  *string
}

type SetActiveEvent struct {
	Active bool
}

func NewActiveEvent(active bool) EventState {
	return SetActiveEvent{Active: active}
}

func (t SetActiveEvent) GetEventType() string {
	return "SetActiveEvent"
}

func (t SetActiveEvent) Serialize() string {
	return SerializeEvent(t)
}

func SerializeEvent(ev any) string {
	bytes, err := json.Marshal(ev)
	if err != nil {
		panicState := fmt.Errorf("State failed to serialize: %v", err)
		panic(panicState)
	}
	stringly := string(bytes)
	return stringly
}

func (t SampleCreatedEvent) GetEventType() string {
	return "SampleCreatedEvent"
}

func (t SampleCreatedEvent) Serialize() string {
	return SerializeEvent(t)
}

// Embedding StateAggregate in a struct allows us to use reflection to copy fields from the event state to the struct.
type SampleStateAggregate struct {
	StateAggregate[SampleState]
}

func TestStateAggregateImplementsAggregateInterface(t *testing.T) {
	stateAggregate := SampleStateAggregate{}
	var _ Aggregate = &stateAggregate
}

func TestApplyEventStateSetsFields(t *testing.T) {

	aggregate := SampleStateAggregate{}
	var firstName = "John"
	var lastName = "Smith"
	event := NewStateEvent(SampleCreatedEvent{
		FirstName: &firstName,
		LastName:  &lastName,
	})
	aggregate.ApplyEventState(event, time.Now(), "")
	if aggregate.State.FirstName != firstName {
		t.Errorf("Expected name to be %s, got %s", firstName, aggregate.State.FirstName)
	}
	if aggregate.State.LastName != lastName {
		t.Errorf("Expected name to be %s, got %s", firstName, aggregate.State.FirstName)
	}
}

func TestApplyEventStateIgnoresNilPointerFields(t *testing.T) {

	aggregate := SampleStateAggregate{}
	aggregate.State.FirstName = "Timothy"
	aggregate.State.LastName = "Harvey"
	var firstName = "John"
	event := NewStateEvent(SampleCreatedEvent{
		FirstName: &firstName,
		LastName:  nil,
	})
	aggregate.ApplyEventState(event, time.Now(), "")
	if aggregate.State.FirstName != firstName {
		t.Errorf("Expected name to be %s, got %s", firstName, aggregate.State.FirstName)
	}
	if aggregate.State.LastName != "Harvey" {
		t.Errorf("Expected name to be %s, got %s", "Harvey", aggregate.State.LastName)
	}
}

func TestEventStoreContextLoadsState(t *testing.T) {
	store := NewEventStore(NewMemoryStorageEngine())
	var firstName = "John"
	var lastName = "Smith"
	state := SampleStateAggregate{}
	err := store.WithContext(context.Background(), func(etx EventStoreContext) error {
		etx.CreateAggregateInto(&state)
		err := etx.LoadStateInto(&state, 1)
		if err != nil {
			return err
		}
		event := NewStateEvent(SampleCreatedEvent{
			FirstName: &firstName,
			LastName:  &lastName,
		})
		etx.ApplyEventTo(&state, event, time.Now().UTC(), "")
		return nil
	})

	if state.State.FirstName != firstName {
		t.Errorf("Expected name to be %s, got %s", firstName, state.State.FirstName)
	}
	if state.State.LastName != lastName {
		t.Errorf("Expected name to be %s, got %s", lastName, state.State.LastName)
	}

	if err != nil {
		t.Errorf("Event store context failed: %v", err)
	}
}

func TestApplyEventStateForNonStateEvents(t *testing.T) {

	aggregate := SampleStateAggregate{}
	aggregate.State.FirstName = "Timothy"
	aggregate.State.LastName = "Harvey"
	aggregate.State.Active = false

	aggregate.ApplyOtherEvents = func(eventState EventState, eventTime time.Time, reference string) error {
		switch event := eventState.(type) {
		case SetActiveEvent:
			aggregate.State.Active = event.Active
			return nil
		default:
			return fmt.Errorf("unknown event type %s", event.GetEventType())
		}
	}

	event := NewActiveEvent(true)

	err := aggregate.ApplyEventState(event, time.Now(), "")
	if err != nil {
		t.Errorf("ApplyEventState failed: %v", err)
	}
	if aggregate.State.Active != true {
		t.Errorf("Expected active to be true, got %v", aggregate.State.Active)
	}

	event = NewActiveEvent(false)
	aggregate.ApplyEventState(event, time.Now(), "")
	if aggregate.State.Active != false {
		t.Errorf("Expected active to be false, got %v", aggregate.State.Active)
	}
}

func TestApplyEventStateForNonStateEventsNoApplyMethod(t *testing.T) {

	aggregate := SampleStateAggregate{}
	aggregate.State.FirstName = "Timothy"
	aggregate.State.LastName = "Harvey"
	aggregate.State.Active = false

	event := NewActiveEvent(true)

	err := aggregate.ApplyEventState(event, time.Now(), "")
	// This should fail.
	if err == nil {
		t.Errorf("ApplyEventState should have failed.")
	}

	if err.Error() != ErrEventStateIsNotAStateEvent.Error() {
		t.Errorf("Expected error message to be %s, got %s", ErrEventStateIsNotAStateEvent.Error(), err.Error())
	}

	if aggregate.State.Active != false {
		t.Errorf("Expected active to be false, got %v", aggregate.State.Active)
	}

}
