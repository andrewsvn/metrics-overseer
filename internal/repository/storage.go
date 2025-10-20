package repository

import (
	"context"
	"errors"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

// all methods can return error in case when some internal storage problem occurs
// or method is unsupported for chosen metric
// error is not returned if data is not found in get methods

type Storage interface {
	SetGauge(ctx context.Context, id string, value float64) error
	AddCounter(ctx context.Context, id string, delta int64) error

	GetByID(ctx context.Context, id string) (*model.Metrics, error)

	// BatchUpdate allows receiving multiple metrics values and accumulate them simultaneously
	// if any metric is invalid, all data is discarded, and an error returned on the validation step
	BatchUpdate(ctx context.Context, metrics []*model.Metrics) error

	// GetAllSorted should return the full list of metrics sorted by ID lexicographically
	GetAllSorted(ctx context.Context) ([]*model.Metrics, error)

	// SetAll allows to bulk set data from an external source (no validations performed)
	SetAll(ctx context.Context, metrics []*model.Metrics) error

	ResetAll(ctx context.Context) error

	Ping(ctx context.Context) error

	Close() error
}

var (
	ErrMetricNotFound  = errors.New("metric not found in storage")
	ErrIncorrectAccess = errors.New("wrong access method used for metric")
)
