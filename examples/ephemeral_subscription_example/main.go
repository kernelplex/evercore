package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "os"
    "time"

    evercore "github.com/kernelplex/evercore/base"
    "github.com/kernelplex/evercore/evercoreuri"
    "modernc.org/sqlite"
)

// Register modernc sqlite under driver name "sqlite3" for dburl if used
var registered = false

func init() {
    if registered { return }
    registered = true
    sql.Register("sqlite3", &sqlite.Driver{})
}

// Cache item aggregate and events
type CacheItemState struct { Key string; Value string }
type CacheItemAggregate struct{ evercore.StateAggregate[CacheItemState] }

//evercoregen:state_event
type CacheItemCreated struct { Key string; Value string }
//evercoregen:state_event
type CacheItemUpdated struct { Value string }
//evercoregen:state_event
type CacheItemDeleted struct{}

func main() {
    // Generic DSN: if not provided, default to in-memory sqlite
    dsn := os.Getenv("EVERCORE_DSN")
    if dsn == "" {
        dsn = "sqlite://:memory:?cache=shared"
    }

    store, err := evercoreuri.Connect(dsn)
    if err != nil { log.Fatalf("connect: %v", err) }

    // Seed three cache items
    var ids []int64
    err = store.WithContext(context.Background(), func(etx evercore.EventStoreContext) error {
        for i := 1; i <= 3; i++ {
            agg := CacheItemAggregate{}
            if err := etx.CreateAggregateInto(&agg); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(CacheItemCreated{Key: fmt.Sprintf("K-%d", i), Value: fmt.Sprintf("V-%d", i)}), time.Now().UTC(), ""); err != nil { return err }
            ids = append(ids, agg.GetId())
        }
        return nil
    })
    if err != nil { log.Fatalf("seed: %v", err) }

    // Build a trivial in-memory cache keyed by aggregate id
    cache := map[int64]string{
        ids[0]: "V-1",
        ids[1]: "V-2",
        ids[2]: "V-3",
    }

    // Start an ephemeral subscription that listens for updates and deletes.
    // We start at the end so only new changes affect the cache.
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    go func() {
        err := store.RunEphemeralSubscription(
            ctx,
            evercore.SubscriptionFilter{AggregateType: "CacheItemState", EventTypes: []string{"CacheItemUpdated", "CacheItemDeleted"}},
            evercore.StartFrom{Kind: evercore.StartEnd},
            evercore.Options{BatchSize: 100, PollInterval: 100 * time.Millisecond},
            func(_ context.Context, events []evercore.SerializedEvent) error {
                for _, e := range events {
                    switch e.EventType {
                    case "CacheItemUpdated":
                        var upd CacheItemUpdated
                        _ = evercore.DecodeEventStateTo(e, &upd)
                        cache[e.AggregateId] = upd.Value
                        fmt.Printf("cache: set id=%d value=%s\n", e.AggregateId, upd.Value)
                    case "CacheItemDeleted":
                        delete(cache, e.AggregateId)
                        fmt.Printf("cache: delete id=%d\n", e.AggregateId)
                    }
                }
                return nil
            },
        )
        if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
            log.Printf("ephemeral sub error: %v", err)
        }
    }()

    // Simulate updates that the ephemeral sub will react to
    time.Sleep(150 * time.Millisecond)
    _ = store.WithContext(context.Background(), func(etx evercore.EventStoreContext) error {
        // Update first (we know sequence=1 after the initial create)
        agg1 := CacheItemAggregate{}
        agg1.SetId(ids[0])
        agg1.SetSequence(1)
        if err := etx.ApplyEventTo(&agg1, evercore.NewStateEvent(CacheItemUpdated{Value: "V-1-upd"}), time.Now().UTC(), ""); err != nil { return err }
        // Delete second
        agg2 := CacheItemAggregate{}
        agg2.SetId(ids[1])
        agg2.SetSequence(1)
        if err := etx.ApplyEventTo(&agg2, evercore.NewStateEvent(CacheItemDeleted{}), time.Now().UTC(), ""); err != nil { return err }
        return nil
    })

    <-ctx.Done()
    fmt.Printf("final cache: %+v\n", cache)
}
