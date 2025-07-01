package metrics

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestCounterAccumulator(t *testing.T) {
	cntAcc := NewMetricAccumulator("cnt", model.Counter)
	assert.Nil(t, cntAcc.Delta)
	assert.Empty(t, cntAcc.Values)

	err := cntAcc.AccumulateCounter(1)
	assert.NoError(t, err)

	err = cntAcc.AccumulateGauge(100)
	assert.ErrorAs(t, err, &model.ErrMethodNotSupported)

	cntAcc.AccumulateCounter(2)
	cntAcc.AccumulateCounter(3)
	assert.Equal(t, int64(6), *cntAcc.Delta)

	err = cntAcc.ExtractAndSend(func(id, mtype, value string) error {
		assert.Equal(t, "cnt", id)
		assert.Equal(t, model.Counter, mtype)
		assert.Equal(t, "6", value)
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(0), *cntAcc.Delta)

	cntAcc.AccumulateCounter(5)
	err = cntAcc.ExtractAndSend(func(id, mtype, value string) error {
		assert.Equal(t, "5", value)
		return fmt.Errorf("sender error")
	})
	assert.Error(t, err)
	assert.Equal(t, int64(5), *cntAcc.Delta)

	err = cntAcc.AccumulateCounter(-3)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), *cntAcc.Delta)
}

func TestGaugeAccumulator(t *testing.T) {
	gaAcc := NewMetricAccumulator("mem", model.Gauge)
	assert.Nil(t, gaAcc.Delta)
	assert.Empty(t, gaAcc.Values)

	err := gaAcc.AccumulateCounter(1)
	assert.ErrorAs(t, err, &model.ErrMethodNotSupported)

	err = gaAcc.AccumulateGauge(1.5)
	assert.NoError(t, err)

	gaAcc.AccumulateGauge(3.0)
	gaAcc.AccumulateGauge(4.5)
	assert.Equal(t, 3, len(gaAcc.Values))

	err = gaAcc.ExtractAndSend(func(id, mtype, value string) error {
		assert.Equal(t, "mem", id)
		assert.Equal(t, model.Gauge, mtype)
		fval, err := strconv.ParseFloat(value, 64)
		assert.NoError(t, err)
		assert.Equal(t, 3.0, fval)
		return nil
	})
	assert.NoError(t, err)
	assert.Empty(t, gaAcc.Values)

	gaAcc.AccumulateGauge(2.0)
	gaAcc.AccumulateGauge(-2.5)
	err = gaAcc.ExtractAndSend(func(id, mtype, value string) error {
		fval, err := strconv.ParseFloat(value, 64)
		assert.NoError(t, err)
		assert.Equal(t, -0.25, fval)
		return fmt.Errorf("sender error")
	})
	assert.Error(t, err)
	assert.Equal(t, 2, len(gaAcc.Values))
}
