package db

import "context"

type Connection interface {
	Ping(ctx context.Context) error
	Close()
}
