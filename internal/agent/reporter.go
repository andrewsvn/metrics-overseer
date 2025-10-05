package agent

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/agent/accumulation"
	"github.com/andrewsvn/metrics-overseer/internal/agent/reporting"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sending"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"go.uber.org/zap"
	"strings"
	"sync"
	"time"
)

type Reporter struct {
	interval time.Duration

	stor        *accumulation.Storage
	executor    reporting.Executor
	reportMutex sync.Mutex

	logger *zap.SugaredLogger
}

func NewReporter(cfg *agentcfg.Config, storage *accumulation.Storage, l *zap.Logger) (*Reporter, error) {
	reportRetryPolicy := retrying.NewLinearPolicy(
		cfg.MaxRetryCount,
		time.Duration(cfg.InitialRetryDelaySec)*time.Second,
		time.Duration(cfg.RetryDelayIncrementSec)*time.Second,
	)

	serverAddr := strings.Trim(cfg.ServerAddr, "\"")
	sndr, err := sending.NewRestSender(serverAddr, reportRetryPolicy, cfg.SecretKey, l)
	if err != nil {
		return nil, fmt.Errorf("failed to create sender: %w", err)
	}

	rLogger := l.Sugar().With("component", "agent-reporting")
	return &Reporter{
		interval: time.Duration(cfg.ReportIntervalSec) * time.Second,
		stor:     storage,
		executor: newExecutor(cfg, sndr, rLogger),
		logger:   rLogger,
	}, nil
}

func newExecutor(cfg *agentcfg.Config, sndr sending.MetricSender, sl *zap.SugaredLogger) reporting.Executor {
	if cfg.MaxNumberOfRequests > 0 {
		return reporting.NewWorkerPoolExecutor(cfg.MaxNumberOfRequests, sndr, sl)
	}
	return reporting.NewBatchExecutor(sndr, sl)
}

func (r *Reporter) Start(ctx context.Context, wg *sync.WaitGroup) {
	ticker := time.NewTicker(r.interval)

	wg.Add(1)
	go func() {
		defer ticker.Stop()
		defer r.executor.Shutdown()
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				r.execReport()
				return
			case <-ticker.C:
				r.execReport()
			}
		}
	}()
}

func (r *Reporter) execReport() {
	r.reportMutex.Lock()
	defer r.reportMutex.Unlock()

	r.logger.Info("reporting metrics to server")
	marray := make([]*model.Metrics, 0)
	for ma := range r.stor.GetAll() {
		metric, err := ma.StageChanges()
		if err != nil {
			r.logger.Errorw("unable to stage metric for sending",
				"metric", ma.ID,
				"error", err,
			)
			continue
		}

		if metric != nil {
			marray = append(marray, metric)
		}
	}

	result := r.executor.Execute(marray)

	for _, id := range result.SuccessIDs {
		err := r.stor.Get(id).CommitStaged()
		if err != nil {
			r.logger.Errorw("unable to commit staged metric",
				"metric", id,
				"error", err)
		}
	}
	for _, id := range result.FailureIDs {
		err := r.stor.Get(id).RollbackStaged()
		if err != nil {
			r.logger.Errorw("unable to rollback staged metric",
				"metric", id,
				"error", err)
		}
	}
}
