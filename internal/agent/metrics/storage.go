package metrics

import (
	"iter"
	"maps"
	"sync"
)

type AccumulatorStorage struct {
	accums map[string]*MetricAccumulator
	mutex  sync.Mutex
}

func NewAccumulatorStorage() *AccumulatorStorage {
	return &AccumulatorStorage{
		accums: make(map[string]*MetricAccumulator),
	}
}

func (storage *AccumulatorStorage) GetOrNew(id string) *MetricAccumulator {
	if storage.accums[id] == nil {
		storage.mutex.Lock()
		if storage.accums[id] == nil {
			storage.accums[id] = NewMetricAccumulator(id)
		}
		storage.mutex.Unlock()
	}
	return storage.accums[id]
}

func (storage *AccumulatorStorage) Get(id string) *MetricAccumulator {
	return storage.accums[id]
}

func (storage *AccumulatorStorage) GetAll() iter.Seq[*MetricAccumulator] {
	return maps.Values(storage.accums)
}

func (storage *AccumulatorStorage) Length() int {
	return len(storage.accums)
}
