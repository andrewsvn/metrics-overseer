package db

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/dbcfg"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDB struct {
	dbpool *pgxpool.Pool
}

func NewPostgresDB(ctx context.Context, cfg *dbcfg.Config) (*PostgresDB, error) {
	dbc, err := pgxpool.New(ctx, cfg.DBConnString)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &PostgresDB{
		dbpool: dbc,
	}, nil
}

func (pgdb *PostgresDB) Close() {
	if pgdb.dbpool != nil {
		pgdb.dbpool.Close()
	}
}

func (pgdb *PostgresDB) Ping(ctx context.Context) error {
	return pgdb.dbpool.Ping(ctx)
}
