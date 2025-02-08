package evercorepostgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kernelplex/evercore/pkg/evercore"
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

// Creates a new Postgres backed storage engine.
func NewPostgresStorageEngineWithConnection(connectionString string) (*PostgresStorageEngine, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, err
	}
	return &PostgresStorageEngine{
		db: db,
	}, nil
}

func (s *PostgresStorageEngine) GetTransactionInfo() (evercore.StorageEngineTxInfo, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (stor *PostgresStorageEngine) GetMaxKeyLength() int {
	return maxKeyLength
}

func (s *PostgresStorageEngine) GetEventTypeId(ctx context.Context, name string) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	queries := New(s.db)
	qtx := queries.WithTx(tx)
	eventTypeId, err := qtx.GetEventTypeIdByName(ctx, name)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	if eventTypeId != 0 {
		return eventTypeId, nil
	}

	eventTypeId, err = qtx.AddEventType(ctx, name)
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return eventTypeId, nil
}

func (s *PostgresStorageEngine) GetAggregateTypeId(ctx context.Context, aggregateTypeName string) (int64, error) {

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	queries := New(s.db)
	qtx := queries.WithTx(tx)
	aggregateTypeId, err := qtx.GetAggregateTypeIdByName(ctx, aggregateTypeName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	if aggregateTypeId != 0 {
		return aggregateTypeId, nil
	}

	aggregateTypeId, err = qtx.AddAggregateType(ctx, aggregateTypeName)
	if err != nil {
		return 0, err
	}
	err = tx.Commit()
	if err != nil {
		return 0, err
	}
	return aggregateTypeId, nil
}

func (s *PostgresStorageEngine) NewAggregate(tx evercore.StorageEngineTxInfo, ctx context.Context, aggregateTypeId int64) (int64, error) {
	db := tx.(*sql.Tx)
	queries := New(db)
	id, err := queries.AddAggregate(ctx, aggregateTypeId)
	return id, err
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
	return id, err
}

func (s *PostgresStorageEngine) GetAggregateById(ctx context.Context, aggregateTypeId int64, aggregateId int64) (int64, *string, error) {
	queries := New(s.db)
	params := GetAggregateByIdParams{
		AggregateTypeID: aggregateTypeId,
		AggregateID:     aggregateId,
	}
	result, err := queries.GetAggregateById(ctx, params)
	if err != nil {
		return 0, nil, err
	}

	var key *string
	if result.NaturalKey.Valid {
		key = &result.NaturalKey.String
	} else {
		key = nil
	}
	return result.ID, key, nil
}

func (s *PostgresStorageEngine) GetAggregateByKey(ctx context.Context, aggregateTypeId int64, naturalKey string) (int64, error) {
	queries := New(s.db)
	params := GetAggregateIdByNaturalKeyParams{
		AggregateTypeID: aggregateTypeId,
		NaturalKey:      sql.NullString{String: naturalKey, Valid: true},
	}
	id, err := queries.GetAggregateIdByNaturalKey(ctx, params)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (s *PostgresStorageEngine) GetAggregateTypes(ctx context.Context) ([]evercore.IdNamePair, error) {
	queries := New(s.db)

	aggregateTypes, err := queries.GetAggregateTypes(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return []evercore.IdNamePair{}, nil
	}

	if err != nil {
		return nil, err
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

func (s *PostgresStorageEngine) GetEventTypes(ctx context.Context) ([]evercore.IdNamePair, error) {
	queries := New(s.db)

	eventTypes, err := queries.GetEventTypes(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return []evercore.IdNamePair{}, nil
	}

	if err != nil {
		return nil, err
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

func (s *PostgresStorageEngine) GetSnapshotForAggregate(ctx context.Context, aggregateId int64) (*evercore.Snapshot, error) {
	queries := New(s.db)

	snapshotRow, err := queries.GetMostRecentSnapshot(ctx, aggregateId)

	// If we have no rows, just return nil
	if err != nil && errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	snapshot := evercore.Snapshot{
		AggregateId: snapshotRow.AggregateID,
		State:       snapshotRow.State,
		Sequence:    snapshotRow.Sequence,
	}
	return &snapshot, nil
}

func (s *PostgresStorageEngine) GetEventsForAggregate(ctx context.Context, aggregateId int64, afterSequence int64) ([]evercore.SerializedEvent, error) {
	queries := New(s.db)

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
			return err
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
			return err
		}
	}

	return nil
}
