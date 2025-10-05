package agent

import (
	"github.com/andrewsvn/metrics-overseer/internal/agent/accumulation"
	"github.com/andrewsvn/metrics-overseer/internal/agent/reporting"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/mocks"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/mock"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentPolling(t *testing.T) {
	l, err := logging.NewZapLogger("info")
	require.NoError(t, err)
	stor := accumulation.NewAccumulatorStorage()
	p := NewPoller(agentcfg.Default(), stor, l)

	p.execMemstatsPoll()
	assert.Greater(t, p.stor.Length(), 2)
	assert.NotNil(t, p.stor.Get("RandomValue"))
	assert.NotNil(t, p.stor.Get("PollCount"))

	p.execGopsPoll()
	assert.NotNil(t, p.stor.Get("TotalMemory"))
	assert.NotNil(t, p.stor.Get("FreeMemory"))
}

func TestAgentReporting(t *testing.T) {
	var mcnt int
	var cnt1val, cnt2val int64
	var gauge1val, gauge2val float64
	var err error

	l, err := logging.NewZapLogger("info")
	require.NoError(t, err)
	stor := accumulation.NewAccumulatorStorage()
	r, err := NewReporter(agentcfg.Default(), stor, l)
	require.NoError(t, err)

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
	r.executor = reporting.NewBatchExecutor(msender, l.Sugar())

	err = stor.GetOrNew("cnt1").AccumulateCounter(1)
	assert.NoError(t, err)
	_ = stor.GetOrNew("cnt2").AccumulateCounter(0)
	err = stor.GetOrNew("gauge1").AccumulateGauge(1.25)
	assert.NoError(t, err)
	_ = stor.GetOrNew("gauge2").AccumulateGauge(-3.14)

	_ = stor.GetOrNew("cnt1").AccumulateCounter(2)
	_ = stor.GetOrNew("cnt2").AccumulateCounter(1)
	_ = stor.GetOrNew("gauge1").AccumulateGauge(1.75)
	_ = stor.GetOrNew("gauge2").AccumulateGauge(3.15)

	r.execReport()

	assert.Equal(t, 4, mcnt)
	assert.Equal(t, int64(3), cnt1val)
	assert.Equal(t, int64(1), cnt2val)
	assert.InDelta(t, 1.5, gauge1val, 0.0001)
	assert.InDelta(t, 0.005, gauge2val, 0.0001)

	_ = stor.GetOrNew("cnt1").AccumulateCounter(1)
	_ = stor.GetOrNew("gauge1").AccumulateGauge(1.6)

	r.execReport()

	assert.Equal(t, 2, mcnt)
	assert.Equal(t, int64(1), cnt1val)
	assert.Equal(t, 1.6, gauge1val)
}
