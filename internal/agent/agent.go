package agent

import (
	"fmt"
	"go.uber.org/zap"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/agent/metrics"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type Agent struct {
	pollInterval   time.Duration
	reportInterval time.Duration

	accums map[string]*metrics.MetricAccumulator
	sndr   sender.MetricSender
	logger *zap.SugaredLogger
}

func NewAgent(cfg *agentcfg.Config, logger *zap.Logger) (*Agent, error) {
	agentLogger := logger.Sugar().With(zap.String("component", "agent"))

	serverAddr := strings.Trim(cfg.ServerAddr, "\"")
	sndr, err := sender.NewRestSender(serverAddr, agentLogger.Desugar())
	if err != nil {
		return nil, fmt.Errorf("can't construct agent from config: %w", err)
	}

	agentLogger.Infow("Initializing metrics-overseer agent",
		"poll interval (sec)", cfg.PollIntervalSec,
		"report interval (sec)", cfg.ReportIntervalSec)

	a := &Agent{
		pollInterval:   time.Duration(cfg.PollIntervalSec) * time.Second,
		reportInterval: time.Duration(cfg.ReportIntervalSec) * time.Second,

		accums: make(map[string]*metrics.MetricAccumulator),
		sndr:   sndr,
		logger: agentLogger,
	}
	return a, nil
}

func (a *Agent) Run() {
	a.logger.Info("Starting metrics-overseer agent")

	pollTicker := time.NewTicker(a.pollInterval)
	reportTicker := time.NewTicker(a.reportInterval)

	go a.poll(pollTicker.C)
	go a.report(reportTicker.C)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	a.logger.Info("Shutting down metrics-overseer agent")
}

func (a *Agent) poll(tc <-chan time.Time) {
	for {
		<-tc
		a.execPoll()
	}
}

func (a *Agent) execPoll() {
	a.logger.Info("Polling metrics")

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

func (a *Agent) report(tc <-chan time.Time) {
	for {
		<-tc
		a.execReport()
	}
}

func (a *Agent) execReport() {
	a.logger.Info("Reporting metrics to server")
	marray := make([]*model.Metrics, 0, len(a.accums))
	for _, ma := range a.accums {
		metric, err := ma.StageChanges()
		if err != nil {
			a.logger.Errorw("unable to stage metric for sending",
				"metric", ma.ID,
				"error", err,
			)
			continue
		}
		defer func(ma *metrics.MetricAccumulator) {
			_ = ma.RollbackStaged()
		}(ma)

		if metric != nil {
			marray = append(marray, metric)
		}
	}

	err := a.sndr.SendMetricArray(marray)
	if err != nil {
		a.logger.Errorw("unable to send metrics to server",
			"error", err,
		)
		return
	}

	for _, m := range marray {
		err := a.accums[m.ID].CommitStaged()
		if err != nil {
			a.logger.Errorw("unable to commit staged metric",
				"metric", m.ID,
				"error", err)
		}
	}
}

func (a *Agent) storeCounterMetric(id string, delta int64) {
	ma, exist := a.accums[id]
	if !exist {
		ma = metrics.NewMetricAccumulator(id, model.Counter)
		a.accums[id] = ma
	}
	err := ma.AccumulateCounter(delta)
	if err != nil {
		a.logger.Errorw("failed to store counter metric",
			"metric", id,
			"reason", err.Error(),
		)
	}
}

func (a *Agent) storeGaugeMetric(id string, value float64) {
	ma, exist := a.accums[id]
	if !exist {
		ma = metrics.NewMetricAccumulator(id, model.Gauge)
		a.accums[id] = ma
	}
	err := ma.AccumulateGauge(value)
	if err != nil {
		a.logger.Errorw("failed to store gauge metric",
			"metric", id,
			"reason", err.Error(),
		)
	}
}
