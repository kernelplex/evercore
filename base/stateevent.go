package evercore

import (
	"encoding/json"
	"reflect"
)

func SerializeToJson(ev any) string {
	serialized, err := json.Marshal(ev)
	if err != nil {
		panic("State failed to serialize.")
	}
	return string(serialized)
}

func DeserializeFromJson[T any](serialized string, target *T) error {
	return json.Unmarshal([]byte(serialized), target)
}

func TypeNameFromType[T any](ev T) string {
	t := reflect.TypeOf(ev)
	return t.Name()
}

type iStateEvent interface {
	GetState() any
}

type StateEvent[T any] struct {
	State T
}

func NewStateEvent[T any](state T) StateEvent[T] {
	ev := StateEvent[T]{}
	ev.State = state
	return ev
}

func (ev StateEvent[T]) GetEventType() string {
	return TypeNameFromType(ev.State)
}

func (ev *StateEvent[T]) Deserialize(serialized string) error {
	return DeserializeFromJson(serialized, &ev.State)
}

func (ev StateEvent[T]) Serialize() string {
	return SerializeToJson(ev.State)
}

func (ev StateEvent[T]) GetState() any {
	return ev.State
}
