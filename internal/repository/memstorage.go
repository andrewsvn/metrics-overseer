package repository

import (
	"context"
	"fmt"
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

func (ms *MemStorage) GetGauge(_ context.Context, id string) (*float64, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, ErrMetricNotFound
	}
	return m.GetGauge()
}

func (ms *MemStorage) SetGauge(_ context.Context, id string, value float64) error {
	m, exists := ms.data[id]
	if !exists {
		m = model.NewGaugeMetrics(id)
		ms.data[id] = m
	}
	return m.SetGauge(value)
}

func (ms *MemStorage) GetCounter(_ context.Context, id string) (*int64, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, ErrMetricNotFound
	}
	return m.GetCounter()
}

func (ms *MemStorage) AddCounter(_ context.Context, id string, delta int64) error {
	m, exists := ms.data[id]
	if !exists {
		m = model.NewCounterMetrics(id)
		ms.data[id] = m
	}
	return m.AddCounter(delta)
}

func (ms *MemStorage) GetByID(_ context.Context, id string) (*model.Metrics, error) {
	m, exists := ms.data[id]
	if !exists {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

func (ms *MemStorage) BatchUpdate(ctx context.Context, metrics []*model.Metrics) error {
	for _, m := range metrics {
		old := ms.data[m.ID]
		if old != nil && old.MType != m.MType {
			return fmt.Errorf("%w: for metric id=%s old type=%s, new type=%s",
				model.ErrIncorrectAccess, m.ID, old.MType, m.MType)
		}
	}
	for _, m := range metrics {
		switch m.MType {
		case model.Counter:
			_ = ms.AddCounter(ctx, m.ID, *m.Delta)
		case model.Gauge:
			_ = ms.SetGauge(ctx, m.ID, *m.Value)
		}
	}
	return nil
}

func (ms *MemStorage) GetAllSorted(_ context.Context) ([]*model.Metrics, error) {
	mlist := make([]*model.Metrics, 0, len(ms.data))
	for _, v := range ms.data {
		mlist = append(mlist, v)
	}
	slices.SortFunc(mlist, func(a *model.Metrics, b *model.Metrics) int {
		return strings.Compare(a.ID, b.ID)
	})
	return mlist, nil
}

func (ms *MemStorage) SetAll(_ context.Context, metrics []*model.Metrics) error {
	for _, m := range metrics {
		ms.data[m.ID] = model.NewMetrics(m.ID, m.MType, m.Delta, m.Value)
	}
	return nil
}

func (ms *MemStorage) ResetAll(_ context.Context) error {
	for _, m := range ms.data {
		m.Reset()
	}
	return nil
}

func (ms *MemStorage) Ping(_ context.Context) error {
	// memory storage is always available
	return nil
}

func (ms *MemStorage) Close() error {
	return nil
}
