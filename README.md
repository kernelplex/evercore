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

The evercoregen tool generates boilerplate code for your
aggregates and events. First mark your types with these
annotations:

```go
//evercoregen:aggregate
type UserAggregate struct {
    base.StateAggregate[UserState]
}

//evercoregen:state_event 
type UserCreatedEvent struct {
    Username string
    Email    string
}

//evercoregen:event
type UserLoggedInEvent struct {
    Timestamp time.Time
}
```

Then install and run evercoregen:

```bash
go install github.com/kernelplex/evercore/cmd/evercoregen@latest
evercoregen -output-dir=internal/generated -output-pkg=generated
```

Here's a complete example:

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

### The evercoregen tool

The evercoregen tool can be used to automatically generate the
event and aggregate lists as well as build an event decoder
for your project.

To install the tool in your project (go v1.24+):

```bash
 go get --tool github.com/kernelplex/evercore/cmd/evercoregen/
```

This tool will look for sentinel comments in your *.go files:

### For Aggregates

```go
// evercore:aggregate
type UserAggregate struct {
  ...
}
```

### For State Events

```go
// evercore:state_event
type UserCreatedEvent struct {
  ...

## For Other Events

```go
// evercore:event
type UserCreatedEvent struct {
  ...


### Using the evercoregen tool

To invoke the tool:

```bash
go tool evercoregen  -output-dir=internal/generated -output-pkg=generated
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
