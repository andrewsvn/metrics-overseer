package metrics

import (
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

	metric, err := cntAcc.StageChanges()
	require.NoError(t, err)
	assert.Equal(t, "cnt", metric.ID)
	assert.Equal(t, model.Counter, metric.MType)
	assert.Equal(t, int64(6), *metric.Delta)
	assert.Nil(t, metric.Value)
	assert.Nil(t, cntAcc.Delta)

	err = cntAcc.CommitStaged()
	require.NoError(t, err)
	assert.Nil(t, cntAcc.Delta)

	_ = cntAcc.AccumulateCounter(5)
	metric, err = cntAcc.StageChanges()
	require.NoError(t, err)
	assert.Equal(t, int64(5), *metric.Delta)
	assert.Nil(t, cntAcc.Delta)

	err = cntAcc.RollbackStaged()
	require.NoError(t, err)
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

	metric, err := gaAcc.StageChanges()
	require.NoError(t, err)
	assert.Equal(t, "mem", metric.ID)
	assert.Equal(t, model.Gauge, metric.MType)
	assert.InDelta(t, 3.0, *metric.Value, 0.0001)
	assert.Nil(t, metric.Delta)
	assert.Empty(t, gaAcc.Values)

	err = gaAcc.CommitStaged()
	require.NoError(t, err)
	assert.Empty(t, gaAcc.Values)

	_ = gaAcc.AccumulateGauge(2.0)
	_ = gaAcc.AccumulateGauge(-2.5)

	metric, err = gaAcc.StageChanges()
	require.NoError(t, err)
	assert.InDelta(t, -0.25, *metric.Value, 0.0001)
	assert.Empty(t, gaAcc.Values)

	err = gaAcc.RollbackStaged()
	require.NoError(t, err)
	assert.Equal(t, 2, len(gaAcc.Values))
}
