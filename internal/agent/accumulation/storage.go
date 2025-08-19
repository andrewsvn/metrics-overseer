package accumulation

import (
	"iter"
	"maps"
	"sync"
)

type Storage struct {
	accums map[string]*MetricAccumulator
	mutex  sync.Mutex
}

func NewAccumulatorStorage() *Storage {
	return &Storage{
		accums: make(map[string]*MetricAccumulator),
	}
}

func (storage *Storage) GetOrNew(id string) *MetricAccumulator {
	if storage.accums[id] == nil {
		storage.mutex.Lock()
		if storage.accums[id] == nil {
			storage.accums[id] = NewMetricAccumulator(id)
		}
		storage.mutex.Unlock()
	}
	return storage.accums[id]
}

func (storage *Storage) Get(id string) *MetricAccumulator {
	return storage.accums[id]
}

func (storage *Storage) GetAll() iter.Seq[*MetricAccumulator] {
	return maps.Values(storage.accums)
}

func (storage *Storage) Length() int {
	return len(storage.accums)
}
