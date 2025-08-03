package repository

import (
	"context"
	"errors"
	"fmt"
	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type PostgresDBStorage struct {
	pool   *pgxpool.Pool
	logger *zap.SugaredLogger
	sqrl   squirrel.StatementBuilderType
}

func NewPostgresDBStorage(pool *pgxpool.Pool, logger *zap.Logger) *PostgresDBStorage {
	pgLogger := logger.Sugar().With(zap.String("component", "postgres-storage"))
	return &PostgresDBStorage{
		pool:   pool,
		logger: pgLogger,
		sqrl:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}
}

func (pgs *PostgresDBStorage) GetGauge(id string) (*float64, error) {
	m, err := pgs.GetByID(id)
	if err != nil {
		return nil, err
	}
	return m.GetGauge()
}

func (pgs *PostgresDBStorage) SetGauge(id string, value float64) error {
	query, args, err := pgs.sqrl.Insert("metrics").
		Columns("id", "mtype", "value").
		Values(id, model.Gauge, value).
		Suffix(`
            ON CONFLICT (id) DO UPDATE
            SET value = EXCLUDED.value
            WHERE metrics.mtype = '` + model.Gauge + `'`).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to compose set metric query: %w", err)
	}

	pgs.logger.Debugw("set gauge query", "query", query, "args", args)
	res, err := pgs.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute set gauge query: %w", err)
	}

	if res.RowsAffected() == 0 {
		return model.ErrIncorrectAccess
	}
	return nil
}

func (pgs *PostgresDBStorage) GetCounter(id string) (*int64, error) {
	m, err := pgs.GetByID(id)
	if err != nil {
		return nil, err
	}
	return m.GetCounter()
}

func (pgs *PostgresDBStorage) AddCounter(id string, delta int64) error {
	query, args, err := pgs.sqrl.Insert("metrics").
		Columns("id", "mtype", "delta").
		Values(id, model.Counter, delta).
		Suffix(`
            ON CONFLICT (id) DO UPDATE
            SET delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta
            WHERE metrics.mtype = '` + model.Counter + `'`).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to compose set metric query: %w", err)
	}

	pgs.logger.Debugw("add counter query", "query", query, "args", args)
	res, err := pgs.pool.Exec(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute add counter query: %w", err)
	}

	if res.RowsAffected() == 0 {
		return model.ErrIncorrectAccess
	}
	return nil
}

func (pgs *PostgresDBStorage) GetByID(id string) (*model.Metrics, error) {
	query, args, err := pgs.sqrl.Select("mtype", "delta", "value").
		From("metrics").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to compose get metric query: %w", err)
	}

	pgs.logger.Debugw("get metric by ID query", "query", query, "args", args)
	row := pgs.pool.QueryRow(context.Background(), query, args...)
	var mtype string
	var delta *int64
	var value *float64
	if err := row.Scan(&mtype, &delta, &value); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMetricNotFound
		}
		return nil, fmt.Errorf("failed to extract metric from DB row: %w", err)
	}

	return model.NewMetrics(id, mtype, delta, value), nil
}

func (pgs *PostgresDBStorage) BatchUpdate(metrics []*model.Metrics) error {
	err := pgs.batchValidate(metrics)
	if err != nil {
		return err
	}
	return pgs.batchSet(metrics)
}

func (pgs *PostgresDBStorage) GetAllSorted() ([]*model.Metrics, error) {
	query, args, err := pgs.sqrl.Select("id", "mtype", "delta", "value").
		From("metrics").
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to compose get all metrics query: %w", err)
	}

	pgs.logger.Debugw("get all metrics query", "query", query, "args", args)
	rows, err := pgs.pool.Query(context.Background(), query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute get all metrics query: %w", err)
	}
	defer rows.Close()

	metrics := make([]*model.Metrics, 0)
	for rows.Next() {
		var id string
		var mtype string
		var delta *int64
		var value *float64
		err := rows.Scan(&id, &mtype, &delta, &value)
		if err != nil {
			return nil, fmt.Errorf("failed to extract metrics from DB row: %w", err)
		}
		metrics = append(metrics, model.NewMetrics(id, mtype, delta, value))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to extract all metrics from DB rows: %w", err)
	}
	return metrics, nil
}

func (pgs *PostgresDBStorage) SetAll(metrics []*model.Metrics) error {
	return pgs.BatchUpdate(metrics)
}

func (pgs *PostgresDBStorage) ResetAll() error {
	if _, err := pgs.pool.Exec(context.Background(), "TRUNCATE TABLE metrics"); err != nil {
		return fmt.Errorf("failed to truncate all metrics table: %w", err)
	}
	return nil
}

func (pgs *PostgresDBStorage) Close() error {
	pgs.pool.Close()
	return nil
}

func (pgs *PostgresDBStorage) batchValidate(metrics []*model.Metrics) error {
	query, args, err := pgs.sqrl.Select("id", "mtype").From("metrics").ToSql()
	if err != nil {
		return fmt.Errorf("failed to compose get metric types query: %w", err)
	}

	pgs.logger.Debugw("batch get metric query", "query", query, "args", args)
	rows, err := pgs.pool.Query(context.Background(), query, args...)
	if err != nil {
		return fmt.Errorf("failed to get metric types: %w", err)
	}
	defer rows.Close()

	mtypes := make(map[string]string)
	for rows.Next() {
		var id string
		var mtype string
		if err := rows.Scan(&id, &mtype); err != nil {
			return fmt.Errorf("failed to read metric type: %w", err)
		}
		mtypes[id] = mtype
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to read metric types: %w", err)
	}

	for _, metric := range metrics {
		mtype, ok := mtypes[metric.ID]
		if ok && metric.MType != mtype {
			return fmt.Errorf("%w: for metric id=%s expected=%s, actual=%s",
				model.ErrIncorrectAccess, metric.ID, mtype, metric.MType)
		}
	}
	return nil
}

func (pgs *PostgresDBStorage) batchSet(metrics []*model.Metrics) error {
	tx, err := pgs.pool.BeginTx(context.Background(), pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to initialize DB transaction: %w", err)
	}
	defer func() {
		err := tx.Rollback(context.Background())
		if err != nil {
			pgs.logger.Warnw("failed to rollback transaction",
				"err", err,
			)
		}
	}()

	for _, m := range metrics {
		query, args, err := pgs.sqrl.Insert("metrics").
			Columns("id", "mtype", "delta", "value").
			Values(m.ID, m.MType, m.Delta, m.Value).
			Suffix(`
				ON CONFLICT (id) DO UPDATE
				SET mtype = EXCLUDED.mtype,
					delta = COALESCE(metrics.delta, 0) + EXCLUDED.delta,
					value = EXCLUDED.value
        	`).
			ToSql()
		if err != nil {
			return fmt.Errorf("failed to compose set metrics query: %w", err)
		}

		if _, err := tx.Exec(context.Background(), query, args...); err != nil {
			return fmt.Errorf("failed to set metrics: %w", err)
		}
	}

	return tx.Commit(context.Background())
}
