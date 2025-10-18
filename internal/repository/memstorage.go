package repository

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type MemStorage struct {
	data map[string]*model.Metrics

	mutex *sync.RWMutex
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		data:  make(map[string]*model.Metrics),
		mutex: &sync.RWMutex{},
	}
}

func (ms *MemStorage) SetGauge(_ context.Context, id string, value float64) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.setGaugeInMutex(id, value)
}

func (ms *MemStorage) setGaugeInMutex(id string, value float64) error {
	if ms.data[id] == nil {
		ms.data[id] = model.NewGaugeMetricsWithValue(id, value)
		return nil
	}
	if ms.data[id].MType != model.Gauge {
		return ErrIncorrectAccess
	}

	ms.data[id].SetGauge(value)
	return nil
}

func (ms *MemStorage) AddCounter(_ context.Context, id string, delta int64) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()
	return ms.addCounterInMutex(id, delta)
}

func (ms *MemStorage) addCounterInMutex(id string, delta int64) error {
	if ms.data[id] == nil {
		ms.data[id] = model.NewCounterMetricsWithDelta(id, delta)
		return nil
	}
	if ms.data[id].MType != model.Counter {
		return ErrIncorrectAccess
	}

	ms.data[id].AddCounter(delta)
	return nil
}

func (ms *MemStorage) GetByID(_ context.Context, id string) (*model.Metrics, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	m, exists := ms.data[id]
	if !exists {
		return nil, ErrMetricNotFound
	}
	return m, nil
}

func (ms *MemStorage) BatchUpdate(_ context.Context, metrics []*model.Metrics) error {
	ms.mutex.Lock()
	defer ms.mutex.Unlock()

	// perform all validations before update to prevent partial update
	for _, m := range metrics {
		old := ms.data[m.ID]
		if old != nil && old.MType != m.MType {
			return fmt.Errorf("%w: for metric id=%s old type=%s, new type=%s",
				ErrIncorrectAccess, m.ID, old.MType, m.MType)
		}
	}

	for _, m := range metrics {
		switch m.MType {
		case model.Counter:
			if m.Delta != nil {
				_ = ms.addCounterInMutex(m.ID, *m.Delta)
			}
		case model.Gauge:
			if m.Value != nil {
				_ = ms.setGaugeInMutex(m.ID, *m.Value)
			}
		}
	}
	return nil
}

func (ms *MemStorage) GetAllSorted(_ context.Context) ([]*model.Metrics, error) {
	ms.mutex.RLock()
	defer ms.mutex.RUnlock()

	mlist := make([]*model.Metrics, 0)
	for _, v := range ms.data {
		mlist = append(mlist, v)
	}
	slices.SortFunc(mlist, func(a *model.Metrics, b *model.Metrics) int {
		return strings.Compare(a.ID, b.ID)
	})
	return mlist, nil
}

func (ms *MemStorage) SetAll(_ context.Context, metrics []*model.Metrics) error {
	ms.mutex.Lock()
	for _, m := range metrics {
		ms.data[m.ID] = model.NewMetrics(m.ID, m.MType, m.Delta, m.Value)
	}
	ms.mutex.Unlock()
	return nil
}

func (ms *MemStorage) ResetAll(_ context.Context) error {
	ms.mutex.Lock()
	for _, m := range ms.data {
		m.Reset()
	}
	ms.mutex.Unlock()
	return nil
}

func (ms *MemStorage) Ping(_ context.Context) error {
	// memory storage is always available
	return nil
}

func (ms *MemStorage) Close() error {
	return nil
}
