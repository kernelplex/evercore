package evercore_test

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/kernelplex/evercore/pkg/evercore"
)

const (
	EventCreated = "created"
	EventUpdated = "updated"
)

const (
	AggregateUser = "user"
)

type User struct {
	FirstName    string
	LastName     string
	Email        string
	PasswordHash string
}

type UserAggregate struct {
	Id       int64
	State    User
	Sequence int64
}

func (agg *UserAggregate) SetId(id int64) {
	agg.Id = id
}

func (agg *UserAggregate) GetId() int64 {
	return agg.Id
}

func (agg *UserAggregate) SetSequence(sequence int64) {
	agg.Sequence = sequence
}

func (agg *UserAggregate) GetAggregateType() string {
	return AggregateUser
}

func (agg *UserAggregate) GetSnapshotFrequency() int64 {
	return 2
}

func (agg *UserAggregate) GetSnapshotState() (*string, error) {
	bytes, err := json.Marshal(agg.State)
	if err != nil {
		return nil, err
	}
	stringly := string(bytes)
	return &stringly, nil

}

type UserUpdatedEvent struct {
	FirstName    *string
	LastName     *string
	PasswordHash *string
}

func (ev UserUpdatedEvent) GetEventType() string {
	return EventUpdated
}

func (ev UserUpdatedEvent) Serialize() string {
	serialized, err := json.Marshal(ev)
	if err != nil {
		panic("State failed to serialize.")
	}
	return string(serialized)
}

func (ev *UserUpdatedEvent) Deserialize(serialized string) error {
	err := json.Unmarshal([]byte(serialized), ev)
	return err
}

type UserCreatedEvent struct {
	FirstName    string
	LastName     string
	Email        string
	PasswordHash string
}

func (ev UserCreatedEvent) GetEventType() string {
	return EventCreated
}

func (ev UserCreatedEvent) Serialize() string {
	serialized, err := json.Marshal(ev)
	if err != nil {
		panic("State failed to serialize.")
	}
	return string(serialized)
}

func (ev *UserCreatedEvent) Deserialize(serialized string) error {
	err := json.Unmarshal([]byte(serialized), ev)
	return err
}

func (agg *UserAggregate) DecodeEvent(event evercore.SerializedEvent) (evercore.EventState, error) {
	switch event.EventType {
	case EventCreated:
		state := UserCreatedEvent{}
		err := evercore.DecodeEventStateTo(event, &state)
		return state, err
	case EventUpdated:
		state := UserUpdatedEvent{}
		err := evercore.DecodeEventStateTo(event, &state)
		return state, err
	}
	return nil, fmt.Errorf("Unable to decode event type: %s", event.EventType)
}

func (agg *UserAggregate) GetSequence() int64 {
	return agg.Sequence
}

func (agg *UserAggregate) ApplyEventState(eventState evercore.EventState, time time.Time, reference string) error {

	switch eventState.(type) {
	case UserCreatedEvent:
		created := eventState.(UserCreatedEvent)
		agg.State.Email = created.Email
		agg.State.FirstName = created.FirstName
		agg.State.LastName = created.LastName
		agg.State.PasswordHash = created.PasswordHash

	case UserUpdatedEvent:
		updated := eventState.(UserUpdatedEvent)
		agg.State.FirstName = Coalesce(updated.FirstName, agg.State.FirstName)
		agg.State.LastName = Coalesce(updated.LastName, agg.State.LastName)
		agg.State.PasswordHash = Coalesce(updated.PasswordHash, agg.State.PasswordHash)
	}

	return nil
}

func (agg *UserAggregate) ApplySnapshot(snapshot *evercore.Snapshot) error {
	newState := User{}
	json.Unmarshal([]byte(snapshot.State), &newState)
	agg.Id = snapshot.AggregateId
	agg.State = newState
	agg.Sequence = snapshot.Sequence
	return nil

}

func Coalesce[T any](target *T, defaultValue T) T {
	if target != nil {
		return *target
	}
	return defaultValue
}

type OpinionatedTestCases struct {
	store        *evercore.EventStore
	userState    User
	sampleUserId int64
}

func TestCases(t *testing.T) {
	storageEngine := evercore.NewMemoryStorageEngine()
	eventstore := evercore.NewEventStore(storageEngine)

	userState := User{
		FirstName:    "Miguel",
		LastName:     "Chavez",
		Email:        "chavez@example.com",
		PasswordHash: "zx9893oqpdcleiturt",
	}

	testCases := OpinionatedTestCases{
		store:     eventstore,
		userState: userState,
	}

	t.Run("New user aggregate", testCases.CreateUser)
	t.Run("Load user aggregate", testCases.LoadUser)
	t.Run("Update user aggregate", testCases.UpdateUser)
	t.Run("Load updated user", testCases.LoadUpdatedUser)
}

func (c *OpinionatedTestCases) F(t *testing.T) {
	t.Errorf("Not implemented")
}

func (c *OpinionatedTestCases) CreateUser(t *testing.T) {
	ctx := context.Background()
	err := c.store.WithContext(ctx, func(etx evercore.EventStoreContext) error {
		user := UserAggregate{}
		err := etx.CreateAggregateWithKeyInto(&user, c.userState.Email)
		if err != nil {
			return err
		}
		c.sampleUserId = user.Id

		event := UserCreatedEvent{
			FirstName:    c.userState.FirstName,
			LastName:     c.userState.LastName,
			Email:        c.userState.Email,
			PasswordHash: c.userState.PasswordHash,
		}
		etx.ApplyEventTo(&user, event, time.Now().UTC(), "")

		if !reflect.DeepEqual(user.State, c.userState) {
			t.Errorf("Aggregate state does not match.")
			t.Errorf("Expected: %+v", c.userState)
			t.Errorf("Got: %+v", user.State)
		}
		return nil
	})
	if err != nil {
		t.Errorf("Failed to create aggregate: %v", err)
	}
}

func (c *OpinionatedTestCases) LoadUser(t *testing.T) {
	ctx := context.Background()
	email, err := evercore.InContext(ctx, c.store, func(etx evercore.EventStoreContext) (string, error) {
		user := UserAggregate{}
		err := etx.LoadStateInto(&user, c.sampleUserId)
		if err != nil {
			return "", err
		}

		if !reflect.DeepEqual(user.State, c.userState) {
			t.Errorf("Aggregate state does not match.")
			t.Errorf("Expected: %+v", c.userState)
			t.Errorf("Got: %+v", user.State)
		}
		return user.State.Email, nil
	})
	if err != nil {
		t.Errorf("Failed to create aggregate: %v", err)
	}

	if email != c.userState.Email {
		t.Errorf("Received wrong return type: wanted %v got %v", c.userState.Email, email)
	}
}

func (c *OpinionatedTestCases) UpdateUser(t *testing.T) {
	ctx := context.Background()
	err := c.store.WithContext(ctx, func(etx evercore.EventStoreContext) error {
		user := UserAggregate{}
		err := etx.LoadStateInto(&user, c.sampleUserId)
		if err != nil {
			return err
		}
		newLastName := "Rodriguez"
		updateEvent := UserUpdatedEvent{
			LastName: &newLastName,
		}
		etx.ApplyEventTo(&user, updateEvent, time.Now().UTC(), "")
		modifiedUser := c.userState
		modifiedUser.LastName = newLastName

		if !reflect.DeepEqual(user.State, modifiedUser) {
			t.Errorf("Aggregate state does not match.")
			t.Errorf("Expected: %+v", c.userState)
			t.Errorf("Got: %+v", user.State)
		}

		return nil
	})
	if err != nil {
		t.Errorf("Failed to create aggregate: %v", err)
	}
}

func (c *OpinionatedTestCases) LoadUpdatedUser(t *testing.T) {
	ctx := context.Background()
	err := c.store.WithContext(ctx, func(etx evercore.EventStoreContext) error {
		user := UserAggregate{}
		err := etx.LoadStateByKeyInto(&user, c.userState.Email)
		if err != nil {
			return err
		}
		newLastName := "Rodriguez"
		modifiedUser := c.userState
		modifiedUser.LastName = newLastName

		if !reflect.DeepEqual(user.State, modifiedUser) {
			t.Errorf("Aggregate state does not match.")
			t.Errorf("Expected: %+v", c.userState)
			t.Errorf("Got: %+v", user.State)
		}

		return nil
	})
	if err != nil {
		t.Errorf("Failed to create aggregate: %v", err)
	}
}
