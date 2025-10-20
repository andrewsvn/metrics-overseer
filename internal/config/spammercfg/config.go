package spammercfg

import "flag"

// Config contains settings for metrics spammer tool regulating intensity and volume of metrics spamming
type Config struct {
	// metrics-overseer root URL to send spam requests
	URL string
	// NumWorkers is a number of simultaneous spammer threads
	NumWorkers int
	// BatchSize is a size of metrics batch sent in each request from each spammer thread.
	// If set to 1, a single metric request used to spam
	BatchSize int
	// SendMetricsIntervalMs specifies interval in milliseconds between sending metrics
	SendMetricsIntervalMs int
	// GetMetricsPageIntervalMs specifies interval in milliseconds between requesting metrics page.
	// This operation is done in a single routine and can be disabled by setting this interval to 0
	GetMetricsPageIntervalMs int
}

func Read() *Config {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	return cfg
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.URL, "url", "http://localhost:8080", "URL to send metrics to")
	flag.IntVar(&cfg.NumWorkers, "w", 10, "Number of workers")
	flag.IntVar(&cfg.BatchSize, "b", 10, "Batch size")
	flag.IntVar(&cfg.SendMetricsIntervalMs, "si", 500, "Interval between sends in milliseconds")
	flag.IntVar(&cfg.GetMetricsPageIntervalMs, "gi", 0, "Interval between checks in milliseconds (checks omitted if 0)")
}
