package repository

import (
	"slices"
	"strings"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type MemStorage struct {
	data map[string]*model.Metrics
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data: make(map[string]*model.Metrics),
	}
}

func (ms *MemStorage) GetGauge(id string) (*float64, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, ErrMetricNotFound
	}
	return m.GetGauge()
}

func (ms *MemStorage) SetGauge(id string, value float64) error {
	m, exists := ms.data[id]
	if !exists {
		m = model.NewGaugeMetrics(id)
		ms.data[id] = m
	}
	return m.SetGauge(value)
}

func (ms *MemStorage) GetCounter(id string) (*int64, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, ErrMetricNotFound
	}
	return m.GetCounter()
}

func (ms *MemStorage) AddCounter(id string, delta int64) error {
	m, exists := ms.data[id]
	if !exists {
		m = model.NewCounterMetrics(id)
		ms.data[id] = m
	}
	return m.AddCounter(delta)
}

func (ms *MemStorage) GetByID(id string) (*model.Metrics, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

func (ms *MemStorage) GetAllSorted() ([]*model.Metrics, error) {
	mlist := make([]*model.Metrics, 0, len(ms.data))
	for _, v := range ms.data {
		mlist = append(mlist, v)
	}
	slices.SortFunc(mlist, func(a *model.Metrics, b *model.Metrics) int {
		return strings.Compare(a.ID, b.ID)
	})
	return mlist, nil
}

func (ms *MemStorage) ResetAll() error {
	for _, m := range ms.data {
		m.Reset()
	}
	return nil
}
