//go:build !race

package evercore

import (
    "context"
    "testing"
    "time"
)

// This test validates StartEnd behavior for ephemeral subscriptions.
// It is excluded under -race because the MemoryStorageEngine is not
// concurrency-safe for simultaneous reads/writes.
func TestRunEphemeralSubscription_StartEnd_SingleType_NoRace(t *testing.T) {
    store := NewEventStore(NewMemoryStorageEngine())

    // Seed: 2 created + shipped events
    if err := store.WithContext(context.Background(), func(etx EventStoreContext) error {
        for i := 0; i < 2; i++ {
            agg := testOrderAgg{}
            if err := etx.CreateAggregateInto(&agg); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderCreatedEvent{ID: "O", Status: "Created"}), time.Now().UTC(), ""); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderShippedEvent{Status: "Shipped"}), time.Now().UTC(), ""); err != nil { return err }
        }
        return nil
    }); err != nil {
        t.Fatalf("seed failed: %v", err)
    }

    // Start ephemeral from end; expect only newly added shipped events to be delivered
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    got := make(chan int, 1)
    done := make(chan struct{})
    go func() {
        defer close(done)
        _ = store.RunEphemeralSubscription(
            ctx,
            SubscriptionFilter{AggregateType: "testOrderState", EventTypes: []string{"testOrderShippedEvent"}},
            StartFrom{Kind: StartEnd},
            Options{BatchSize: 10, PollInterval: 200 * time.Millisecond},
            func(_ context.Context, events []SerializedEvent) error {
                got <- len(events)
                return nil
            },
        )
    }()

    // Give the loop time to enter sleep
    time.Sleep(50 * time.Millisecond)
    _ = store.WithContext(context.Background(), func(etx EventStoreContext) error {
        agg := testOrderAgg{}
        if err := etx.CreateAggregateInto(&agg); err != nil { return err }
        if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderCreatedEvent{ID: "N", Status: "Created"}), time.Now().UTC(), ""); err != nil { return err }
        if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderShippedEvent{Status: "Shipped"}), time.Now().UTC(), ""); err != nil { return err }
        return nil
    })

    select {
    case n := <-got:
        if n != 1 {
            t.Fatalf("expected 1 shipped event from end, got %d", n)
        }
    case <-ctx.Done():
        t.Fatalf("timeout waiting for ephemeral delivery")
    }

    // Ensure subscription goroutine exits cleanly
    cancel()
    <-done
}

