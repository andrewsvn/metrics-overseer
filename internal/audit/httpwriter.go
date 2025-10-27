package audit

import (
	"fmt"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/go-resty/resty/v2"
)

type HTTPWriter struct {
	url string
}

func NewHTTPWriter(url string) *HTTPWriter {
	return &HTTPWriter{
		url: url,
	}
}

func (hw *HTTPWriter) OnMetricsUpdate(ts time.Time, ipAddr string, metrics ...*model.Metrics) error {
	req := resty.New().R()
	req.URL = hw.url
	req.SetHeader("Content-Type", "application/json")
	req.SetBody(NewPayload(ts, ipAddr, metrics...))

	_, err := req.Post(hw.url)
	if err != nil {
		return fmt.Errorf("failed to send metrics to audit service %s", hw.url)
	}
	return nil
}
