package repository

import (
	"errors"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

// all methods can return error in case when some internal storage problem occurs
// or method is unsupported for chosen metric
// error is not returned if data is not found in get methods

type Storage interface {
	GetGauge(id string) (*float64, error)
	SetGauge(id string, value float64) error

	GetCounter(id string) (*int64, error)
	AddCounter(id string, delta int64) error

	GetByID(id string) (*model.Metrics, error)

	// BatchUpdate allows receiving multiple metrics values and store them simultaneously
	// if any metric is invalid, all data is discarded, and an error returned on the validation step
	BatchUpdate([]*model.Metrics) error

	// GetAllSorted should return the full list of metrics sorted by ID lexicographically
	GetAllSorted() ([]*model.Metrics, error)

	// SetAll allows to bulk set data from an external source (no validations performed)
	SetAll(metrics []*model.Metrics) error

	ResetAll() error

	Close() error
}

var (
	ErrMetricNotFound = errors.New("metric not found in storage")
)
