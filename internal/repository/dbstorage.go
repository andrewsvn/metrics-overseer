package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/andrewsvn/metrics-overseer/internal/db"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"go.uber.org/zap"
)

type PostgresDBStorage struct {
	conn    db.Connection
	sqrl    squirrel.StatementBuilderType
	retrier *retrying.Executor
	logger  *zap.SugaredLogger
}

func isPgErrorRetryable(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}

	switch pgErr.Code {
	// Class 08
	case pgerrcode.ConnectionException,
		pgerrcode.ConnectionDoesNotExist,
		pgerrcode.ConnectionFailure:
		return true
	}

	return false
}

func NewPostgresDBStorage(conn db.Connection, logger *zap.Logger, retryPolicy retrying.Policy) *PostgresDBStorage {
	pgLogger := logger.Sugar().With(zap.String("component", "postgres-storage"))

	retrier := retrying.NewExecutorBuilder(retryPolicy).
		WithLogger(pgLogger, "executing query").
		WithRetryablePredicate(isPgErrorRetryable).
		Executor()

	return &PostgresDBStorage{
		conn:    conn,
		sqrl:    squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
		retrier: retrier,
		logger:  pgLogger,
	}
}

func (pgs *PostgresDBStorage) SetGauge(ctx context.Context, id string, value float64) error {
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

	var res pgconn.CommandTag
	err = pgs.retrier.Run(func() error {
		pgs.logger.Debugw("set gauge query", "query", query, "args", args)
		var err error
		res, err = pgs.conn.Pool().Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to execute set gauge query: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return ErrIncorrectAccess
	}
	return nil
}

func (pgs *PostgresDBStorage) AddCounter(ctx context.Context, id string, delta int64) error {
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

	var res pgconn.CommandTag
	err = pgs.retrier.Run(func() error {
		pgs.logger.Debugw("add counter query", "query", query, "args", args)
		var err error
		res, err = pgs.conn.Pool().Exec(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to execute add counter query: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if res.RowsAffected() == 0 {
		return ErrIncorrectAccess
	}
	return nil
}

func (pgs *PostgresDBStorage) GetByID(ctx context.Context, id string) (*model.Metrics, error) {
	query, args, err := pgs.sqrl.Select("mtype", "delta", "value").
		From("metrics").
		Where(squirrel.Eq{"id": id}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to compose get metric query: %w", err)
	}

	var mtype string
	var delta *int64
	var value *float64

	err = pgs.retrier.Run(func() error {
		pgs.logger.Debugw("get metric by ID query", "query", query, "args", args)
		row := pgs.conn.Pool().QueryRow(ctx, query, args...)
		if err := row.Scan(&mtype, &delta, &value); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrMetricNotFound
			}
			return fmt.Errorf("failed to extract metric from DB row: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return model.NewMetrics(id, mtype, delta, value), nil
}

func (pgs *PostgresDBStorage) BatchUpdate(ctx context.Context, metrics []*model.Metrics) error {
	err := pgs.retrier.Run(func() error {
		return pgs.batchValidate(ctx, metrics)
	})
	if err != nil {
		return err
	}

	return pgs.retrier.Run(func() error {
		return pgs.batchSet(ctx, metrics)
	})
}

func (pgs *PostgresDBStorage) GetAllSorted(ctx context.Context) ([]*model.Metrics, error) {
	query, args, err := pgs.sqrl.Select("id", "mtype", "delta", "value").
		From("metrics").
		OrderBy("id ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to compose get all metrics query: %w", err)
	}

	var metrics []*model.Metrics

	err = pgs.retrier.Run(func() error {
		pgs.logger.Debugw("get all metrics query", "query", query, "args", args)
		rows, err := pgs.conn.Pool().Query(ctx, query, args...)
		if err != nil {
			return fmt.Errorf("failed to execute get all metrics query: %w", err)
		}
		defer rows.Close()

		metrics = make([]*model.Metrics, 0)
		for rows.Next() {
			var id string
			var mtype string
			var delta *int64
			var value *float64
			err := rows.Scan(&id, &mtype, &delta, &value)
			if err != nil {
				return fmt.Errorf("failed to extract metrics from DB row: %w", err)
			}
			metrics = append(metrics, model.NewMetrics(id, mtype, delta, value))
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("failed to extract all metrics from DB rows: %w", err)
		}
		return nil
	})

	if err != nil {
		return nil, err
	}
	return metrics, nil
}

func (pgs *PostgresDBStorage) SetAll(ctx context.Context, metrics []*model.Metrics) error {
	return pgs.BatchUpdate(ctx, metrics)
}

func (pgs *PostgresDBStorage) ResetAll(ctx context.Context) error {
	return pgs.retrier.Run(func() error {
		if _, err := pgs.conn.Pool().Exec(ctx, "TRUNCATE TABLE metrics"); err != nil {
			return fmt.Errorf("failed to truncate all metrics table: %w", err)
		}
		return nil
	})
}

func (pgs *PostgresDBStorage) Close() error {
	pgs.conn.Close()
	return nil
}

func (pgs *PostgresDBStorage) batchValidate(ctx context.Context, metrics []*model.Metrics) error {
	query, args, err := pgs.sqrl.Select("id", "mtype").From("metrics").ToSql()
	if err != nil {
		return fmt.Errorf("failed to compose get metric types query: %w", err)
	}

	pgs.logger.Debugw("batch get metric query", "query", query, "args", args)
	rows, err := pgs.conn.Pool().Query(ctx, query, args...)
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
				ErrIncorrectAccess, metric.ID, mtype, metric.MType)
		}
	}
	return nil
}

func (pgs *PostgresDBStorage) batchSet(ctx context.Context, metrics []*model.Metrics) error {
	tx, err := pgs.conn.Pool().BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("failed to initialize DB transaction: %w", err)
	}
	defer func() {
		err := tx.Rollback(ctx)
		if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
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

		if _, err := tx.Exec(ctx, query, args...); err != nil {
			return fmt.Errorf("failed to set metrics: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (pgs *PostgresDBStorage) Ping(ctx context.Context) error {
	return pgs.conn.Ping(ctx)
}
