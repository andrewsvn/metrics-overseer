package pool

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type ResettableMock struct {
	resets *int
	Val    int
}

func (ms *ResettableMock) Reset() {
	*ms.resets++
	ms.Val = 0
}

func TestResettableObjectPool(t *testing.T) {
	var news, resets int
	pool := NewResettableObjectPool(func() *ResettableMock {
		news++
		return &ResettableMock{
			resets: &resets,
		}
	})

	mocks := make([]*ResettableMock, 0)
	for i := 0; i < 5; i++ {
		m := pool.Get()
		m.Val = i
		mocks = append(mocks, m)
	}

	for i := 0; i < 3; i++ {
		pool.Put(mocks[i])
	}
	mocks = mocks[3:]

	for i := 0; i < 4; i++ {
		mocks = append(mocks, pool.Get())
	}

	assert.Equal(t, 6, news)
	assert.Equal(t, 3, mocks[0].Val)
	assert.Equal(t, 4, mocks[1].Val)
	assert.Equal(t, 0, mocks[2].Val)
	assert.Equal(t, 0, mocks[3].Val)
	assert.Equal(t, 0, mocks[4].Val)
	assert.Equal(t, 0, mocks[5].Val)

	for i := 0; i < 5; i++ {
		pool.Put(mocks[i])
	}

	assert.Equal(t, 8, resets)
}
