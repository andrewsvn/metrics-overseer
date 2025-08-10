package db

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Connection interface {
	Ping(ctx context.Context) error
	Pool() *pgxpool.Pool
	Close()
}
