package audit

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWriterOnMetricsUpdate(t *testing.T) {
	l, _ := logging.NewZapLogger("info")
	fsw := NewFileWriter("fswtest.txt", l)
	defer os.Remove(fsw.filename)

	metrics := []*model.Metrics{
		model.NewCounterMetrics("cnt1"),
		model.NewCounterMetrics("cnt2"),
		model.NewGaugeMetrics("gauge1"),
		model.NewGaugeMetrics("gauge2"),
	}

	ts := time.Now()
	events := []struct {
		timestamp time.Time
		ipAddr    string
		metrics   []*model.Metrics
	}{
		{
			timestamp: ts,
			ipAddr:    "10.10.10.10",
			metrics:   metrics[0:2],
		},
		{
			timestamp: ts.Add(5 * time.Second),
			ipAddr:    "11.11.11.11",
			metrics:   metrics[1:],
		},
		{
			timestamp: ts.Add(2 * time.Minute),
			ipAddr:    "12.12.12.12",
			metrics:   metrics[:1],
		},
	}

	for _, event := range events {
		fsw.OnMetricsUpdate(event.timestamp, event.ipAddr, event.metrics...)
	}

	f, err := os.Open(fsw.filename)
	require.NoError(t, err)
	defer f.Close()

	data, err := io.ReadAll(f)
	require.NoError(t, err)

	payloadStrs := strings.Split(string(data), "\n")
	// last line in file is always empty
	assert.Equal(t, len(events), len(payloadStrs)-1)
	var payload Payload
	for i, event := range events {
		err := json.Unmarshal([]byte(payloadStrs[i]), &payload)
		assert.NoError(t, err)
		assert.Equal(t, event.timestamp.Unix(), payload.Timestamp)
		assert.Equal(t, event.ipAddr, payload.IPAddress)
		assert.Equal(t, len(event.metrics), len(payload.MetricNames))
		assert.Equal(t, event.metrics[0].ID, payload.MetricNames[0])
	}
}

func BenchmarkFileWriterOnMetricsUpdate(b *testing.B) {
	b.StopTimer()
	l, _ := logging.NewZapLogger("info")
	fsw := NewFileWriter("fswtest.txt", l)
	defer os.Remove(fsw.filename)

	metrics := make([]*model.Metrics, 10)
	for i := 0; i < 10; i++ {
		metrics[i] = model.NewCounterMetrics(fmt.Sprintf("cnt%d", i))
	}
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		fsw.OnMetricsUpdate(time.Now(), "127.0.0.1", metrics...)
	}
}
