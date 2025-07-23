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

	// GetAllSorted should return the full list of metrics sorted by ID lexicographically
	GetAllSorted() ([]*model.Metrics, error)
	SetAll(metrics []*model.Metrics) error

	ResetAll() error
}

var (
	ErrMetricNotFound = errors.New("metric not found in storage")
)
