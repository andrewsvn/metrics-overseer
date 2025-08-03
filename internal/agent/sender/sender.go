package sender

import (
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type MetricSender interface {
	SendMetricValue(id string, mtype string, value string) error
	SendMetric(metric *model.Metrics) error
	SendMetricArray(metrics []*model.Metrics) error
}
