package agent

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/agent/reporting"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/agent/metrics"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type Agent struct {
	pollInterval   time.Duration
	reportInterval time.Duration
	gracePeriod    time.Duration

	accums      *metrics.AccumulatorStorage
	mSender     sender.MetricSender
	reporter    reporting.Reporter
	reportMutex sync.Mutex

	logger *zap.SugaredLogger
}

func NewAgent(cfg *agentcfg.Config, logger *zap.Logger) (*Agent, error) {
	agentLogger := logger.Sugar().With(zap.String("component", "agent"))

	reportRetryPolicy := retrying.NewLinearPolicy(
		cfg.MaxRetryCount,
		time.Duration(cfg.InitialRetryDelaySec)*time.Second,
		time.Duration(cfg.RetryDelayIncrementSec)*time.Second,
	)

	serverAddr := strings.Trim(cfg.ServerAddr, "\"")
	sndr, err := sender.NewRestSender(serverAddr, logger, reportRetryPolicy, cfg.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("can't construct agent from config: %w", err)
	}

	reporter := newReporter(cfg, logger)

	agentLogger.Infow("initializing metrics-overseer agent",
		"memPoll interval (sec)", cfg.PollIntervalSec,
		"report interval (sec)", cfg.ReportIntervalSec,
		"parallel report requests", cfg.MaxNumberOfRequests)

	a := &Agent{
		pollInterval:   time.Duration(cfg.PollIntervalSec) * time.Second,
		reportInterval: time.Duration(cfg.ReportIntervalSec) * time.Second,
		gracePeriod:    time.Duration(cfg.GracePeriodSec) * time.Second,

		accums:   metrics.NewAccumulatorStorage(),
		mSender:  sndr,
		reporter: reporter,

		logger: agentLogger,
	}
	return a, nil
}

func newReporter(cfg *agentcfg.Config, l *zap.Logger) reporting.Reporter {
	if cfg.MaxNumberOfRequests > 0 {
		return reporting.NewPoolReporter(cfg.MaxNumberOfRequests, l)
	}
	return reporting.NewBatchReporter(l)
}

func (a *Agent) Run() {
	a.logger.Info("starting metrics-overseer agent")

	ctx, done := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	memPollTicker := time.NewTicker(a.pollInterval)
	gopsPollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)

	wg := &sync.WaitGroup{}
	go a.memPoll(ctx, wg, memPollTicker.C)
	go a.gopsPoll(ctx, wg, gopsPollTicker.C)
	go a.report(ctx, wg, reportTicker.C)

	<-stop
	done()

	a.logger.Info("shutting down metrics-overseer agent...")
	_, shutdownCancel := context.WithTimeout(context.Background(), a.gracePeriod)
	defer shutdownCancel()

	wg.Wait()
	a.logger.Info("metrics-overseer agent successfully stopped")
}

func (a *Agent) memPoll(ctx context.Context, wg *sync.WaitGroup, tc <-chan time.Time) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tc:
			a.execMemstatsPoll()
		}
	}
}

func (a *Agent) execMemstatsPoll() {
	a.logger.Info("polling memstats metrics")

	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)

	a.storeGaugeMetric("Alloc", float64(ms.Alloc))
	a.storeGaugeMetric("BuckHashSys", float64(ms.BuckHashSys))
	a.storeGaugeMetric("Frees", float64(ms.Frees))
	a.storeGaugeMetric("GCCPUFraction", ms.GCCPUFraction)
	a.storeGaugeMetric("GCSys", float64(ms.GCSys))
	a.storeGaugeMetric("HeapAlloc", float64(ms.HeapAlloc))
	a.storeGaugeMetric("HeapIdle", float64(ms.HeapIdle))
	a.storeGaugeMetric("HeapInuse", float64(ms.HeapInuse))
	a.storeGaugeMetric("HeapObjects", float64(ms.HeapObjects))
	a.storeGaugeMetric("HeapReleased", float64(ms.HeapReleased))
	a.storeGaugeMetric("HeapSys", float64(ms.HeapSys))
	a.storeGaugeMetric("LastGC", float64(ms.LastGC))
	a.storeGaugeMetric("Lookups", float64(ms.Lookups))
	a.storeGaugeMetric("MCacheInuse", float64(ms.MCacheInuse))
	a.storeGaugeMetric("MCacheSys", float64(ms.MCacheSys))
	a.storeGaugeMetric("MSpanInuse", float64(ms.MSpanInuse))
	a.storeGaugeMetric("MSpanSys", float64(ms.MSpanSys))
	a.storeGaugeMetric("Mallocs", float64(ms.Mallocs))
	a.storeGaugeMetric("NextGC", float64(ms.NextGC))
	a.storeGaugeMetric("NumForcedGC", float64(ms.NumForcedGC))
	a.storeGaugeMetric("NumGC", float64(ms.NumGC))
	a.storeGaugeMetric("OtherSys", float64(ms.OtherSys))
	a.storeGaugeMetric("PauseTotalNs", float64(ms.PauseTotalNs))
	a.storeGaugeMetric("StackInuse", float64(ms.StackInuse))
	a.storeGaugeMetric("StackSys", float64(ms.StackSys))
	a.storeGaugeMetric("Sys", float64(ms.Sys))
	a.storeGaugeMetric("TotalAlloc", float64(ms.TotalAlloc))

	a.storeGaugeMetric("RandomValue", rand.Float64())
	a.storeCounterMetric("PollCount", 1)
}

func (a *Agent) gopsPoll(ctx context.Context, wg *sync.WaitGroup, tc <-chan time.Time) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tc:
			a.execGopsPoll()
		}
	}
}

func (a *Agent) execGopsPoll() {
	a.logger.Info("polling gops metrics")

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		a.logger.Errorw("failed to get gops memory metrics", "error", err)
		return
	}

	a.storeGaugeMetric("TotalMemory", float64(vmStat.Total))
	a.storeGaugeMetric("FreeMemory", float64(vmStat.Free))

	cpuUtils, err := cpu.Percent(0, true)
	if err != nil {
		a.logger.Errorw("failed to get gops cpu metrics", "error", err)
		return
	}

	for id, cpuUtil := range cpuUtils {
		a.storeGaugeMetric(fmt.Sprintf("CPUutilization%d", id+1), cpuUtil)
	}
}

func (a *Agent) report(ctx context.Context, wg *sync.WaitGroup, tc <-chan time.Time) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-ctx.Done():
			a.execReport()
			return
		case <-tc:
			a.execReport()
		}
	}
}

func (a *Agent) execReport() {
	a.reportMutex.Lock()
	defer a.reportMutex.Unlock()

	a.logger.Info("reporting metrics to server")
	marray := make([]*model.Metrics, 0)
	for ma := range a.accums.GetAll() {
		metric, err := ma.StageChanges()
		if err != nil {
			a.logger.Errorw("unable to stage metric for sending",
				"metric", ma.ID,
				"error", err,
			)
			continue
		}

		if metric != nil {
			marray = append(marray, metric)
		}
	}

	result := a.reporter.Execute(context.Background(), a.mSender, marray)

	for _, id := range result.SuccessIDs {
		err := a.accums.Get(id).CommitStaged()
		if err != nil {
			a.logger.Errorw("unable to commit staged metric",
				"metric", id,
				"error", err)
		}
	}
	for _, id := range result.FailureIDs {
		err := a.accums.Get(id).RollbackStaged()
		if err != nil {
			a.logger.Errorw("unable to rollback staged metric",
				"metric", id,
				"error", err)
		}
	}
}

func (a *Agent) storeCounterMetric(id string, delta int64) {
	ma := a.accums.GetOrNew(id)
	err := ma.AccumulateCounter(delta)
	if err != nil {
		a.logger.Errorw("failed to store counter metric",
			"metric", id,
			"reason", err.Error(),
		)
	}
}

func (a *Agent) storeGaugeMetric(id string, value float64) {
	ma := a.accums.GetOrNew(id)
	err := ma.AccumulateGauge(value)
	if err != nil {
		a.logger.Errorw("failed to store gauge metric",
			"metric", id,
			"reason", err.Error(),
		)
	}
}
