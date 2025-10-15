package audit

import (
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

type HTTPWriter struct {
	url    string
	logger *zap.SugaredLogger
}

func NewHTTPWriter(url string, l *zap.Logger) *HTTPWriter {
	return &HTTPWriter{
		url:    url,
		logger: l.Sugar().With(zap.String("component", "audit-httpwriter")),
	}
}

func (hw HTTPWriter) OnMetricsUpdate(ts time.Time, ipAddr string, metrics ...*model.Metrics) {
	req := resty.New().R()
	req.URL = hw.url
	req.SetHeader("Content-Type", "application/json")
	req.SetBody(NewPayload(ts, ipAddr, metrics...))

	_, err := req.Post(hw.url)
	if err != nil {
		hw.logger.Error("Failed to send metrics to audit service", "url", hw.url, "error", err)
	}
}
