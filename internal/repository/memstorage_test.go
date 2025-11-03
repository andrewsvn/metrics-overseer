package repository

import (
	"context"
	"fmt"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorageCounters(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	_ = ms.AddCounter(ctx, "cnt1", 1)
	_ = ms.AddCounter(ctx, "cnt2", 2)
	_ = ms.AddCounter(ctx, "cnt1", 3)
	_ = ms.AddCounter(ctx, "cnt2", 4)

	cnt1, err := ms.GetByID(ctx, "cnt1")
	assert.NoError(t, err)
	assert.Equal(t, "cnt1", cnt1.ID)
	assert.Equal(t, model.Counter, cnt1.MType)
	assert.Equal(t, int64(4), *cnt1.Delta)
	assert.Nil(t, cnt1.Value)

	cnt2, err := ms.GetByID(ctx, "cnt2")
	assert.NoError(t, err)
	assert.Equal(t, "cnt2", cnt2.ID)
	assert.Equal(t, model.Counter, cnt2.MType)
	assert.Equal(t, int64(6), *cnt2.Delta)
	assert.Nil(t, cnt2.Value)

	_, err = ms.GetByID(ctx, "cnt3")
	assert.ErrorAs(t, err, &ErrMetricNotFound)

	_ = ms.AddCounter(ctx, "cnt1", -2)
	_ = ms.AddCounter(ctx, "cnt2", -8)

	cnt1, _ = ms.GetByID(ctx, "cnt1")
	assert.Equal(t, int64(2), *cnt1.Delta)
	cnt2, _ = ms.GetByID(ctx, "cnt2")
	assert.Equal(t, int64(-2), *cnt2.Delta)

	err = ms.ResetAll(ctx)
	require.NoError(t, err)
	cnt1, err = ms.GetByID(ctx, "cnt1")
	assert.Error(t, err)
}

func TestMemStorageGauges(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	_ = ms.SetGauge(ctx, "gauge1", 1.11)
	_ = ms.SetGauge(ctx, "gauge2", 3.33)

	g1, err := ms.GetByID(ctx, "gauge1")
	assert.NoError(t, err)
	assert.Equal(t, "gauge1", g1.ID)
	assert.Equal(t, model.Gauge, g1.MType)
	assert.Equal(t, 1.11, *g1.Value)
	assert.Nil(t, g1.Delta)

	g2, err := ms.GetByID(ctx, "gauge2")
	assert.NoError(t, err)
	assert.Equal(t, "gauge2", g2.ID)
	assert.Equal(t, model.Gauge, g2.MType)
	assert.Equal(t, 3.33, *g2.Value)

	_, err = ms.GetByID(ctx, "gauge3")
	assert.ErrorAs(t, err, &ErrMetricNotFound)

	_ = ms.SetGauge(ctx, "gauge1", 0.0)
	_ = ms.SetGauge(ctx, "gauge2", -2.22)
	g1, _ = ms.GetByID(ctx, "gauge1")
	assert.Equal(t, 0.0, *g1.Value)
	g2, _ = ms.GetByID(ctx, "gauge2")
	assert.Equal(t, -2.22, *g2.Value)
}

func TestMemStorageGetAll(t *testing.T) {
	ms := NewMemStorage()
	ctx := context.Background()

	_ = ms.SetGauge(ctx, "1gauge1", 1.11)
	_ = ms.SetGauge(ctx, "2gauge2", 3.33)
	_ = ms.AddCounter(ctx, "1cnt1", 1)
	_ = ms.AddCounter(ctx, "2cnt2", 2)

	metrics, err := ms.GetAllSorted(ctx)
	require.NoError(t, err)
	assert.Equal(t, 4, len(metrics))
	assert.Equal(t, "1cnt1", metrics[0].ID)
	assert.Equal(t, "1gauge1", metrics[1].ID)

	_ = ms.SetGauge(ctx, "0gauge0", 2.22)
	_ = ms.SetGauge(ctx, "2gauge2", -3.33)
	_ = ms.AddCounter(ctx, "0cnt0", 3)
	_ = ms.AddCounter(ctx, "2cnt2", -2)

	metrics, err = ms.GetAllSorted(ctx)
	require.NoError(t, err)
	assert.Equal(t, 6, len(metrics))
	assert.Equal(t, "0cnt0", metrics[0].ID)
	assert.Equal(t, "0gauge0", metrics[1].ID)
	assert.Equal(t, "2cnt2", metrics[4].ID)
	assert.Equal(t, "2gauge2", metrics[5].ID)
}

func BenchmarkMemStorageAddCounter(b *testing.B) {
	b.StopTimer()
	ms := setupMemStorageForBenchmark()
	ctx := context.Background()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = ms.AddCounter(ctx, "cnt255", 5)
	}
}

func BenchmarkMemStorageSetGauge(b *testing.B) {
	b.StopTimer()
	ms := setupMemStorageForBenchmark()
	ctx := context.Background()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = ms.SetGauge(ctx, "gauge511", 3.14)
	}
}

func BenchmarkMemStorageGetAll(b *testing.B) {
	b.StopTimer()
	ms := setupMemStorageForBenchmark()
	ctx := context.Background()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		batch, _ := ms.GetAllSorted(ctx)
		_ = batch
	}
}

func BenchmarkMemStorageBatchUpdate(b *testing.B) {
	b.StopTimer()
	ctx := context.Background()
	metrics := makeMetricsBatch()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		ms := NewMemStorage()
		_ = ms.BatchUpdate(ctx, metrics)
	}
}

func setupMemStorageForBenchmark() *MemStorage {
	ms := NewMemStorage()
	_ = ms.SetAll(context.Background(), makeMetricsBatch())
	return ms
}

func makeMetricsBatch() []*model.Metrics {
	metrics := make([]*model.Metrics, 0, 20000)
	for i := 0; i < 10000; i++ {
		m := model.NewCounterMetrics(fmt.Sprintf("cnt%d", i))
		m.AddCounter(int64(i))
		metrics = append(metrics, m)

		m = model.NewGaugeMetrics(fmt.Sprintf("gauge%d", i))
		m.SetGauge(1.5 * float64(i))
		metrics = append(metrics, m)
	}
	return metrics
}
