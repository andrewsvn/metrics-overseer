package service

import (
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type Auditor interface {
	OnMetricsUpdate(ts time.Time, ipAddr string, metrics ...*model.Metrics) error
}
