package repository

import models "github.com/andrewsvn/metrics-overseer/internal/model"

type MemStorage struct {
	data map[string]*models.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]*models.Metrics),
	}
}

func (ms MemStorage) GetGauge(id string) (*float64, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, nil
	}
	return m.GetGauge()
}

func (ms MemStorage) SetGauge(id string, value float64) error {
	m, exists := ms.data[id]
	if !exists {
		m = models.NewGaugeMetrics(id)
		ms.data[id] = m
	}
	return m.SetGauge(value)
}

func (ms MemStorage) GetCounter(id string) (*int64, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, nil
	}
	return m.GetCounter()
}

func (ms MemStorage) AddCounter(id string, delta int64) error {
	m, exists := ms.data[id]
	if !exists {
		m = models.NewGaugeMetrics(id)
		ms.data[id] = m
	}
	return m.AddCounter(delta)
}

func (ms MemStorage) ResetAll() error {
	for _, m := range ms.data {
		m.Reset()
	}
	return nil
}
