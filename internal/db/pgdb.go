package db

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type PostgresDB struct {
	dbpool *pgxpool.Pool
}

func NewPostgresDB(ctx context.Context, cfg *servercfg.DatabaseConfig) (*PostgresDB, error) {
	dbc, err := pgxpool.New(ctx, strings.Trim(cfg.DBConnString, "\""))
	if err != nil {
		return nil, fmt.Errorf("failed to create database connection: %w", err)
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

func (pgdb *PostgresDB) Pool() *pgxpool.Pool {
	return pgdb.dbpool
}
