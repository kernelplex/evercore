package evercore

import (
	"encoding/json"
	"fmt"
	"reflect"
	"time"
)

// StateAggregate can be used for aggregates that contain simple state.
type StateAggregate[T any] struct {
	Id            int64
	State         T
	Sequence      int64
	aggregateType string
}

// EventDecoder is a function that decodes an event into an EventState.
type EventDecoder func(aggregateType string, ev SerializedEvent) (EventState, error)

var eventDecoders []EventDecoder

// RegisterStateEventDecoder sets the EventDecoder for StateAggregate - the function should be able to
// decode all events for any aggregate using StateAggregate.
func RegisterStateEventDecoder(decoder EventDecoder) {
	eventDecoders = append(eventDecoders, decoder)
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

// ApplyEventState applies an event state to the aggregate
func (t *StateAggregate[T]) ApplyEventState(eventState EventState, eventTime time.Time, reference string) error {
	var stateValue any
	value, ok := eventState.(iStateEvent)
	if ok {
		stateValue = value.GetState()
	} else {
		stateValue = eventState
	}
	return CopyFields(stateValue, &t.State)
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
		eventState, err := decoder(t.GetAggregateType(), ev)
		if err != nil {
			return nil, err
		}
		if eventState != nil {
			return eventState, nil
		}
	}
	return nil, fmt.Errorf("no decoder could handle event type %s", ev.EventType)
}
