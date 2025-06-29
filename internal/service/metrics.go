package service

import "github.com/andrewsvn/metrics-overseer/internal/repository"

type MetricsService struct {
	storage repository.Storage
}

func NewMetricsService(st repository.Storage) *MetricsService {
	return &MetricsService{
		storage: st,
	}
}

func (ms *MetricsService) AccumulateCounter(id string, inc int64) error {
	return ms.storage.AddCounter(id, inc)
}

func (ms *MetricsService) SetGauge(id string, val float64) error {
	return ms.storage.SetGauge(id, val)
}
