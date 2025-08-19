package reporting

import (
	"github.com/andrewsvn/metrics-overseer/internal/agent/sending"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
)

// BatchExecutor sends all metrics as one big batch update
// either all metrics will be successfully sent or none of them at all
type BatchExecutor struct {
	mSender sending.MetricSender
	logger  *zap.SugaredLogger
}

func NewBatchExecutor(sndr sending.MetricSender, sl *zap.SugaredLogger) *BatchExecutor {
	return &BatchExecutor{
		mSender: sndr,
		logger:  sl,
	}
}

func (e *BatchExecutor) Execute(ms []*model.Metrics) *Result {
	ids := make([]string, 0, len(ms))
	for _, m := range ms {
		ids = append(ids, m.ID)
	}

	e.logger.Debugw("batch report execute started", "count", len(ms))
	defer e.logger.Debugw("batch report execute finished", "count", len(ms))

	err := e.mSender.SendMetricArray(ms)
	if err != nil {
		e.logger.Errorw("unable to send metrics to server",
			"error", err,
		)
		return FailureResult(ids...)
	}

	return SuccessResult(ids...)
}

func (e *BatchExecutor) Shutdown() {
}
