package evercoresqlite

import (
    "context"
    "database/sql"
    "errors"
    "fmt"
    "strings"
    "time"

	"github.com/kernelplex/evercore/base"
)

const maxKeyLength = 64

type SqliteStorageEngine struct {
	db *sql.DB
}

func WrapError(msg string, err error) error {
	storageErr := evercore.NewStorageEngineError(msg, err)

	// Check for SQLite unique constraint violation
	if err != nil {
		// Check for contains UNIQUE constraint failed
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			storageErr.ErrorType = evercore.ErrorTypeConstraintViolation
		} else if errors.Is(err, sql.ErrNoRows) {
			storageErr.ErrorType = evercore.ErrorNotFound
		}
	}
	return storageErr
}

func WrapErrorf(msg string, err error, args ...any) error {
	storageErr := evercore.NewStorageEngineError(fmt.Sprintf(msg, args...), err)
	// Check for SQLite unique constraint violation
	if err != nil {
		if err.Error() == "UNIQUE constraint failed" {
			storageErr.ErrorType = evercore.ErrorTypeConstraintViolation
		} else if errors.Is(err, sql.ErrNoRows) {
			storageErr.ErrorType = evercore.ErrorNotFound
		}
	}
	return storageErr
}

// Creates a new Sqlite3 backed storage engine.
func NewSqliteStorageEngine(db *sql.DB) *SqliteStorageEngine {
	return &SqliteStorageEngine{
		db: db,
	}
}

// Creates a new Sqlite backed storage engine connecting to the connection string.
func NewSqliteStorageEngineWithConnection(connectionString string) (*SqliteStorageEngine, error) {
	db, err := sql.Open("sqlite", connectionString)
	if err != nil {
		return nil, WrapError("failed to begin transaction", err)
	}
	return &SqliteStorageEngine{
		db: db,
	}, nil
}

func (stor *SqliteStorageEngine) GetMaxKeyLength() int {
	return maxKeyLength
}

func (s *SqliteStorageEngine) GetTransactionInfo() (evercore.StorageEngineTxInfo, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, WrapError("failed to get snapshot for aggregate", err)
	}
	return tx, nil
}

func (s *SqliteStorageEngine) GetEventTypeId(tx evercore.StorageEngineTxInfo, ctx context.Context, name string) (int64, error) {
	db := s.maybeWrapTx(tx)
	qtx := New(db)

	eventTypeId, err := qtx.GetEventTypeIdByName(ctx, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, WrapError("failed to get event type id", err)
	}

	if eventTypeId != 0 {
		return eventTypeId, nil
	}

	eventTypeId, err = qtx.AddEventType(ctx, name)
	if err != nil {
		return 0, WrapError("failed to add event type", err)
	}
	return eventTypeId, nil
}

func (s *SqliteStorageEngine) GetAggregateTypeId(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeName string) (int64, error) {
	db := s.maybeWrapTx(tx)
	qtx := New(db)
	aggregateTypeId, err := qtx.GetAggregateTypeIdByName(ctx, aggregateTypeName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, WrapError("failed to get aggregate type id", err)
	}

	if aggregateTypeId != 0 {
		return aggregateTypeId, nil
	}

	aggregateTypeId, err = qtx.AddAggregateType(ctx, aggregateTypeName)
	if err != nil {
		return 0, WrapError("failed to add aggregate type", err)
	}
	return aggregateTypeId, nil
}

func (s *SqliteStorageEngine) maybeWrapTx(tx evercore.StorageEngineTxInfo) DBTX {
	if tx == nil {
		return s.db
	}
	return tx.(*sql.Tx)
}

func (s *SqliteStorageEngine) NewAggregate(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64) (int64, error) {
	db := s.maybeWrapTx(tx)
	queries := New(db)
	id, err := queries.AddAggregate(ctx, aggregateTypeId)
	if err != nil {
		return 0, WrapError("failed to create new aggregate", err)
	}
	return id, nil
}

func (s *SqliteStorageEngine) NewAggregateWithKey(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error) {
	if len(naturalKey) > maxKeyLength {
		return 0, evercore.ErrorKeyExceedsMaximumLength
	}
	db := s.maybeWrapTx(tx)

	queries := New(db)
	params := AddAggregateWithNaturalKeyParams{
		AggregateTypeID: aggregateTypeId,
		NaturalKey:      sql.NullString{String: naturalKey, Valid: true},
	}
	id, err := queries.AddAggregateWithNaturalKey(ctx, params)
	if err != nil {
		return 0, WrapError("failed to create aggregate with key", err)
	}
	return id, nil
}

func (s *SqliteStorageEngine) ChangeAggregateNaturalKey(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateId int64, naturalKey string) error {

	if len(naturalKey) > maxKeyLength {
		return evercore.ErrorKeyExceedsMaximumLength
	}
	db := s.maybeWrapTx(tx)
	queries := New(db)
	params := ChangeAggregateNaturalKeyParams{
		AggregateID: aggregateId,
		NaturalKey:  sql.NullString{String: naturalKey, Valid: true},
	}
	err := queries.ChangeAggregateNaturalKey(ctx, params)
	if err != nil {
		return WrapError("failed to change aggregate natural key", err)
	}
	return nil

}

func (s *SqliteStorageEngine) GetAggregateById(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, aggregateId int64) (int64, *string, error) {
	db := s.maybeWrapTx(tx)
	queries := New(db)
	params := GetAggregateByIdParams{
		AggregateTypeID: aggregateTypeId,
		AggregateID:     aggregateId,
	}
	result, err := queries.GetAggregateById(ctx, params)
	if err != nil {
		return 0, nil, WrapError("failed to get aggregate by id", err)
	}

	var key *string
	if result.NaturalKey.Valid {
		key = &result.NaturalKey.String
	} else {
		key = nil
	}
	return result.ID, key, nil
}

func (s *SqliteStorageEngine) GetAggregateByKey(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error) {
	db := s.maybeWrapTx(tx)
	queries := New(db)
	params := GetAggregateIdByNaturalKeyParams{
		AggregateTypeID: aggregateTypeId,
		NaturalKey:      sql.NullString{String: naturalKey, Valid: true},
	}
	id, err := queries.GetAggregateIdByNaturalKey(ctx, params)
	if err != nil {
		return 0, WrapError("failed to get aggregate by key", err)
	}

	return id, nil
}

func (s *SqliteStorageEngine) GetOrCreateAggregateByKey(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (bool, int64, error) {
	if len(naturalKey) > maxKeyLength {
		return false, 0, evercore.ErrorKeyExceedsMaximumLength
	}

	db := s.maybeWrapTx(tx)
	queries := New(db)

	// First try to get existing aggregate
	params := GetAggregateIdByNaturalKeyParams{
		AggregateTypeID: aggregateTypeId,
		NaturalKey:      sql.NullString{String: naturalKey, Valid: true},
	}
	aggregateId, err := queries.GetAggregateIdByNaturalKey(ctx, params)
	if err == nil {
		return false, aggregateId, nil
	}

	// If not found, create new aggregate
	if errors.Is(err, sql.ErrNoRows) {
		createParams := AddAggregateWithNaturalKeyParams{
			AggregateTypeID: aggregateTypeId,
			NaturalKey:      sql.NullString{String: naturalKey, Valid: true},
		}
		aggregateId, err = queries.AddAggregateWithNaturalKey(ctx, createParams)
		if err != nil {
			return false, 0, WrapError("failed to create aggregate with key", err)
		}
		return true, aggregateId, nil
	}

	return false, 0, WrapError("failed to get or create aggregate by key", err)
}

func (s *SqliteStorageEngine) GetEventTypes(tx evercore.StorageEngineTxInfo, ctx context.Context) ([]evercore.IdNamePair, error) {
	db := s.maybeWrapTx(tx)
	queries := New(db)

	eventTypes, err := queries.GetEventTypes(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []evercore.IdNamePair{}, nil
		}
		return nil, WrapError("failed to get event types in GetEventTypes", err)
	}

	var localEventTypes = make([]evercore.IdNamePair, len(eventTypes))
	for idx, eventType := range eventTypes {
		localEventTypes[idx] = evercore.IdNamePair{
			Id:   eventType.ID,
			Name: eventType.Name,
		}
	}
	return localEventTypes, nil
}

func (s *SqliteStorageEngine) GetAggregateTypes(tx evercore.StorageEngineTxInfo, ctx context.Context) ([]evercore.IdNamePair, error) {
	db := s.maybeWrapTx(tx)
	queries := New(db)

	aggregateTypes, err := queries.GetAggregateTypes(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []evercore.IdNamePair{}, nil
		}
		return nil, WrapError("failed to get aggregate types in GetAggregateTypes", err)
	}

	var localAggregateTypes = make([]evercore.IdNamePair, len(aggregateTypes))
	for idx, aggregateType := range aggregateTypes {
		localAggregateTypes[idx] = evercore.IdNamePair{
			Id:   aggregateType.ID,
			Name: aggregateType.Name,
		}
	}
	return localAggregateTypes, nil
}

func (s *SqliteStorageEngine) GetSnapshotForAggregate(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateId int64) (*evercore.Snapshot, error) {
	db := s.maybeWrapTx(tx)
	queries := New(db)

	snapshotRow, err := queries.GetMostRecentSnapshot(ctx, aggregateId)

	// If we have no rows, just return nil
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, WrapError("failed to get events for aggregate", err)
	}

	snapshot := evercore.Snapshot{
		AggregateId: snapshotRow.AggregateID,
		State:       snapshotRow.State,
		Sequence:    snapshotRow.Sequence,
	}
	return &snapshot, nil
}

func (s *SqliteStorageEngine) GetEventsForAggregate(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateId int64, afterSequence int64) ([]evercore.SerializedEvent, error) {
	db := s.maybeWrapTx(tx)
	queries := New(db)

	params := GetEventsForAggregateParams{
		AggregateID:   aggregateId,
		AfterSequence: afterSequence,
	}

	eventRows, err := queries.GetEventsForAggregate(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []evercore.SerializedEvent{}, nil
		}
		return nil, WrapError("failed to get events for aggregate in GetEventsForAggregate", err)
	}

	resultEvents := make([]evercore.SerializedEvent, 0, len(eventRows))

	for _, eventRow := range eventRows {
		event := evercore.SerializedEvent{
			AggregateId: aggregateId,
			EventType:   eventRow.EventType,
			Sequence:    eventRow.Sequence,
			Reference:   eventRow.Reference,
			State:       eventRow.State,
			EventTime:   eventRow.EventTime,
		}
		resultEvents = append(resultEvents, event)
	}
	return resultEvents, nil
}

func (s *SqliteStorageEngine) WriteState(tx evercore.StorageEngineTxInfo, ctx context.Context, events []evercore.StorageEngineEvent, snapshots evercore.SnapshotSlice) error {
	db := s.maybeWrapTx(tx)

	queries := New(db)

	// var addEventParams = AddEventParams{}
	for _, event := range events {
		addEventParams := AddEventParams{
			AggregateID: event.AggregateID,
			Sequence:    event.Sequence,
			EventTypeID: event.EventTypeID,
			State:       event.State,
			EventTime:   event.EventTime,
			Reference:   event.Reference,
		}

		err := queries.AddEvent(ctx, addEventParams)
		if err != nil {
			return WrapError("failed to add event", err)
		}
	}

	for _, snapshot := range snapshots {
		addSnapshotParams := AddSnapshotParams{
			AggregateID: snapshot.AggregateId,
			Sequence:    snapshot.Sequence,
			State:       snapshot.State,
		}
		err := queries.AddSnapshot(ctx, addSnapshotParams)
		if err != nil {
			return WrapError("failed to add snapshot", err)
		}
	}

	return nil
}

func (s *SqliteStorageEngine) Close() error {
    return s.db.Close()
}

// ---------------- Subscriptions ----------------

func (s *SqliteStorageEngine) UpsertSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, aggregateTypeId *int64, eventTypeId *int64, aggregateKey *string, startFrom string, startEventId int64, startTimestamp *time.Time) (int64, error) {
    db := s.maybeWrapTx(tx)
    // UPSERT without modifying last_event_id
    _, err := db.ExecContext(ctx, `
        INSERT INTO subscriptions (name, aggregate_type_id, event_type_id, aggregate_key, start_from, start_event_id, start_timestamp)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(name) DO UPDATE SET
          aggregate_type_id=excluded.aggregate_type_id,
          event_type_id=excluded.event_type_id,
          aggregate_key=excluded.aggregate_key,
          start_from=excluded.start_from,
          start_event_id=excluded.start_event_id,
          start_timestamp=excluded.start_timestamp,
          updated_at=CURRENT_TIMESTAMP
    `, name, aggregateTypeId, eventTypeId, aggregateKey, startFrom, startEventId, startTimestamp)
    if err != nil {
        return 0, WrapError("failed to upsert subscription", err)
    }
    var id int64
    err = db.QueryRowContext(ctx, `SELECT id FROM subscriptions WHERE name = ?`, name).Scan(&id)
    if err != nil {
        return 0, WrapError("failed to fetch subscription id", err)
    }
    return id, nil
}

func (s *SqliteStorageEngine) AddSubscriptionEventType(tx evercore.StorageEngineTxInfo, ctx context.Context, subscriptionId int64, eventTypeId int64) error {
    db := s.maybeWrapTx(tx)
    _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO subscription_event_types (subscription_id, event_type_id) VALUES (?, ?)`, subscriptionId, eventTypeId)
    if err != nil {
        return WrapError("failed to add subscription event type", err)
    }
    return nil
}

func (s *SqliteStorageEngine) GetSubscriptionByName(tx evercore.StorageEngineTxInfo, ctx context.Context, name string) (*evercore.Subscription, error) {
    db := s.maybeWrapTx(tx)
    row := db.QueryRowContext(ctx, `
        SELECT id, name, aggregate_type_id, event_type_id, aggregate_key,
               start_from, start_event_id, start_timestamp,
               last_event_id, active, lease_owner, lease_expires_at
        FROM subscriptions WHERE name = ?
    `, name)

    var sub evercore.Subscription
    var aggTypeId sql.NullInt64
    var evtTypeId sql.NullInt64
    var aggKey sql.NullString
    var startTs sql.NullTime
    var leaseOwner sql.NullString
    var leaseExp sql.NullTime

    if err := row.Scan(&sub.ID, &sub.Name, &aggTypeId, &evtTypeId, &aggKey, &sub.StartFrom, &sub.StartEventID, &startTs, &sub.LastEventID, &sub.Active, &leaseOwner, &leaseExp); err != nil {
        return nil, WrapError("failed to get subscription", err)
    }
    if aggTypeId.Valid {
        v := aggTypeId.Int64
        sub.AggregateTypeID = &v
    }
    if evtTypeId.Valid {
        v := evtTypeId.Int64
        sub.EventTypeID = &v
    }
    if aggKey.Valid {
        v := aggKey.String
        sub.AggregateKey = &v
    }
    if startTs.Valid {
        v := startTs.Time
        sub.StartTimestamp = &v
    }
    if leaseOwner.Valid {
        v := leaseOwner.String
        sub.LeaseOwner = &v
    }
    if leaseExp.Valid {
        v := leaseExp.Time
        sub.LeaseExpiresAt = &v
    }
    return &sub, nil
}

func (s *SqliteStorageEngine) SetSubscriptionActive(tx evercore.StorageEngineTxInfo, ctx context.Context, id int64, active bool) error {
    db := s.maybeWrapTx(tx)
    _, err := db.ExecContext(ctx, `UPDATE subscriptions SET active = ?, updated_at=CURRENT_TIMESTAMP WHERE id = ?`, active, id)
    if err != nil {
        return WrapError("failed to set subscription active", err)
    }
    return nil
}

func (s *SqliteStorageEngine) ClaimSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, owner string, lease time.Duration) (bool, error) {
    db := s.maybeWrapTx(tx)
    leaseUntil := time.Now().Add(lease)
    res, err := db.ExecContext(ctx, `
        UPDATE subscriptions
        SET lease_owner = ?, lease_expires_at = ?, updated_at=CURRENT_TIMESTAMP
        WHERE name = ? AND active = 1 AND (lease_owner IS NULL OR lease_expires_at < ?)
    `, owner, leaseUntil, name, time.Now())
    if err != nil {
        return false, WrapError("failed to claim subscription", err)
    }
    n, _ := res.RowsAffected()
    return n == 1, nil
}

func (s *SqliteStorageEngine) RenewSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, owner string, lease time.Duration) (bool, error) {
    db := s.maybeWrapTx(tx)
    leaseUntil := time.Now().Add(lease)
    res, err := db.ExecContext(ctx, `
        UPDATE subscriptions
        SET lease_expires_at = ?, updated_at=CURRENT_TIMESTAMP
        WHERE name = ? AND lease_owner = ? AND active = 1
    `, leaseUntil, name, owner)
    if err != nil {
        return false, WrapError("failed to renew subscription", err)
    }
    n, _ := res.RowsAffected()
    return n == 1, nil
}

func (s *SqliteStorageEngine) ReleaseSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, name string, owner string) error {
    db := s.maybeWrapTx(tx)
    _, err := db.ExecContext(ctx, `
        UPDATE subscriptions SET lease_owner = NULL, lease_expires_at = NULL, updated_at=CURRENT_TIMESTAMP
        WHERE name = ? AND lease_owner = ?
    `, name, owner)
    if err != nil {
        return WrapError("failed to release subscription", err)
    }
    return nil
}

func (s *SqliteStorageEngine) hasMultiEventTypes(ctx context.Context, db DBTX, subscriptionId int64) (bool, error) {
    row := db.QueryRowContext(ctx, `SELECT 1 FROM subscription_event_types WHERE subscription_id = ? LIMIT 1`, subscriptionId)
    var one int
    err := row.Scan(&one)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return false, nil
        }
        return false, err
    }
    return true, nil
}

func (s *SqliteStorageEngine) GetEventsForSubscription(tx evercore.StorageEngineTxInfo, ctx context.Context, sub *evercore.Subscription, limit int) ([]evercore.SerializedEvent, error) {
    db := s.maybeWrapTx(tx)

    multi, err := s.hasMultiEventTypes(ctx, db, sub.ID)
    if err != nil {
        return nil, WrapError("failed to check subscription event types", err)
    }

    // Build base query
    q := `SELECT id, aggregate_id, natural_key, sequence, aggregate_type_id, aggregate_type, event_type_id, event_type, event_time, state
          FROM event_log WHERE id > ?`
    args := []any{sub.LastEventID}

    if sub.AggregateTypeID != nil {
        q += " AND aggregate_type_id = ?"
        args = append(args, *sub.AggregateTypeID)
    }
    if sub.AggregateKey != nil {
        q += " AND natural_key = ?"
        args = append(args, *sub.AggregateKey)
    }
    if multi {
        q += " AND event_type_id IN (SELECT event_type_id FROM subscription_event_types WHERE subscription_id = ?)"
        args = append(args, sub.ID)
    } else if sub.EventTypeID != nil {
        q += " AND event_type_id = ?"
        args = append(args, *sub.EventTypeID)
    }

    q += " ORDER BY id ASC LIMIT ?"
    args = append(args, limit)

    rows, err := db.QueryContext(ctx, q, args...)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return []evercore.SerializedEvent{}, nil
        }
        return nil, WrapError("failed to query events for subscription", err)
    }
    defer rows.Close()

    results := make([]evercore.SerializedEvent, 0, limit)
    for rows.Next() {
        var id, aggregateID, sequence int64
        var naturalKey sql.NullString
        var aggTypeId, evtTypeId int64
        var aggTypeName, evtTypeName string
        var eventTime time.Time
        var state string
        if err := rows.Scan(&id, &aggregateID, &naturalKey, &sequence, &aggTypeId, &aggTypeName, &evtTypeId, &evtTypeName, &eventTime, &state); err != nil {
            return nil, WrapError("failed to scan events for subscription", err)
        }
        se := evercore.SerializedEvent{
            EventID:    id,
            AggregateId: aggregateID,
            EventType:   evtTypeName,
            State:       state,
            Sequence:    sequence,
            Reference:   "",
            EventTime:   eventTime,
        }
        if naturalKey.Valid {
            // keep available via Reference if desired; but SerializedEvent has no natural key field; leave as-is
        }
        results = append(results, se)
    }
    return results, nil
}

func (s *SqliteStorageEngine) AdvanceSubscriptionCursor(tx evercore.StorageEngineTxInfo, ctx context.Context, id int64, lastEventId int64) error {
    db := s.maybeWrapTx(tx)
    _, err := db.ExecContext(ctx, `UPDATE subscriptions SET last_event_id = ?, updated_at=CURRENT_TIMESTAMP WHERE id = ?`, lastEventId, id)
    if err != nil {
        return WrapError("failed to advance subscription cursor", err)
    }
    return nil
}

func (s *SqliteStorageEngine) GetMaxEventId(tx evercore.StorageEngineTxInfo, ctx context.Context) (int64, error) {
    db := s.maybeWrapTx(tx)
    var id sql.NullInt64
    if err := db.QueryRowContext(ctx, `SELECT MAX(id) FROM events`).Scan(&id); err != nil {
        return 0, WrapError("failed to get max event id", err)
    }
    if !id.Valid {
        return 0, nil
    }
    return id.Int64, nil
}

func (s *SqliteStorageEngine) GetFirstEventIdFromTimestamp(tx evercore.StorageEngineTxInfo, ctx context.Context, ts time.Time) (int64, error) {
    db := s.maybeWrapTx(tx)
    var id sql.NullInt64
    if err := db.QueryRowContext(ctx, `SELECT id FROM events WHERE event_time >= ? ORDER BY id ASC LIMIT 1`, ts).Scan(&id); err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            return 0, nil
        }
        return 0, WrapError("failed to get first event id from timestamp", err)
    }
    if !id.Valid {
        return 0, nil
    }
    return id.Int64, nil
}
