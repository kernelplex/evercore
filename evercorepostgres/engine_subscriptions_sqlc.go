//go:build withsqlc

package evercorepostgres

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "time"

    evercore "github.com/kernelplex/evercore/base"
)

// ---------------- Subscriptions (sqlc) ----------------

func (s *PostgresStorageEngine) UpsertSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, aggregateTypeId *int64, eventTypeId *int64, aggregateKey *string, startFrom string, startEventId int64, startTimestamp *time.Time) (int64, error) {
    db := s.maybeWrapTx(tx)
    q := New(db)
    id, err := q.UpsertSubscription(ctx, UpsertSubscriptionParams{
        Name:            name,
        AggregateTypeID: sql.NullInt64{Int64: valueOrZero(aggregateTypeId), Valid: aggregateTypeId != nil},
        EventTypeID:     sql.NullInt64{Int64: valueOrZero(eventTypeId), Valid: eventTypeId != nil},
        AggregateKey:    sql.NullString{String: valueOrEmpty(aggregateKey), Valid: aggregateKey != nil},
        StartFrom:       startFrom,
        StartEventID:    startEventId,
        StartTimestamp:  sql.NullTime{Time: valueOrZeroTime(startTimestamp), Valid: startTimestamp != nil},
    })
    if err != nil { return 0, WrapError("failed to upsert subscription", err) }
    return id, nil
}

func (s *PostgresStorageEngine) AddSubscriptionEventType(tx evercore.StorageEngineTxInfo, ctx context.Context, subscriptionId int64, eventTypeId int64) error {
    db := s.maybeWrapTx(tx)
    q := New(db)
    return WrapError("failed to add subscription event type", q.AddSubscriptionEventType(ctx, AddSubscriptionEventTypeParams{SubscriptionID: subscriptionId, EventTypeID: eventTypeId}))
}

func (s *PostgresStorageEngine) GetSubscriptionByName(tx evercore.StorageEngineTxInfo, ctx context.Context, name string) (*evercore.Subscription, error) {
    db := s.maybeWrapTx(tx)
    q := New(db)
    row, err := q.GetSubscriptionByName(ctx, name)
    if err != nil { return nil, WrapError("failed to get subscription", err) }
    sub := toBaseSubscription(row)
    return &sub, nil
}

func (s *PostgresStorageEngine) SetSubscriptionActive(tx evercore.StorageEngineTxInfo, ctx context.Context, id int64, active bool) error {
    db := s.maybeWrapTx(tx)
    q := New(db)
    return WrapError("failed to set subscription active", q.SetSubscriptionActive(ctx, SetSubscriptionActiveParams{ID: id, Active: active}))
}

func (s *PostgresStorageEngine) ClaimSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, owner string, lease time.Duration) (bool, error) {
    db := s.maybeWrapTx(tx)
    q := New(db)
    n, err := q.ClaimSubscription(ctx, ClaimSubscriptionParams{Name: name, Owner: owner, LeaseExpiresAt: time.Now().Add(lease), Now: time.Now()})
    if err != nil { return false, WrapError("failed to claim subscription", err) }
    return n == 1, nil
}

func (s *PostgresStorageEngine) RenewSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, owner string, lease time.Duration) (bool, error) {
    db := s.maybeWrapTx(tx)
    q := New(db)
    n, err := q.RenewSubscription(ctx, RenewSubscriptionParams{Name: name, Owner: owner, LeaseExpiresAt: time.Now().Add(lease)})
    if err != nil { return false, WrapError("failed to renew subscription", err) }
    return n == 1, nil
}

func (s *PostgresStorageEngine) ReleaseSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, owner string) error {
    db := s.maybeWrapTx(tx)
    q := New(db)
    return WrapError("failed to release subscription", q.ReleaseSubscription(ctx, ReleaseSubscriptionParams{Name: name, Owner: owner}))
}

func (s *PostgresStorageEngine) GetEventsForSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, sub *evercore.Subscription, limit int) ([]evercore.SerializedEvent, error) {
    db := s.maybeWrapTx(tx)
    // Fallback to raw SQL for this query due to sqlc LIMIT parameter parsing constraints.
    multi := sub.EventTypeID == nil

    q := `SELECT id, aggregate_id, natural_key, sequence, aggregate_type_id, aggregate_type, event_type_id, event_type, event_time, state
          FROM event_log WHERE id > $1`
    args := []any{sub.LastEventID}
    idx := 2
    if sub.AggregateTypeID != nil { q += fmt.Sprintf(" AND aggregate_type_id = $%d", idx); args = append(args, *sub.AggregateTypeID); idx++ }
    if sub.AggregateKey != nil { q += fmt.Sprintf(" AND natural_key = $%d", idx); args = append(args, *sub.AggregateKey); idx++ }
    if multi { q += fmt.Sprintf(" AND event_type_id IN (SELECT event_type_id FROM subscription_event_types WHERE subscription_id = $%d)", idx); args = append(args, sub.ID); idx++ } else if sub.EventTypeID != nil { q += fmt.Sprintf(" AND event_type_id = $%d", idx); args = append(args, *sub.EventTypeID); idx++ }
    q += fmt.Sprintf(" ORDER BY id ASC LIMIT $%d", idx); args = append(args, limit)

    rows, err := db.QueryContext(ctx, q, args...)
    if err != nil { if errors.Is(err, sql.ErrNoRows) { return []evercore.SerializedEvent{}, nil }; return nil, WrapError("failed to query events for subscription", err) }
    defer rows.Close()
    out := make([]evercore.SerializedEvent, 0, limit)
    for rows.Next() {
        var id, aggregateID, sequence int64
        var naturalKey sql.NullString
        var aggTypeId, evtTypeId int64
        var aggTypeName, evtTypeName string
        var eventTime time.Time
        var state string
        if err := rows.Scan(&id, &aggregateID, &naturalKey, &sequence, &aggTypeId, &aggTypeName, &evtTypeId, &evtTypeName, &eventTime, &state); err != nil { return nil, WrapError("failed to scan events for subscription", err) }
        out = append(out, evercore.SerializedEvent{ EventID: id, AggregateId: aggregateID, EventType: evtTypeName, State: state, Sequence: sequence, Reference: "", EventTime: eventTime })
    }
    return out, nil
}

func (s *PostgresStorageEngine) AdvanceSubscriptionCursor(tx evercore.StorageEngineTxInfo, ctx context.Context, id int64, lastEventId int64) error {
    db := s.maybeWrapTx(tx)
    q := New(db)
    return WrapError("failed to advance subscription cursor", q.AdvanceSubscriptionCursor(ctx, AdvanceSubscriptionCursorParams{ID: id, LastEventID: lastEventId}))
}

func (s *PostgresStorageEngine) GetMaxEventId(tx evercore.StorageEngineTxInfo, ctx context.Context) (int64, error) {
    db := s.maybeWrapTx(tx)
    q := New(db)
    id, err := q.GetMaxEventId(ctx)
    if err != nil { return 0, WrapError("failed to get max event id", err) }
    return id, nil
}

func (s *PostgresStorageEngine) GetFirstEventIdFromTimestamp(tx evercore.StorageEngineTxInfo, ctx context.Context, ts time.Time) (int64, error) {
    db := s.maybeWrapTx(tx)
    q := New(db)
    id, err := q.GetFirstEventIdFromTimestamp(ctx, ts)
    if err != nil { return 0, WrapError("failed to get first event id from timestamp", err) }
    return id, nil
}

// helpers
func valueOrZero(p *int64) int64 { if p == nil { return 0 }; return *p }
func valueOrEmpty(p *string) string { if p == nil { return "" }; return *p }
func valueOrZeroTime(p *time.Time) time.Time { if p == nil { return time.Time{} }; return *p }
func nullInt64(p *int64) sql.NullInt64 { if p == nil { return sql.NullInt64{} }; return sql.NullInt64{Int64:*p, Valid:true} }
func nullString(p *string) sql.NullString { if p == nil { return sql.NullString{} }; return sql.NullString{String:*p, Valid:true} }

func toBaseSubscription(row GetSubscriptionByNameRow) evercore.Subscription {
    sub := evercore.Subscription{ ID: row.ID, Name: row.Name, StartFrom: row.StartFrom, StartEventID: row.StartEventID, LastEventID: row.LastEventID, Active: row.Active }
    if row.AggregateTypeID.Valid { v := row.AggregateTypeID.Int64; sub.AggregateTypeID = &v }
    if row.EventTypeID.Valid { v := row.EventTypeID.Int64; sub.EventTypeID = &v }
    if row.AggregateKey.Valid { v := row.AggregateKey.String; sub.AggregateKey = &v }
    if row.StartTimestamp.Valid { v := row.StartTimestamp.Time; sub.StartTimestamp = &v }
    if row.LeaseOwner.Valid { v := row.LeaseOwner.String; sub.LeaseOwner = &v }
    if row.LeaseExpiresAt.Valid { v := row.LeaseExpiresAt.Time; sub.LeaseExpiresAt = &v }
    return sub
}
