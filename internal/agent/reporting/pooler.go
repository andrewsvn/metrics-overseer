package reporting

import (
	"github.com/andrewsvn/metrics-overseer/internal/agent/sending"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
)

type WorkerPoolExecutor struct {
	rate    int
	mSender sending.MetricSender
	tasks   chan *model.Metrics
	results chan *Result

	logger *zap.SugaredLogger
}

func NewWorkerPoolExecutor(rate int, sndr sending.MetricSender, sl *zap.SugaredLogger) *WorkerPoolExecutor {
	e := &WorkerPoolExecutor{
		rate:    rate,
		mSender: sndr,
		tasks:   make(chan *model.Metrics, rate),
		results: make(chan *Result, rate),
		logger:  sl,
	}

	for i := 0; i < rate; i++ {
		go e.worker(i)
	}

	return e
}

func (e *WorkerPoolExecutor) Execute(ms []*model.Metrics) *Result {
	e.logger.Debugw("worker pool report execute started", "count", len(ms))
	go func() {
		for _, m := range ms {
			e.tasks <- m
		}
	}()

	result := &Result{}
	for i := 0; i < len(ms); i++ {
		result.Append(<-e.results)
	}
	e.logger.Debugw("worker pool report execute finished")

	return result
}

func (e *WorkerPoolExecutor) Shutdown() {
	close(e.tasks)
	close(e.results)
}

func (e *WorkerPoolExecutor) worker(id int) {
	for m := range e.tasks {
		err := e.mSender.SendMetric(m)
		if err != nil {
			e.logger.Errorw("unable to send metric to server",
				"metric", m.ID,
				"workerID", id,
				"error", err)
			e.results <- FailureResult(m.ID)
			continue
		}
		e.logger.Debugw("metric sent to server",
			"metric", m.ID,
			"worker", id)
		e.results <- SuccessResult(m.ID)
	}
}
