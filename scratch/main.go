package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/evercoresqlite"
	_ "github.com/mattn/go-sqlite3"
)

type Tenant struct {
	Parent     int64
	Name       string
	SystemName string
}

type TenantAggregate struct {
	evercore.StateAggregate[Tenant]
}

type TenantCreatedEvent struct {
	Parent     int64
	Name       string
	SystemName string
}

func (ev TenantCreatedEvent) GetEventType() string {
	return "TenantCreatedEvent"
}

func (ev TenantCreatedEvent) Serialize() string {
	serialized, err := json.Marshal(ev)
	if err != nil {
		panic("State failed to serialize.")
	}
	return string(serialized)
}

func (ev *TenantCreatedEvent) Deserialize(serialized string) error {
	err := json.Unmarshal([]byte(serialized), ev)
	return err
}

func main() {
	// db, err := sql.Open("sqlite3", "file:///mnt/data/seddy/proj/evercore/scratch/data.db?cache=shared")
	// db, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	db, err := sql.Open("sqlite3", "file:///mnt/data/seddy/proj/evercore/scratch/data.db")
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA synchronous=NORMAL;")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA jounal_size_limit=6144000;")
	if err != nil {
		panic(err)
	}

	evercoresqlite.MigrateUp(db)

	engine := evercoresqlite.NewSqliteStorageEngine(db)
	es := evercore.NewEventStore(engine)
	ctx := context.Background()
	err = es.WithContext(ctx, func(etx evercore.EventStoreContext) error {
		tenantAggregate := TenantAggregate{}
		err = etx.CreateAggregateWithKeyInto(&tenantAggregate, "mycooltenant")
		if err != nil {
			return err
		}
		createdEvent := TenantCreatedEvent{
			Parent:     0,
			Name:       "My Cool Tenant",
			SystemName: "mycooltenant",
		}
		err = etx.ApplyEventTo(&tenantAggregate, createdEvent, time.Now().UTC(), "")
		if err != nil {
			return err
		}
		fmt.Printf("Aggregate state: %+v\n", tenantAggregate.State)
		return nil
	})
	if err != nil {
		panic(err)
	}

}
