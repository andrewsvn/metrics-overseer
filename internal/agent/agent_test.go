package agent

import (
	"strconv"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestAgentAccumulators(t *testing.T) {
	a := NewAgent()
	assert.Empty(t, a.accums)

	a.storeCounterMetric("cnt1", 1)
	a.storeCounterMetric("cnt2", 0)
	a.storeGaugeMetric("gauge1", 1.25)
	a.storeGaugeMetric("gauge2", -3.14)

	assert.Equal(t, 4, len(a.accums))
	assert.NotNil(t, a.accums["cnt1"])
	assert.NotNil(t, a.accums["cnt2"])
	assert.NotNil(t, a.accums["gauge1"])
	assert.NotNil(t, a.accums["gauge2"])
}

func TestAgentPolling(t *testing.T) {
	a := NewAgent()
	a.execPoll()
	assert.Greater(t, len(a.accums), 2)
	_, exists := a.accums["RandomValue"]
	assert.True(t, exists)
	_, exists = a.accums["PollCount"]
	assert.True(t, exists)
}

func TestAgentReporting(t *testing.T) {
	a := NewAgent()

	a.storeCounterMetric("cnt1", 1)
	a.storeCounterMetric("cnt2", 0)
	a.storeGaugeMetric("gauge1", 1.25)
	a.storeGaugeMetric("gauge2", -3.14)

	a.storeCounterMetric("cnt1", 2)
	a.storeCounterMetric("cnt2", 1)
	a.storeGaugeMetric("gauge1", 1.75)
	a.storeGaugeMetric("gauge2", 3.14)

	var callTotal int
	cntCalls := make(map[string]int64)
	gaugeCalls := make(map[string]float64)
	a.sendfunc = func(id, mtype, value string) error {
		callTotal += 1
		switch mtype {
		case model.Counter:
			ival, err := strconv.Atoi(value)
			assert.NoError(t, err)
			cntCalls[id] = int64(ival)
		case model.Gauge:
			fval, err := strconv.ParseFloat(value, 64)
			assert.NoError(t, err)
			gaugeCalls[id] = fval
		default:
			assert.Fail(t, "Incorrect metric type passed: "+mtype)
		}
		return nil
	}

	a.execReport()
	assert.Equal(t, 4, callTotal)
	assert.Equal(t, int64(3), cntCalls["cnt1"])
	assert.Equal(t, int64(1), cntCalls["cnt2"])
	assert.Equal(t, 1.5, gaugeCalls["gauge1"])
	assert.Equal(t, 0.0, gaugeCalls["gauge2"])

	callTotal = 0
	for k := range cntCalls {
		delete(cntCalls, k)
	}
	for k := range gaugeCalls {
		delete(gaugeCalls, k)
	}

	a.storeCounterMetric("cnt1", 1)
	a.storeGaugeMetric("gauge1", 1.6)
	a.execReport()
	assert.Equal(t, 2, callTotal)
}
