package evercorepostgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/kernelplex/evercore/base"
)

const maxKeyLength = 64

type PostgresStorageEngine struct {
	db *sql.DB
}

func NewPostgresStorageEngine(db *sql.DB) *PostgresStorageEngine {
	return &PostgresStorageEngine{
		db: db,
	}
}

func WrapError(msg string, err error) error {
	storageErr := evercore.NewStorageEngineError(msg, err)

	// Check for PostgreSQL unique constraint violation
	if err != nil {
		// Check for contains duplicate key value violates unique constraint
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			storageErr.ErrorType = evercore.ErrorTypeConstraintViolation
		} else if errors.Is(err, sql.ErrNoRows) {
			storageErr.ErrorType = evercore.ErrorNotFound
		}
	}
	return storageErr
}

func WrapErrorf(msg string, err error, args ...any) error {
	storageErr := evercore.NewStorageEngineError(fmt.Sprintf(msg, args...), err)
	// Check for PostgreSQL unique constraint violation
	if err != nil {
		if err.Error() == "pq: duplicate key value violates unique constraint" {
			storageErr.ErrorType = evercore.ErrorTypeConstraintViolation
		}
	}
	return storageErr
}

// Creates a new Postgres backed storage engine.
func NewPostgresStorageEngineWithConnection(connectionString string) (*PostgresStorageEngine, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, evercore.NewStorageEngineError("failed to get aggregate by id", err)
	}
	return &PostgresStorageEngine{
		db: db,
	}, nil
}

func (s *PostgresStorageEngine) GetTransactionInfo() (evercore.StorageEngineTxInfo, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, WrapError("failed to get snapshot for aggregate", err)
	}
	return tx, nil
}

func (stor *PostgresStorageEngine) GetMaxKeyLength() int {
	return maxKeyLength
}

func (s *PostgresStorageEngine) GetEventTypeId(tx evercore.StorageEngineTxInfo, ctx context.Context, name string) (int64, error) {
	db := tx.(*sql.Tx)
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

func (s *PostgresStorageEngine) GetAggregateTypeId(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeName string) (int64, error) {
	db := tx.(*sql.Tx)
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

func (s *PostgresStorageEngine) NewAggregate(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64) (int64, error) {
	db := tx.(*sql.Tx)
	queries := New(db)
	id, err := queries.AddAggregate(ctx, aggregateTypeId)
	if err != nil {
		return 0, WrapError("failed to create new aggregate", err)
	}
	return id, nil
}

func (s *PostgresStorageEngine) NewAggregateWithKey(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error) {

	if len(naturalKey) > maxKeyLength {
		return 0, evercore.ErrorKeyExceedsMaximumLength
	}

	db := tx.(*sql.Tx)
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

func (s *PostgresStorageEngine) GetAggregateById(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, aggregateId int64) (int64, *string, error) {
	db := tx.(*sql.Tx)
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

func (s *PostgresStorageEngine) GetAggregateByKey(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error) {
	db := tx.(*sql.Tx)

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

func (s *PostgresStorageEngine) GetOrCreateAggregateByKey(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64, naturalKey string) (bool, int64, error) {
	if len(naturalKey) > maxKeyLength {
		return false, 0, evercore.ErrorKeyExceedsMaximumLength
	}

	db := tx.(*sql.Tx)
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

func (s *PostgresStorageEngine) GetAggregateTypes(tx evercore.StorageEngineTxInfo, ctx context.Context) ([]evercore.IdNamePair, error) {
	db := tx.(*sql.Tx)
	queries := New(db)

	aggregateTypes, err := queries.GetAggregateTypes(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return []evercore.IdNamePair{}, nil
	}

	if err != nil {
		return nil, WrapError("failed to get events for aggregate", err)
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

func (s *PostgresStorageEngine) GetEventTypes(tx evercore.StorageEngineTxInfo, ctx context.Context) ([]evercore.IdNamePair, error) {
	db := tx.(*sql.Tx)
	queries := New(db)

	eventTypes, err := queries.GetEventTypes(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return []evercore.IdNamePair{}, nil
	}

	if err != nil {
		return nil, WrapError("failed to get snapshot for aggregate", err)
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

func (s *PostgresStorageEngine) GetSnapshotForAggregate(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateId int64) (*evercore.Snapshot, error) {
	db := tx.(*sql.Tx)
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

func (s *PostgresStorageEngine) GetEventsForAggregate(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateId int64, afterSequence int64) ([]evercore.SerializedEvent, error) {
	db := tx.(*sql.Tx)
	queries := New(db)

	params := GetEventsForAggregateParams{
		AggregateID:   aggregateId,
		AfterSequence: afterSequence,
	}

	eventRows, err := queries.GetEventsForAggregate(ctx, params)

	// If we have no rows, just return an empty array.
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return []evercore.SerializedEvent{}, nil
	}

	if err != nil {
		return nil, err
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

func (s *PostgresStorageEngine) WriteState(tx evercore.StorageEngineTxInfo, ctx context.Context, events []evercore.StorageEngineEvent, snapshots evercore.SnapshotSlice) error {
	db := tx.(*sql.Tx)

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

func (s *PostgresStorageEngine) Close() error {
	return s.db.Close()
}
