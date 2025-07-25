package repository

import (
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemStorageCounters(t *testing.T) {
	ms := NewMemStorage()

	_ = ms.AddCounter("cnt1", 1)
	_ = ms.AddCounter("cnt2", 2)
	_ = ms.AddCounter("cnt1", 3)
	_ = ms.AddCounter("cnt2", 4)

	cnt1, err := ms.GetCounter("cnt1")
	assert.NoError(t, err)
	assert.Equal(t, int64(4), *cnt1)
	cnt2, err := ms.GetCounter("cnt2")
	assert.NoError(t, err)
	assert.Equal(t, int64(6), *cnt2)
	_, err = ms.GetCounter("cnt3")
	assert.ErrorAs(t, err, &ErrMetricNotFound)

	_ = ms.AddCounter("cnt1", -2)
	_ = ms.AddCounter("cnt2", -8)
	cnt1, _ = ms.GetCounter("cnt1")
	assert.Equal(t, int64(2), *cnt1)
	cnt2, _ = ms.GetCounter("cnt2")
	assert.Equal(t, int64(-2), *cnt2)

	_, err = ms.GetGauge("cnt1")
	assert.ErrorAs(t, err, &model.ErrIncorrectAccess)

	err = ms.ResetAll()
	require.NoError(t, err)
	cnt1, _ = ms.GetCounter("cnt1")
	assert.Nil(t, cnt1)
}

func TestMemStorageGauges(t *testing.T) {
	ms := NewMemStorage()

	_ = ms.SetGauge("gauge1", 1.11)
	_ = ms.SetGauge("gauge2", 3.33)

	g1, err := ms.GetGauge("gauge1")
	assert.NoError(t, err)
	assert.Equal(t, 1.11, *g1)
	g2, err := ms.GetGauge("gauge2")
	assert.NoError(t, err)
	assert.Equal(t, 3.33, *g2)
	_, err = ms.GetGauge("gauge3")
	assert.ErrorAs(t, err, &ErrMetricNotFound)

	_ = ms.SetGauge("gauge1", 0.0)
	_ = ms.SetGauge("gauge2", -2.22)
	g1, _ = ms.GetGauge("gauge1")
	assert.Equal(t, 0.0, *g1)
	g2, _ = ms.GetGauge("gauge2")
	assert.Equal(t, -2.22, *g2)

	_, err = ms.GetCounter("gauge1")
	assert.ErrorAs(t, err, &model.ErrIncorrectAccess)
}

func TestMemStorageGetAll(t *testing.T) {
	ms := NewMemStorage()

	_ = ms.SetGauge("1gauge1", 1.11)
	_ = ms.SetGauge("2gauge2", 3.33)
	_ = ms.AddCounter("1cnt1", 1)
	_ = ms.AddCounter("2cnt2", 2)

	metrics, err := ms.GetAllSorted()
	require.NoError(t, err)
	assert.Equal(t, 4, len(metrics))
	assert.Equal(t, "1cnt1", metrics[0].ID)
	assert.Equal(t, "1gauge1", metrics[1].ID)

	_ = ms.SetGauge("0gauge0", 2.22)
	_ = ms.SetGauge("2gauge2", -3.33)
	_ = ms.AddCounter("0cnt0", 3)
	_ = ms.AddCounter("2cnt2", -2)

	metrics, err = ms.GetAllSorted()
	require.NoError(t, err)
	assert.Equal(t, 6, len(metrics))
	assert.Equal(t, "0cnt0", metrics[0].ID)
	assert.Equal(t, "0gauge0", metrics[1].ID)
	assert.Equal(t, "2cnt2", metrics[4].ID)
	assert.Equal(t, "2gauge2", metrics[5].ID)
}

func TestMemStorageGetByID(t *testing.T) {
	ms := NewMemStorage()

	_ = ms.SetGauge("gauge1", 1.11)
	_ = ms.AddCounter("cnt1", 1)

	metric, err := ms.GetByID("cnt1")
	require.NoError(t, err)
	assert.Equal(t, "cnt1", metric.ID)
	assert.Equal(t, model.Counter, metric.MType)
	assert.Equal(t, int64(1), *metric.Delta)
	assert.Nil(t, metric.Value)

	metric, err = ms.GetByID("gauge1")
	require.NoError(t, err)
	assert.Equal(t, "gauge1", metric.ID)
	assert.Equal(t, model.Gauge, metric.MType)
	assert.Equal(t, 1.11, *metric.Value)
	assert.Nil(t, metric.Delta)

	_, err = ms.GetByID("cnt2")
	require.ErrorAs(t, err, &ErrMetricNotFound)
}
