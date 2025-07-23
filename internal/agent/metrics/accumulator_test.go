package metrics

import (
	"fmt"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCounterAccumulator(t *testing.T) {
	cntAcc := NewMetricAccumulator("cnt", model.Counter)
	assert.Nil(t, cntAcc.Delta)
	assert.Empty(t, cntAcc.Values)

	err := cntAcc.AccumulateCounter(1)
	require.NoError(t, err)

	err = cntAcc.AccumulateGauge(100)
	assert.ErrorAs(t, err, &model.ErrIncorrectAccess)

	_ = cntAcc.AccumulateCounter(2)
	_ = cntAcc.AccumulateCounter(3)
	assert.Equal(t, int64(6), *cntAcc.Delta)

	err = cntAcc.ExtractAndSend(func(metric *model.Metrics) error {
		assert.Equal(t, "cnt", metric.ID)
		assert.Equal(t, model.Counter, metric.MType)
		assert.Equal(t, int64(6), *metric.Delta)
		assert.Nil(t, metric.Value)
		return nil
	})
	require.NoError(t, err)
	assert.Nil(t, cntAcc.Delta)

	_ = cntAcc.AccumulateCounter(5)
	err = cntAcc.ExtractAndSend(func(metric *model.Metrics) error {
		assert.Equal(t, int64(5), *metric.Delta)
		return fmt.Errorf("sender error")
	})
	assert.Error(t, err)
	assert.Equal(t, int64(5), *cntAcc.Delta)

	err = cntAcc.AccumulateCounter(-3)
	require.NoError(t, err)
	assert.Equal(t, int64(2), *cntAcc.Delta)
}

func TestGaugeAccumulator(t *testing.T) {
	gaAcc := NewMetricAccumulator("mem", model.Gauge)
	assert.Nil(t, gaAcc.Delta)
	assert.Empty(t, gaAcc.Values)

	err := gaAcc.AccumulateCounter(1)
	assert.ErrorAs(t, err, &model.ErrIncorrectAccess)

	err = gaAcc.AccumulateGauge(1.5)
	require.NoError(t, err)

	_ = gaAcc.AccumulateGauge(3.0)
	_ = gaAcc.AccumulateGauge(4.5)
	assert.Equal(t, 3, len(gaAcc.Values))

	err = gaAcc.ExtractAndSend(func(metric *model.Metrics) error {
		assert.Equal(t, "mem", metric.ID)
		assert.Equal(t, model.Gauge, metric.MType)
		assert.Equal(t, 3.0, *metric.Value)
		assert.Nil(t, metric.Delta)
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, gaAcc.Values)

	_ = gaAcc.AccumulateGauge(2.0)
	_ = gaAcc.AccumulateGauge(-2.5)
	err = gaAcc.ExtractAndSend(func(metric *model.Metrics) error {
		assert.Equal(t, -0.25, *metric.Value)
		return fmt.Errorf("sender error")
	})
	assert.Error(t, err)
	assert.Equal(t, 2, len(gaAcc.Values))
}
