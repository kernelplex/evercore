// The uri package provides a standard URI format for connecting to an EverCore
// evnet store. It is meant to handle a lot of the boilerplate around connecting
// and migrating the database.

package evercoreuri

import (
	"database/sql"
	"fmt"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/evercorepostgres"
	"github.com/kernelplex/evercore/evercoresqlite"
	"github.com/xo/dburl"
)

// Connect returns a new event store for the given URI.
// The storage engine will be automatically migrated if necessary.
func Connect(uri string) (*evercore.EventStore, error) {
	// Get the storage engine.
	storageEngine, err := GetStorageEngine(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage engine: %w", err)
	}

	eventStore := evercore.NewEventStore(storageEngine)
	return eventStore, nil
}

// GetStorageEngine returns the storage engine for the given URI.
// This will perform migrations if necessary.
func GetStorageEngine(uri string) (evercore.StorageEngine, error) {
	durl, err := dburl.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("failed to parse uri: %w", err)
	}

	db, err := sql.Open(durl.Driver, durl.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	switch durl.Driver {
	case "postgres":
		evercorepostgres.MigrateUp(db)
		return evercorepostgres.NewPostgresStorageEngine(db), nil

	case "sqlite3":
		evercoresqlite.MigrateUp(db)
		return evercoresqlite.NewSqliteStorageEngine(db), nil
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", durl.Driver)
	}
}
