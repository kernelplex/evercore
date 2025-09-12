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

