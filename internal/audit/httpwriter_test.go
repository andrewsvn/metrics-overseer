package audit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestHTTPWriterOnMetricsUpdate(t *testing.T) {
	metrics := []*model.Metrics{
		model.NewCounterMetrics("cnt1"),
		model.NewCounterMetrics("cnt2"),
		model.NewGaugeMetrics("gauge1"),
		model.NewGaugeMetrics("gauge2"),
	}

	ts := time.Now()
	// use ipAddr as an index to check what event server has received
	events := []struct {
		timestamp time.Time
		ipAddr    string
		metrics   []*model.Metrics
	}{
		{
			timestamp: ts,
			ipAddr:    "0",
			metrics:   metrics[0:2],
		},
		{
			timestamp: ts.Add(5 * time.Second),
			ipAddr:    "1",
			metrics:   metrics[1:],
		},
		{
			timestamp: ts.Add(2 * time.Minute),
			ipAddr:    "2",
			metrics:   metrics[:1],
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body Payload
		err := json.NewDecoder(r.Body).Decode(&body)
		assert.NoError(t, err)

		id, err := strconv.ParseInt(body.IPAddress, 10, 64)
		assert.NoError(t, err)
		assert.Less(t, int(id), len(events))

		assert.Equal(t, events[id].timestamp.Unix(), body.Timestamp)
		assert.Equal(t, len(events[id].metrics), len(body.MetricNames))
		for i, metric := range body.MetricNames {
			assert.Equal(t, events[id].metrics[i].ID, metric)
		}

		w.WriteHeader(http.StatusOK)
	}))

	l, _ := logging.NewZapLogger("info")
	httpw := NewHTTPWriter(srv.URL, l)
	for _, event := range events {
		httpw.OnMetricsUpdate(event.timestamp, event.ipAddr, event.metrics...)
	}
}
