package repository

import "errors"

// all methods have an option to return error in case
// when some internal storage problem occurs
// or method is unsupported for chosen metric
// error is not returned if data is not found in get methods
type Storage interface {
	GetGauge(id string) (*float64, error)
	SetGauge(id string, value float64) error

	GetCounter(id string) (*int64, error)
	AddCounter(id string, delta int64) error

	ResetAll() error
}

var (
	ErrMetricNotFound = errors.New("metric not found in storage")
)
