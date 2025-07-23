package sender

import (
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type MetricValueSendFunc func(id string, mtype string, value string) error
type MetricStructSendFunc func(metric *model.Metrics) error

type MetricSender interface {
	ValueSendFunc() MetricValueSendFunc
	StructSendFunc() MetricStructSendFunc
}
