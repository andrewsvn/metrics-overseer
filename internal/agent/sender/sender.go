package sender

type MetricSendFunc func(id string, mtype string, value string) error

type MetricSender interface {
	MetricSendFunc() MetricSendFunc
}
