package main

import (
    "context"
    "fmt"
    "log"
    "sync"
    "time"

    evercore "github.com/kernelplex/evercore/base"
)

// OrderState defines the aggregate state for orders
type OrderState struct {
    ID     string
    Status string
}

// OrderCreatedEvent
//
//evercoregen:state_event
type OrderCreatedEvent struct {
    ID     string
    Status string
}

// OrderShippedEvent
//
//evercoregen:state_event
type OrderShippedEvent struct {
    Status string
}

// OrderAggregate using StateAggregate
type OrderAggregate struct {
    evercore.StateAggregate[OrderState]
}

func main() {
    // Use in-memory storage engine for a simple demo
    store := evercore.NewEventStore(evercore.NewMemoryStorageEngine())

    // Seed some events
    err := store.WithContext(context.Background(), func(etx evercore.EventStoreContext) error {
        // Create a few orders
        for i := 1; i <= 3; i++ {
            agg := OrderAggregate{}
            if err := etx.CreateAggregateInto(&agg); err != nil {
                return err
            }
            // Created
            if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(OrderCreatedEvent{ID: fmt.Sprintf("O-%d", i), Status: "Created"}), time.Now().UTC(), ""); err != nil {
                return err
            }
            // Ship order 2 and 3
            if i >= 2 {
                if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(OrderShippedEvent{Status: "Shipped"}), time.Now().UTC(), ""); err != nil {
                    return err
                }
            }
        }
        return nil
    })
    if err != nil {
        log.Fatalf("failed to seed events: %v", err)
    }

    // Prepare to run a subscription that listens for OrderShippedEvent
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    wg := sync.WaitGroup{}
    wg.Add(1)
    go func() {
        defer wg.Done()
        _ = store.RunSubscription(
            ctx,
            "orders-shipped",
            evercore.SubscriptionFilter{AggregateType: "OrderState", EventTypes: []string{"OrderShippedEvent"}},
            evercore.StartFrom{Kind: evercore.StartBeginning},
            evercore.Options{BatchSize: 10, PollInterval: 200 * time.Millisecond, Lease: 5 * time.Second},
            func(_ context.Context, events []evercore.SerializedEvent) error {
                for _, e := range events {
                    fmt.Printf("Order shipped event: id=%d agg=%d type=%s at=%s\n", e.EventID, e.AggregateId, e.EventType, e.EventTime.Format(time.RFC3339))
                }
                // Stop after first batch for demo
                cancel()
                return nil
            },
        )
    }()

    wg.Wait()
}

