package pool

import "sync"

type Resettable interface {
	Reset()
}

type ResettableObjectPool[T Resettable] struct {
	free  []T
	mutex *sync.Mutex

	New func() T
}

func NewResettableObjectPool[T Resettable](newF func() T) *ResettableObjectPool[T] {
	return &ResettableObjectPool[T]{
		free:  make([]T, 0),
		mutex: new(sync.Mutex),
		New:   newF,
	}
}

func (re *ResettableObjectPool[T]) Get() T {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	if len(re.free) > 0 {
		ptr := re.free[0]
		re.free = re.free[1:]
		return ptr
	}

	return re.New()
}

func (re *ResettableObjectPool[T]) Put(ptr T) {
	re.mutex.Lock()
	defer re.mutex.Unlock()

	ptr.Reset()
	re.free = append(re.free, ptr)
}
