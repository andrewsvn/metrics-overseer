package reporting

import (
	"context"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
)

// BatchReporter sends all metrics as one big batch update
// either all metrics will be successfully sent or none of them at all
type BatchReporter struct {
	logger *zap.SugaredLogger
}

func NewBatchReporter(l *zap.Logger) *BatchReporter {
	return &BatchReporter{
		logger: l.Sugar().With("package", "batch-reporter"),
	}
}

func (br *BatchReporter) Execute(_ context.Context, sndr sender.MetricSender, ms []*model.Metrics) *Result {
	ids := make([]string, 0, len(ms))
	for _, m := range ms {
		ids = append(ids, m.ID)
	}

	br.logger.Debugw("batch reporter execute started", "count", len(ms))
	defer br.logger.Debugw("batch reporter execute finished", "count", len(ms))

	err := sndr.SendMetricArray(ms)
	if err != nil {
		br.logger.Errorw("unable to send metrics to server",
			"error", err,
		)
		return FailureResult(ids...)
	}

	return SuccessResult(ids...)
}
