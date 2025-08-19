package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/evercoreuri"
	"modernc.org/sqlite"
)

func init() {
	// Register the modernc.org/sqlite driver under the name "sqlite3"
	sql.Register("sqlite3", &sqlite.Driver{})
}

// UserState represents the state of our User aggregate
type UserState struct {
	Username string
	Email    string
	IsActive bool
}

//evercoregen:aggregate
type UserAggregate struct {
	evercore.StateAggregate[UserState]
}

// UserCreatedEvent represents the initial creation of a user
//
//evercoregen:state_event
type UserCreatedEvent struct {
	Username string
	Email    string
	IsActive bool
}

// UserUpdatedEvent represents updates to a user
//
//evercoregen:state_event
type UserUpdatedEvent struct {
	Username *string
	Email    *string
	IsActive *bool
}

func main() {
	// Initialize SQLite in-memory database
	connectionString := "sqlite://:memory:?cache=shared"
	store, err := evercoreuri.Connect(connectionString)
	if err != nil {
		log.Fatalf("Failed to connect to event store: %v", err)
	}

	// Run operations in a transaction
	const username = "John"
	const email = "john@example.com"
	const isActive = true
	user := UserAggregate{}
	err = store.WithContext(context.Background(), func(etx evercore.EventStoreContext) error {
		err := etx.CreateAggregateInto(&user)
		if err != nil {
			return err
		}
		createdEvent := evercore.NewStateEvent(UserCreatedEvent{
			Username: username,
			Email:    email,
			IsActive: isActive,
		})
		err = etx.ApplyEventTo(&user, createdEvent, time.Now().UTC(), "")
		return err
	})

	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}

	if user.State.Username != username {
		log.Fatalf("Expected username to be %s, got %s", username, user.State.Username)
	}
	if user.State.Email != email {
		log.Fatalf("Expected email to be %s, got %s", email, user.State.Email)
	}
	if user.State.IsActive != isActive {
		log.Fatalf("Expected isActive to be %t, got %t", isActive, user.State.IsActive)
	}
}
