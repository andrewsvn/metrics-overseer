package accumulation

import (
	"iter"
	"maps"
	"sync"
)

type Storage struct {
	accums map[string]*MetricAccumulator
	mutex  *sync.RWMutex
}

func NewAccumulatorStorage() *Storage {
	return &Storage{
		accums: make(map[string]*MetricAccumulator),
		mutex:  &sync.RWMutex{},
	}
}

func (storage *Storage) GetOrNew(id string) *MetricAccumulator {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	if storage.accums[id] == nil {
		storage.accums[id] = NewMetricAccumulator(id)
	}
	return storage.accums[id]
}

func (storage *Storage) Get(id string) *MetricAccumulator {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()
	return storage.accums[id]
}

func (storage *Storage) GetAll() iter.Seq[*MetricAccumulator] {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()
	return maps.Values(storage.accums)
}

func (storage *Storage) Length() int {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()
	return len(storage.accums)
}
