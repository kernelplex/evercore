package main

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/evercoresqlite"
	_ "github.com/mattn/go-sqlite3"
)

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

//evercoregen:state_event
// UserCreatedEvent represents the initial creation of a user
type UserCreatedEvent struct {
	Username string
	Email    string
	IsActive bool
}

//evercoregen:state_event
// UserUpdatedEvent represents updates to a user
type UserUpdatedEvent struct {
	Username *string
	Email    *string
	IsActive *bool
}

func main() {
	// Initialize SQLite in-memory database
	connectionString := "file::memory:?cache=shared"
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
		return
	}
	if err := evercoresqlite.MigrateUp(db); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	engine := evercoresqlite.NewSqliteStorageEngine(db)

	// Create event store
	store := evercore.NewEventStore(engine)

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
