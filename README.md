# Evercore - Event Sourcing Framework for Go

Evercore is a production-ready event store implementation supporting multiple storage backends with strong typing and transaction safety.

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
  - Generic aggregate state support
  - Type-safe event handling
- Transactional operations
- Natural key support
- Migration support via Goose

## Quick Start

```go
package main

import (
	"github.com/yourorg/evercore/base"
	"github.com/yourorg/evercore/evercoresqlite"
)

type UserState struct {
	Name  string
	Email string
}

type UserCreated struct {
	base.StateEvent[UserState]
}

func main() {
	// Initialize SQLite storage
	engine := evercoresqlite.NewSqliteStorageEngine(":memory:")
	
	// Create event store
	store := base.NewEventStore(engine)
	
	// Start a context
	ctx := store.NewContext(context.Background())
	defer ctx.Commit()
	
	// Create aggregate
	user := base.NewStateAggregate[UserState]()
	id, _ := ctx.NewAggregateId("User")
	
	// Apply event
	event := UserCreated{State: UserState{Name: "Alice", Email: "alice@example.com"}}
	ctx.ApplyEventTo(user, event, time.Now(), "init")
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
