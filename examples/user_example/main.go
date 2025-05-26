package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yourorg/evercore/base"
	"github.com/yourorg/evercore/evercoresqlite"
)

// UserState represents the state of our User aggregate
type UserState struct {
	ID        int64
	Username  string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
	IsActive  bool
}

// UserCreatedEvent represents the initial creation of a user
type UserCreatedEvent struct {
	base.StateEvent[UserState]
}

// UserUpdatedEvent represents updates to a user
type UserUpdatedEvent struct {
	Username *string
	Email    *string
	IsActive *bool
}

func (e UserUpdatedEvent) GetEventType() string {
	return "UserUpdated"
}

func (e UserUpdatedEvent) Serialize() string {
	return base.SerializeToJson(e)
}

func main() {
	// Initialize SQLite in-memory database
	engine, err := evercoresqlite.NewSqliteStorageEngineWithConnection("file::memory:?cache=shared")
	if err != nil {
		log.Fatalf("Failed to create storage engine: %v", err)
	}

	// Create event store
	store := base.NewEventStore(engine)

	// Run operations in a transaction
	err = store.WithContext(context.Background(), func(ctx base.EventStoreContext) error {
		// Create a new user aggregate
		user := &base.StateAggregate[UserState]{}
		
		// Get a new aggregate ID
		userID, err := ctx.NewAggregateId("User")
		if err != nil {
			return fmt.Errorf("failed to create aggregate ID: %w", err)
		}

		// Create initial user state
		now := time.Now()
		createdEvent := UserCreatedEvent{
			State: UserState{
				ID:        userID,
				Username:  "johndoe",
				Email:     "john@example.com",
				CreatedAt: now,
				UpdatedAt: now,
				IsActive:  true,
			},
		}

		// Apply the creation event
		if err := ctx.ApplyEventTo(user, createdEvent, now, "user_creation"); err != nil {
			return fmt.Errorf("failed to apply creation event: %w", err)
		}

		// Later, update the user
		newEmail := "john.doe@example.org"
		updateEvent := UserUpdatedEvent{
			Email: &newEmail,
		}

		if err := ctx.ApplyEventTo(user, updateEvent, time.Now(), "user_update"); err != nil {
			return fmt.Errorf("failed to apply update event: %w", err)
		}

		// Print final state
		fmt.Printf("User State:\n%+v\n", user.State)

		return nil
	})

	if err != nil {
		log.Fatalf("Transaction failed: %v", err)
	}
}
