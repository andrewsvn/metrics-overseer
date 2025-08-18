package reporting

import (
	"context"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
)

type PoolReporter struct {
	rate    int
	tasks   chan *model.Metrics
	results chan *Result

	logger *zap.SugaredLogger
}

func NewPoolReporter(rate int, logger *zap.Logger) *PoolReporter {
	return &PoolReporter{
		rate:    rate,
		tasks:   make(chan *model.Metrics, rate),
		results: make(chan *Result, rate),

		logger: logger.Sugar().With("component", "pool-reporter"),
	}
}

func (pr *PoolReporter) Execute(ctx context.Context, sndr sender.MetricSender, ms []*model.Metrics) *Result {
	for i := 0; i < pr.rate; i++ {
		go pr.worker(ctx, sndr, i)
	}

	pr.logger.Debugw("pool reporter execute started", "count", len(ms))
	go func() {
		for _, m := range ms {
			pr.tasks <- m
		}
	}()

	result := &Result{}
	for i := 0; i < len(ms); i++ {
		result.Append(<-pr.results)
	}
	pr.logger.Debugw("pool reporter execute finished", "count", len(ms))

	return result
}

func (pr *PoolReporter) worker(ctx context.Context, sndr sender.MetricSender, id int) {
	for {
		select {
		case <-ctx.Done():
			return
		case m := <-pr.tasks:
			err := sndr.SendMetric(m)
			if err != nil {
				pr.logger.Errorw("unable to send metric to server",
					"metric", m.ID,
					"workerID", id,
					"error", err)
				pr.results <- FailureResult(m.ID)
				continue
			}
			pr.logger.Debugw("metric sent to server",
				"metric", m.ID,
				"worker", id)
			pr.results <- SuccessResult(m.ID)
		}
	}
}
