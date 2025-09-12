package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    evercore "github.com/kernelplex/evercore/base"
    "github.com/kernelplex/evercore/evercoreuri"
    _ "github.com/jackc/pgx/v5/stdlib" // register pgx driver
)

// Simple aggregate + events
type PaymentState struct { Ref string; Status string }
type PaymentAggregate struct{ evercore.StateAggregate[PaymentState] }

//evercoregen:state_event
type PaymentInitiated struct { Ref string; Status string }
//evercoregen:state_event
type PaymentSettled struct { Status string }

func main() {
    dsn := os.Getenv("PG_TEST_RUNNER_CONNECTION")
    if dsn == "" {
        log.Fatalf("set PG_TEST_RUNNER_CONNECTION to a postgres DSN, e.g. postgres://user:pass@localhost:5432/dbname?sslmode=disable")
    }

    store, err := evercoreuri.Connect(dsn)
    if err != nil { log.Fatalf("connect: %v", err) }

    // Seed a few payments
    err = store.WithContext(context.Background(), func(etx evercore.EventStoreContext) error {
        for i := 1; i <= 2; i++ {
            agg := PaymentAggregate{}
            if err := etx.CreateAggregateInto(&agg); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(PaymentInitiated{Ref: fmt.Sprintf("P-%d", i), Status: "Initiated"}), time.Now().UTC(), ""); err != nil { return err }
            if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(PaymentSettled{Status: "Settled"}), time.Now().UTC(), ""); err != nil { return err }
        }
        return nil
    })
    if err != nil { log.Fatalf("seed: %v", err) }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    err = store.RunSubscription(
        ctx,
        "payments-settled",
        evercore.SubscriptionFilter{AggregateType: "PaymentState", EventTypes: []string{"PaymentSettled"}},
        evercore.StartFrom{Kind: evercore.StartBeginning},
        evercore.Options{BatchSize: 100, PollInterval: 300 * time.Millisecond, Lease: 10 * time.Second},
        func(_ context.Context, events []evercore.SerializedEvent) error {
            for _, e := range events {
                fmt.Printf("Settled: event_id=%d agg=%d type=%s at=%s\n", e.EventID, e.AggregateId, e.EventType, e.EventTime.Format(time.RFC3339))
            }
            cancel()
            return nil
        },
    )
    if err != nil && err != context.Canceled && err != context.DeadlineExceeded { log.Fatalf("run: %v", err) }
}

