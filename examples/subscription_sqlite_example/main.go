package main

import (
    "context"
    "database/sql"
    "fmt"
    "log"
    "time"

    evercore "github.com/kernelplex/evercore/base"
    "github.com/kernelplex/evercore/evercoreuri"
    "modernc.org/sqlite"
)

// Register modernc sqlite under driver name "sqlite" for dburl
func init() { sql.Register("sqlite", &sqlite.Driver{}) }

// Simple aggregate + events
type InvoiceState struct { Number string; Status string }
type InvoiceAggregate struct{ evercore.StateAggregate[InvoiceState] }

//evercoregen:state_event
type InvoiceCreated struct { Number string; Status string }
//evercoregen:state_event
type InvoicePaid struct { Status string }

func main() {
    // Connect to in-memory sqlite via URI (migrations applied automatically)
    store, err := evercoreuri.Connect("sqlite://:memory:?cache=shared")
    if err != nil { log.Fatalf("connect: %v", err) }

    // Seed some events
    err = store.WithContext(context.Background(), func(etx evercore.EventStoreContext) error {
        for i := 1; i <= 3; i++ {
            agg := InvoiceAggregate{}
            if err := etx.CreateAggregateInto(&agg); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(InvoiceCreated{Number: fmt.Sprintf("INV-%d", i), Status: "Created"}), time.Now().UTC(), ""); err != nil { return err }
            if i >= 2 {
                if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(InvoicePaid{Status: "Paid"}), time.Now().UTC(), ""); err != nil { return err }
            }
        }
        return nil
    })
    if err != nil { log.Fatalf("seed: %v", err) }

    // Run a durable subscription for InvoicePaid events
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()

    err = store.RunSubscription(
        ctx,
        "invoices-paid",
        evercore.SubscriptionFilter{AggregateType: "InvoiceState", EventTypes: []string{"InvoicePaid"}},
        evercore.StartFrom{Kind: evercore.StartBeginning},
        evercore.Options{BatchSize: 100, PollInterval: 200 * time.Millisecond, Lease: 5 * time.Second},
        func(_ context.Context, events []evercore.SerializedEvent) error {
            for _, e := range events {
                fmt.Printf("Paid: event_id=%d aggregate_id=%d type=%s time=%s\n", e.EventID, e.AggregateId, e.EventType, e.EventTime.Format(time.RFC3339))
            }
            // exit after first delivery for demo
            cancel()
            return nil
        },
    )
    if err != nil && err != context.Canceled && err != context.DeadlineExceeded { log.Fatalf("run: %v", err) }
}

