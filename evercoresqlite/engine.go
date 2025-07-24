package evercoresqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

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
		// Cehck for contains UNIQUE constraint failed
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
	db, err := sql.Open("sqlite3", connectionString)
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
