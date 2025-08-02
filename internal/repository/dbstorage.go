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
            SET value = COALESCE(metrics.value, 0)
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

func (pgs *PostgresDBStorage) SetByID(metric *model.Metrics) error {
	query, args, err := pgs.sqrl.Insert("metrics").
		Columns("id", "mtype", "delta", "value").
		Values(metric.ID, metric.MType, metric.Delta, metric.Value).
		Suffix(`
            ON CONFLICT (id) DO UPDATE
            SET mtype = EXCLUDED.mtype,
                delta = EXCLUDED.delta,
                value = EXCLUDED.value
        `).
		ToSql()
	if err != nil {
		return fmt.Errorf("failed to compose set metric query: %w", err)
	}

	pgs.logger.Debugw("set metric by ID query", "query", query, "args", args)
	if _, err := pgs.pool.Exec(context.Background(), query, args...); err != nil {
		return fmt.Errorf("failed to set metric: %w", err)
	}
	return nil
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
	builder := pgs.sqrl.Insert("metrics").Columns("id", "mtype", "delta", "value")
	for _, m := range metrics {
		builder = builder.Values(m.ID, m.MType, m.Delta, m.Value)
	}
	query, args, err := builder.Suffix(`
            ON CONFLICT (id) DO UPDATE
            SET mtype = EXCLUDED.mtype,
                delta = EXCLUDED.delta,
                value = EXCLUDED.value
        `).ToSql()
	if err != nil {
		return fmt.Errorf("failed to compose set all metrics query: %w", err)
	}

	pgs.logger.Debugw("set all metrics query", "query", query, "args", args)
	if _, err := pgs.pool.Exec(context.Background(), query, args...); err != nil {
		return fmt.Errorf("failed to set all metrics: %w", err)
	}
	return nil
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
