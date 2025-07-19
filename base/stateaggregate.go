package evercore

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

func (e ErrorMessage) Error() string {
	return string(e)
}

var ErrEventStateIsNotAStateEvent = ErrorMessage("Event state is not a StateEvent.")

type ApplyFunc func(eventState EventState, eventTime time.Time, reference string) error

// StateAggregate can be used for aggregates that contain simple state.
// It provides common aggregate functionality and handles state management.
//
// HandleOtherEvents is an optional function that can be set to handle events
// that don't implement iStateEvent. When an event is applied that isn't a StateEvent,
// this function will be called if set. If not set, such events will return ErrEventStateIsNotAStateEvent.
//
// Example usage:
//
//	aggregate.HandleOtherEvents = func(eventState EventState, eventTime time.Time, reference string) error {
//	    switch event := eventState.(type) {
//	    case SetActiveEvent:
//	        aggregate.State.Active = event.Active
//	        return nil
//	    default:
//	        return fmt.Errorf("unknown event type %s", event.GetEventType())
//	    }
//	}
type StateAggregate[T any] struct {
	Id            int64
	State         T
	Sequence      int64
	aggregateType string
}

type stateAggregateOtherEvent interface {
	HandleEvent(eventState EventState, eventTime time.Time, reference string) error
}

// EventDecoder is a function that decodes an event into an EventState.
type EventDecoder func(ev SerializedEvent) (EventState, error)

var eventDecoders []EventDecoder

// RegisterEventDecoder sets the EventDecoder for StateAggregate - the function should be able to
// decode all events for any aggregate using StateAggregate.
func RegisterEventDecoder(decoders ...EventDecoder) {
	for _, decoder := range decoders {
		eventDecoders = append(eventDecoders, decoder)
	}
}

// DecodeEventStateTo decodes an event into an EventState.
func DecodeEvent(ev SerializedEvent) (bool, EventState, error) {
	if len(eventDecoders) == 0 {
		panic("No EventDecoders set")
	}

	for _, decoder := range eventDecoders {
		eventState, err := decoder(ev)
		if err != nil {
			return false, nil, err
		}
		if eventState != nil {
			if _, ok := eventState.(iStateEvent); !ok {
				return false, eventState, nil
			}
			return true, eventState, nil
		}
	}
	return false, nil, fmt.Errorf("no decoder could handle event type %s", ev.EventType)
}

// Implement Aggregate interface for StateAggregate
func (t *StateAggregate[T]) GetSequence() int64 {
	return t.Sequence
}

// SetId sets the aggregate id
func (t *StateAggregate[T]) SetId(id int64) {
	t.Id = id
}

// GetId gets the aggregate id
func (t *StateAggregate[T]) GetId() int64 {
	return t.Id
}

// SetSequence sets the aggregate sequence
func (t *StateAggregate[T]) SetSequence(seq int64) {
	t.Sequence = seq
}

// GetAggregateType gets the aggregate type
func (t *StateAggregate[T]) GetAggregateType() string {
	// Check if aggregateType is empty
	if t.aggregateType == "" {
		stateType := reflect.TypeOf(t.State)
		t.aggregateType = stateType.Name()
	}
	return t.aggregateType
}

// GetSnapshotFrequency gets the snapshot frequency
func (t *StateAggregate[T]) GetSnapshotFrequency() int64 {
	return 10
}

type ErrorMessage string

// ApplyEventState applies an event state to the aggregate
func (t *StateAggregate[T]) ApplyEventState(eventState EventState, eventTime time.Time, reference string) error {
	var stateValue any
	value, ok := eventState.(iStateEvent)
	if ok {
		stateValue = value.GetState()
		return CopyFields(stateValue, &t.State)
	}
	return ErrEventStateIsNotAStateEvent
}

// GetSnapshotState gets the snapshot state
func (t *StateAggregate[T]) GetSnapshotState() (*string, error) {
	bytes, err := json.Marshal(t.State)
	if err != nil {
		return nil, err
	}
	stringly := string(bytes)
	return &stringly, nil
}

// ApplySnapshot applies a snapshot to the aggregate
func (t *StateAggregate[T]) ApplySnapshot(snapshot *Snapshot) error {
	var newState T
	err := json.Unmarshal([]byte(snapshot.State), &newState)
	if err != nil {
		return err
	}
	t.State = newState
	return nil
}

// CopyFields uses reflection to copy matching fields from e to t.
// If a pointer field in e is nil, the field is skipped.
func CopyFields(e, t any) error {
	// Get the reflect.Value of e and t.
	// Note: t must be a pointer so that its fields can be set.
	eVal := reflect.ValueOf(e)
	tVal := reflect.ValueOf(t)

	// If e is a pointer, get the underlying value.
	if eVal.Kind() == reflect.Ptr {
		eVal = eVal.Elem()
	}
	// If t is a pointer, get the underlying value.
	if tVal.Kind() == reflect.Ptr {
		tVal = tVal.Elem()
	}

	// Iterate over all fields in e
	for i := range eVal.NumField() {
		// Get the field and its type info from e.
		eField := eVal.Field(i)
		fieldInfo := eVal.Type().Field(i)
		fieldName := fieldInfo.Name

		// Get the corresponding field in t.
		tField := tVal.FieldByName(fieldName)
		if !tField.IsValid() || !tField.CanSet() {
			// Field not found or not settable in t; skip it.
			continue
		}

		// If e's field is a pointer...
		if eField.Kind() == reflect.Ptr {
			// If the pointer is nil, skip copying this field.
			if eField.IsNil() {
				continue
			}
			// Otherwise, use the value the pointer points to.
			eField = eField.Elem()
		}

		// Make sure that the e field value can be assigned to the t field.
		if eField.Type().AssignableTo(tField.Type()) {
			tField.Set(eField)
		} else if eField.Type().ConvertibleTo(tField.Type()) {
			tField.Set(eField.Convert(tField.Type()))
		} else {
			// If the types are not assignable or convertible, you might handle the error here.
			return fmt.Errorf("Cannot assign field %s from %s to %s",
				fieldName, eField.Type(), tField.Type())
		}
	}
	return nil
}

func (t *StateAggregate[T]) DecodeEvent(ev SerializedEvent) (EventState, error) {
	if len(eventDecoders) == 0 {
		panic("No EventDecoders set")
	}

	for _, decoder := range eventDecoders {
		eventState, err := decoder(ev)
		if err != nil {
			return nil, err
		}
		if eventState != nil {
			return eventState, nil
		}
	}
	return nil, fmt.Errorf("no decoder could handle event type %s", ev.EventType)
}
