package model

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounterMetrics(t *testing.T) {
	const id = "cnt"

	m := NewCounterMetrics(id)
	assert.Equal(t, id, m.ID)
	assert.Equal(t, Counter, m.MType)
	assert.Nil(t, m.Delta)
	assert.Nil(t, m.Value)
	initialHash := m.Hash

	m.AddCounter(2)
	require.NotNil(t, m.Delta)
	assert.Equal(t, int64(2), *m.Delta)
	assert.NotEqual(t, initialHash, m.Hash)

	m.AddCounter(-2)
	require.NotNil(t, m.Delta)
	assert.Equal(t, int64(0), *m.Delta)
	assert.NotEqual(t, initialHash, m.Hash)
	valHash := m.Hash

	m = NewCounterMetricsWithDelta(id, 0)
	assert.Equal(t, id, m.ID)
	assert.Equal(t, Counter, m.MType)
	assert.NotNil(t, m.Delta)
	assert.Equal(t, valHash, m.Hash)
}

func TestGaugeMetrics(t *testing.T) {
	const id = "gauge"

	m := NewGaugeMetrics(id)
	assert.Equal(t, id, m.ID)
	assert.Equal(t, Gauge, m.MType)
	assert.Nil(t, m.Delta)
	assert.Nil(t, m.Value)
	initialHash := m.Hash

	m.SetGauge(3.14)
	require.NotNil(t, m.Value)
	assert.Equal(t, 3.14, *m.Value)
	assert.NotEqual(t, initialHash, m.Hash)

	m.SetGauge(0.0)
	require.NotNil(t, m.Value)
	assert.Equal(t, 0.0, *m.Value)
	assert.NotEqual(t, initialHash, m.Hash)
	valHash := m.Hash

	m = NewGaugeMetricsWithValue(id, 0.0)
	require.NotNil(t, m.Value)
	assert.Equal(t, 0.0, *m.Value)
	assert.Equal(t, valHash, m.Hash)
}

func BenchmarkMetricsUpdateHash(b *testing.B) {
	var m *Metrics
	rnd := rand.New(rand.NewSource(0))
	for i := 0; i < b.N; i++ {
		if i%2 == 0 {
			m = NewCounterMetricsWithDelta("cnt", rnd.Int63n(100))
		} else {
			m = NewGaugeMetricsWithValue("gauge", rnd.Float64())
		}
		_ = m
	}
}
