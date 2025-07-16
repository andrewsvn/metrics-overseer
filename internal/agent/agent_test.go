package agent

import (
	"strconv"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSender struct {
	t          *testing.T
	callTotal  int
	cntCalls   map[string]int64
	gaugeCalls map[string]float64
}

func newTestSender(t *testing.T) *testSender {
	return &testSender{
		t:          t,
		cntCalls:   make(map[string]int64),
		gaugeCalls: make(map[string]float64),
	}
}

func (ts *testSender) ValueSendFunc() sender.MetricValueSendFunc {
	return func(id, mtype, value string) error {
		ts.callTotal += 1
		switch mtype {
		case model.Counter:
			ival, err := strconv.ParseInt(value, 10, 64)
			require.NoError(ts.t, err)
			ts.cntCalls[id] = int64(ival)
		case model.Gauge:
			fval, err := strconv.ParseFloat(value, 64)
			require.NoError(ts.t, err)
			ts.gaugeCalls[id] = fval
		default:
			assert.Fail(ts.t, "Incorrect metric type passed: "+mtype)
		}
		return nil
	}
}

func (ts *testSender) StructSendFunc() sender.MetricStructSendFunc {
	return func(metric *model.Metrics) error {
		ts.callTotal += 1
		switch metric.MType {
		case model.Counter:
			assert.NotNil(ts.t, metric.Delta)
			assert.Nil(ts.t, metric.Value)
			ts.cntCalls[metric.ID] = *metric.Delta
		case model.Gauge:
			assert.NotNil(ts.t, metric.Value)
			assert.Nil(ts.t, metric.Delta)
			ts.gaugeCalls[metric.ID] = *metric.Value
		default:
			assert.Fail(ts.t, "Incorrect metric type passed: "+metric.MType)
		}
		return nil
	}
}

func TestAgentAccumulators(t *testing.T) {
	a, err := NewAgent(agentcfg.Default())
	require.NoError(t, err)

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
	a, err := NewAgent(agentcfg.Default())
	require.NoError(t, err)

	a.execPoll()
	assert.Greater(t, len(a.accums), 2)
	_, exists := a.accums["RandomValue"]
	assert.True(t, exists)
	_, exists = a.accums["PollCount"]
	assert.True(t, exists)
}

func TestAgentReporting(t *testing.T) {
	a, err := NewAgent(agentcfg.Default())
	require.NoError(t, err)

	a.storeCounterMetric("cnt1", 1)
	a.storeCounterMetric("cnt2", 0)
	a.storeGaugeMetric("gauge1", 1.25)
	a.storeGaugeMetric("gauge2", -3.14)

	a.storeCounterMetric("cnt1", 2)
	a.storeCounterMetric("cnt2", 1)
	a.storeGaugeMetric("gauge1", 1.75)
	a.storeGaugeMetric("gauge2", 3.14)

	ts := newTestSender(t)
	a.sndr = ts
	a.execReport()

	assert.Equal(t, 4, ts.callTotal)
	assert.Equal(t, int64(3), ts.cntCalls["cnt1"])
	assert.Equal(t, int64(1), ts.cntCalls["cnt2"])
	assert.Equal(t, 1.5, ts.gaugeCalls["gauge1"])
	assert.Equal(t, 0.0, ts.gaugeCalls["gauge2"])

	ts.callTotal = 0
	for k := range ts.cntCalls {
		delete(ts.cntCalls, k)
	}
	for k := range ts.gaugeCalls {
		delete(ts.gaugeCalls, k)
	}

	a.storeCounterMetric("cnt1", 1)
	a.storeGaugeMetric("gauge1", 1.6)
	a.execReport()
	assert.Equal(t, 2, ts.callTotal)
}
