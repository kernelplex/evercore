package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	evercore "github.com/kernelplex/evercore/base"
	"github.com/kernelplex/evercore/evercoreuri"
	"modernc.org/sqlite"
)

// Register modernc sqlite under driver name "sqlite" for dburl
var registered = false

func init() {
	if registered {
		return
	}
	registered = true
	sql.Register("sqlite3", &sqlite.Driver{})
}

// Simple aggregate + events
type InvoiceState struct {
	Number string
	Status string
}
type InvoiceAggregate struct {
	evercore.StateAggregate[InvoiceState]
}

//evercoregen:state_event
type InvoiceCreated struct {
	Number string
	Status string
}

//evercoregen:state_event
type InvoicePaid struct{ Status string }

func main() {
	// Accept DSN from env; fallback to in-memory if missing
	dsn := os.Getenv("SQLITE_TEST_RUNNER_CONNECTION")
	if dsn == "" {
		dsn = "sqlite://:memory:?cache=shared"
	}

	store, err := evercoreuri.Connect(dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}

	// Seed some events
	err = store.WithContext(context.Background(), func(etx evercore.EventStoreContext) error {
		for i := 1; i <= 3; i++ {
			agg := InvoiceAggregate{}
			if err := etx.CreateAggregateInto(&agg); err != nil {
				return err
			}
			if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(InvoiceCreated{Number: fmt.Sprintf("INV-%d", i), Status: "Created"}), time.Now().UTC(), ""); err != nil {
				return err
			}
			if i >= 2 {
				if err := etx.ApplyEventTo(&agg, evercore.NewStateEvent(InvoicePaid{Status: "Paid"}), time.Now().UTC(), ""); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("seed: %v", err)
	}

	// Run a durable subscription for InvoicePaid events
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Coordinate shutdown without canceling inside the handler directly
	stop := make(chan struct{}, 1)
	go func() { <-stop; cancel() }()

	// Retry until claimed if already owned elsewhere
	opts := evercore.Options{BatchSize: 100, PollInterval: 200 * time.Millisecond, Lease: 5 * time.Second}
	retryDelay := opts.Lease / 2
	for {
		err = store.RunSubscription(
			ctx,
			"invoices-paid",
			evercore.SubscriptionFilter{AggregateType: "InvoiceState", EventTypes: []string{"InvoicePaid"}},
			evercore.StartFrom{Kind: evercore.StartBeginning},
			opts,
			func(_ context.Context, events []evercore.SerializedEvent) error {
				for _, e := range events {
					fmt.Printf("Paid: event_id=%d aggregate_id=%d type=%s time=%s\n", e.EventID, e.AggregateId, e.EventType, e.EventTime.Format(time.RFC3339))
				}
				// exit after first delivery for demo
				select { case stop <- struct{}{}: default: }
				return nil
			},
		)
		if errors.Is(err, evercore.ErrSubscriptionAlreadyOwned) {
			select {
			case <-ctx.Done():
				err = ctx.Err()
				break
			case <-time.After(retryDelay):
				continue
			}
		}
		break
	}
	if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
		log.Fatalf("run: %v", err)
	}
}
