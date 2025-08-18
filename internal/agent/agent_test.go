package agent

import (
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/mocks"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentAccumulators(t *testing.T) {
	l, err := logging.NewZapLogger("info")
	require.NoError(t, err)
	a, err := NewAgent(agentcfg.Default(), l)
	require.NoError(t, err)

	assert.Equal(t, 0, a.accums.Length())

	a.storeCounterMetric("cnt1", 1)
	a.storeCounterMetric("cnt2", 0)
	a.storeGaugeMetric("gauge1", 1.25)
	a.storeGaugeMetric("gauge2", -3.14)

	assert.Equal(t, 4, a.accums.Length())
	assert.NotNil(t, a.accums.Get("cnt1"))
	assert.NotNil(t, a.accums.Get("cnt2"))
	assert.NotNil(t, a.accums.Get("gauge1"))
	assert.NotNil(t, a.accums.Get("gauge2"))
}

func TestAgentPolling(t *testing.T) {
	l, err := logging.NewZapLogger("info")
	require.NoError(t, err)
	a, err := NewAgent(agentcfg.Default(), l)
	require.NoError(t, err)

	a.execMemstatsPoll()
	assert.Greater(t, a.accums.Length(), 2)
	assert.NotNil(t, a.accums.Get("RandomValue"))
	assert.NotNil(t, a.accums.Get("PollCount"))
}

func TestAgentReporting(t *testing.T) {
	var mcnt int
	var cnt1val, cnt2val int64
	var gauge1val, gauge2val float64

	msender := new(mocks.MockMetricSender)
	msender.EXPECT().SendMetricArray(mock.Anything).
		RunAndReturn(func(metrics []*model.Metrics) error {
			mcnt = len(metrics)
			for _, m := range metrics {
				switch m.ID {
				case "cnt1":
					cnt1val = *m.Delta
				case "cnt2":
					cnt2val = *m.Delta
				case "gauge1":
					gauge1val = *m.Value
				case "gauge2":
					gauge2val = *m.Value
				}
			}
			return nil
		}).
		Times(2)

	l, err := logging.NewZapLogger("info")
	require.NoError(t, err)
	a, err := NewAgent(agentcfg.Default(), l)
	require.NoError(t, err)
	a.mSender = msender

	a.storeCounterMetric("cnt1", 1)
	a.storeCounterMetric("cnt2", 0)
	a.storeGaugeMetric("gauge1", 1.25)
	a.storeGaugeMetric("gauge2", -3.14)

	a.storeCounterMetric("cnt1", 2)
	a.storeCounterMetric("cnt2", 1)
	a.storeGaugeMetric("gauge1", 1.75)
	a.storeGaugeMetric("gauge2", 3.15)

	a.execReport()

	assert.Equal(t, 4, mcnt)
	assert.Equal(t, int64(3), cnt1val)
	assert.Equal(t, int64(1), cnt2val)
	assert.InDelta(t, 1.5, gauge1val, 0.0001)
	assert.InDelta(t, 0.005, gauge2val, 0.0001)

	a.storeCounterMetric("cnt1", 1)
	a.storeGaugeMetric("gauge1", 1.6)

	a.execReport()

	assert.Equal(t, 2, mcnt)
	assert.Equal(t, int64(1), cnt1val)
	assert.Equal(t, 1.6, gauge1val)
}
