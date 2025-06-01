# Evercore - Event Sourcing Framework for Go

Evercore is an event store implementation supporting multiple storage backends
with strong typing and transaction safety.

## Features

- Multiple storage backends:
  - PostgreSQL (production)
  - SQLite (development/testing)
  - In-memory (unit testing)
- Event sourcing fundamentals:
  - Event storage and replay
  - Snapshot support
  - Optimistic concurrency control
- Strong typing:
  - Generic aggregate state support (StateAggregate[T])
  - Type-safe event handling
  - Automatic state validation
- Transactional operations
- Natural key support
- Migration support via Goose

## Quick Start

```go
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

// UserState represents the aggregate state
type UserState struct {
     Username string
     Email    string
     IsActive bool
}

type UserAggregate struct {
     base.StateAggregate[UserState]
}

// UserCreatedEvent represents the creation event
type UserCreatedEvent struct {
     Username string
     Email    string
     IsActive bool
}

func main() {
     // Initialize SQLite
     db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
     if err != nil {
          log.Fatal(err)
     }
     evercoresqlite.MigrateUp(db)

     // Create event store
     store := base.NewEventStore(evercoresqlite.NewSqliteStorageEngine(db))

     // Run transaction
     err = store.WithContext(context.Background(), func(ctx base.EventStoreContext) error {
          user := UserAggregate{}
          if err := ctx.CreateAggregateInto(&user); err != nil {
               return err
          }

          event := base.NewStateEvent(UserCreatedEvent{
               Username: "johndoe",
               Email:    "john@example.com",
               IsActive: true,
          })
          return ctx.ApplyEventTo(&user, event, time.Now(), "init")
     })

     if err != nil {
          log.Fatal(err)
     }
}
```

## Storage Backends

### PostgreSQL

```bash
export PG_TEST_RUNNER_CONNECTION="postgres://user:pass@localhost:5432/db?sslmode=disable"
make integration-test-postgres
```

### SQLite

```bash
make integration-test-sqlite
```

### In-Memory

```go
engine := base.NewMemoryStorageEngine()
```

## Testing

Run all unit tests:

```bash
make test
```

Run integration tests:

```bash
make integration-test  # All databases
make integration-test-sqlite  # SQLite only
make integration-test-postgres  # PostgreSQL only
```

## Development

Use the scratch directory for experimentation:

```bash
make scratch  # Runs scratch/main.go
```

## Architecture

![Evercore Architecture Diagram](docs/architecture.png)

Key components:

- `EventStore`: Core coordinator
- `StorageEngine`: Pluggable storage interface
- `Aggregate`: Domain object interface
- `EventState`: Event interface
- `StateAggregate`: Generic state container

## Hints

- Use store.Warmup() as early as possible in your application
- Keep the list of known aggregate and event types up to date.  
  This will save database calls during normal operation.
- Use separate databases for your event store and your relational  
  model (espectially if using sqlite)
