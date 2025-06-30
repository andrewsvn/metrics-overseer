package sender

type MetricSendFunc func(name string, value string) error
