package evercore

import (
    "context"
    "testing"
    "time"
)

type testOrderState struct {
    ID     string
    Status string
}

//evercoregen:state_event
type testOrderCreatedEvent struct {
    ID     string
    Status string
}

//evercoregen:state_event
type testOrderShippedEvent struct {
    Status string
}

type testOrderAgg struct{ StateAggregate[testOrderState] }

func TestRunSubscription_MemoryEngine(t *testing.T) {
    store := NewEventStore(NewMemoryStorageEngine())

    // Seed events: 3 orders, 2 with shipped events
    err := store.WithContext(context.Background(), func(etx EventStoreContext) error {
        for i := 1; i <= 3; i++ {
            agg := testOrderAgg{}
            if err := etx.CreateAggregateInto(&agg); err != nil {
                return err
            }
            if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderCreatedEvent{ID: "O", Status: "Created"}), time.Now().UTC(), ""); err != nil {
                return err
            }
            if i >= 2 {
                if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderShippedEvent{Status: "Shipped"}), time.Now().UTC(), ""); err != nil {
                    return err
                }
            }
        }
        return nil
    })
    if err != nil {
        t.Fatalf("seed failed: %v", err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    count := 0
    go func() {
        _ = store.RunSubscription(
            ctx,
            "test-orders-shipped",
            SubscriptionFilter{AggregateType: "testOrderState", EventTypes: []string{"testOrderShippedEvent"}},
            StartFrom{Kind: StartBeginning},
            Options{BatchSize: 10, PollInterval: 50 * time.Millisecond, Lease: 2 * time.Second},
            func(_ context.Context, events []SerializedEvent) error {
                count += len(events)
                cancel()
                return nil
            },
        )
    }()

    select {
    case <-ctx.Done():
        // ok
    case <-time.After(2 * time.Second):
        t.Fatalf("timeout waiting for subscription")
    }

    if count != 2 {
        t.Fatalf("expected 2 shipped events, got %d", count)
    }
}

func TestRunEphemeralSubscription_StartEnd_SingleType(t *testing.T) {
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
    go func() {
        err := store.RunEphemeralSubscription(
            ctx,
            SubscriptionFilter{AggregateType: "testOrderState", EventTypes: []string{"testOrderShippedEvent"}},
            StartFrom{Kind: StartEnd},
            Options{BatchSize: 10, PollInterval: 25 * time.Millisecond},
            func(_ context.Context, events []SerializedEvent) error {
                got <- len(events)
                return nil
            },
        )
        if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
            t.Logf("ephemeral returned: %v", err)
        }
    }()

    // Give the loop one tick to read zero events at end, then add one new shipped event
    time.Sleep(60 * time.Millisecond)
    _ = store.WithContext(context.Background(), func(etx EventStoreContext) error {
        agg := testOrderAgg{}
        if err := etx.CreateAggregateInto(&agg); err != nil { return err }
        if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderCreatedEvent{ID: "N", Status: "Created"}), time.Now().UTC(), ""); err != nil { return err }
        if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderShippedEvent{Status: "Shipped"}), time.Now().UTC(), ""); err != nil { return err }
        return nil
    })

    var n int
    select {
    case n = <-got:
        // ok
    case <-ctx.Done():
        t.Fatalf("timeout waiting for ephemeral delivery")
    }
    if n != 1 {
        t.Fatalf("expected 1 shipped event from end, got %d", n)
    }
}

func TestRunEphemeralSubscription_Beginning_MultiType(t *testing.T) {
    store := NewEventStore(NewMemoryStorageEngine())

    // Seed: 2 aggregates: created + shipped on each
    if err := store.WithContext(context.Background(), func(etx EventStoreContext) error {
        for i := 0; i < 2; i++ {
            agg := testOrderAgg{}
            if err := etx.CreateAggregateInto(&agg); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderCreatedEvent{ID: "X", Status: "Created"}), time.Now().UTC(), ""); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, NewStateEvent(testOrderShippedEvent{Status: "Shipped"}), time.Now().UTC(), ""); err != nil { return err }
        }
        return nil
    }); err != nil {
        t.Fatalf("seed failed: %v", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    total := 0
    err := store.RunEphemeralSubscription(
        ctx,
        SubscriptionFilter{AggregateType: "testOrderState", EventTypes: []string{"testOrderCreatedEvent", "testOrderShippedEvent"}},
        StartFrom{Kind: StartBeginning},
        Options{BatchSize: 100, PollInterval: 10 * time.Millisecond},
        func(_ context.Context, events []SerializedEvent) error {
            total += len(events)
            cancel()
            return nil
        },
    )
    if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
        t.Fatalf("ephemeral failed: %v", err)
    }

    // 2 aggregates x (created + shipped) = 4
    if total != 4 {
        t.Fatalf("expected 4 events from beginning, got %d", total)
    }
}

