package evercore

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

type SampleState struct {
	FirstName string
	LastName  string
}

type SampleCreatedEvent struct {
	FirstName *string
	LastName  *string
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
	event := SampleCreatedEvent{
		FirstName: &firstName,
		LastName:  &lastName,
	}
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
	event := SampleCreatedEvent{
		FirstName: &firstName,
		LastName:  nil,
	}
	aggregate.ApplyEventState(event, time.Now(), "")
	if aggregate.State.FirstName != firstName {
		t.Errorf("Expected name to be %s, got %s", firstName, aggregate.State.FirstName)
	}
	if aggregate.State.LastName != "Harvey" {
		t.Errorf("Expected name to be %s, got %s", "Harvey", aggregate.State.LastName)
	}
}
