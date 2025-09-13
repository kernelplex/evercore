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
// evercore:aggregate
type UserAggregate struct {
    evercore.StateAggregate[UserState]
}

// evercore:state-event 
type UserCreatedEvent struct {
    Username string
    Email    string
}

// evercore:event
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

     evercore "github.com/kernelplex/evercore/base"
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
     evercore.StateAggregate[UserState]
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
     store := evercore.NewEventStore(evercoresqlite.NewSqliteStorageEngine(db))

     // Run transaction
     err = store.WithContext(context.Background(), func(ctx evercore.EventStoreContext) error {
          user := UserAggregate{}
          if err := ctx.CreateAggregateInto(&user); err != nil {
               return err
          }

          event := evercore.NewStateEvent(UserCreatedEvent{
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
make integration-test-postgres
```

### SQLite

```bash
make integration-test-sqlite
```

### In-Memory

```go
engine := evercore.NewMemoryStorageEngine()
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
// evercore:state-event
type UserCreatedEvent struct {
  ...
}
```

### For Other Events

```go
// evercore:event
type UserCreatedEvent struct {
  ...
}
```

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

## Subscriptions

Evercore includes two subscription modes for streaming events in global order:

- Durable (tracked in DB): survives restarts and stores a cursor (`last_event_id`).
- Ephemeral (in-process only): no DB row, dies with the service; great for cache invalidation.

Common features
- Filters: aggregate type, event type(s), and optional aggregate key.
- Start positions: `beginning`, `end`, `event_id`, `timestamp`.
- Delivery: at-least-once; keep handlers idempotent.

Durable API: `EventStore.RunSubscription(ctx, name, filter, start, opts, handler)`
- Claiming may fail if another worker holds the lease. Detect with `errors.Is(err, evercore.ErrSubscriptionAlreadyOwned)` and retry (e.g., at half the lease time).

Ephemeral API: `EventStore.RunEphemeralSubscription(ctx, filter, start, opts, handler)`
- No lease, no DB writes; position is tracked in memory only.

Examples
- Postgres durable: `examples/subscription_postgres_example/main.go`
  - Env: `PG_TEST_RUNNER_CONNECTION=postgres://user:pass@host:5432/db?sslmode=disable`
  - Run: `go run examples/subscription_postgres_example/main.go`
- SQLite durable: `examples/subscription_sqlite_example/main.go`
  - Env (optional): `SQLITE_TEST_RUNNER_CONNECTION=sqlite:///path/to/db.sqlite?cache=shared`
  - Defaults to in-memory if not set
  - Run: `go run examples/subscription_sqlite_example/main.go`
- Ephemeral cache invalidation: `examples/ephemeral_subscription_example/main.go`
  - Env (optional): `EVERCORE_DSN=postgres://...` or `sqlite:///...`; defaults to in-memory SQLite
  - Run: `go run examples/ephemeral_subscription_example/main.go`

## Development

Use the scratch directory for experimentation:

```bash
make scratch  # Runs scratch/main.go
```

## Architecture

Key components:

- `EventStore`: Core coordinator
- `StorageEngine`: Pluggable storage interface
- `Aggregate`: Domain object interface
- `EventState`: Event interface
- `StateAggregate`: Generic state container

## Hints

- Use `store.Warmup()` as early as possible in your application.
- Keep the list of known aggregate and event types up to date to save database calls during normal operation.
- Use separate databases for your event store and your relational model (especially if using SQLite).
