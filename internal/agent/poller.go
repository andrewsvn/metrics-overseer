package agent

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/agent/accumulation"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
	"go.uber.org/zap"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type Poller struct {
	interval time.Duration

	stor   *accumulation.Storage
	logger *zap.SugaredLogger
}

func NewPoller(cfg *agentcfg.Config, stor *accumulation.Storage, l *zap.Logger) *Poller {
	return &Poller{
		interval: time.Duration(cfg.PollIntervalSec) * time.Second,
		stor:     stor,
		logger:   l.Sugar().With("component", "agent-polling"),
	}
}

func (p *Poller) Start(ctx context.Context, wg *sync.WaitGroup) {
	p.startPollFunc(ctx, wg, p.execMemstatsPoll, time.NewTicker(p.interval))
	p.startPollFunc(ctx, wg, p.execGopsPoll, time.NewTicker(p.interval))
}

func (p *Poller) startPollFunc(ctx context.Context, wg *sync.WaitGroup, pf func(), ticker *time.Ticker) {
	wg.Add(1)
	go func() {
		defer ticker.Stop()
		defer wg.Done()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pf()
			}
		}
	}()
}

func (p *Poller) execMemstatsPoll() {
	p.logger.Info("polling memstats accumulation")

	ms := runtime.MemStats{}
	runtime.ReadMemStats(&ms)

	p.storeGaugeMetric("Alloc", float64(ms.Alloc))
	p.storeGaugeMetric("BuckHashSys", float64(ms.BuckHashSys))
	p.storeGaugeMetric("Frees", float64(ms.Frees))
	p.storeGaugeMetric("GCCPUFraction", ms.GCCPUFraction)
	p.storeGaugeMetric("GCSys", float64(ms.GCSys))
	p.storeGaugeMetric("HeapAlloc", float64(ms.HeapAlloc))
	p.storeGaugeMetric("HeapIdle", float64(ms.HeapIdle))
	p.storeGaugeMetric("HeapInuse", float64(ms.HeapInuse))
	p.storeGaugeMetric("HeapObjects", float64(ms.HeapObjects))
	p.storeGaugeMetric("HeapReleased", float64(ms.HeapReleased))
	p.storeGaugeMetric("HeapSys", float64(ms.HeapSys))
	p.storeGaugeMetric("LastGC", float64(ms.LastGC))
	p.storeGaugeMetric("Lookups", float64(ms.Lookups))
	p.storeGaugeMetric("MCacheInuse", float64(ms.MCacheInuse))
	p.storeGaugeMetric("MCacheSys", float64(ms.MCacheSys))
	p.storeGaugeMetric("MSpanInuse", float64(ms.MSpanInuse))
	p.storeGaugeMetric("MSpanSys", float64(ms.MSpanSys))
	p.storeGaugeMetric("Mallocs", float64(ms.Mallocs))
	p.storeGaugeMetric("NextGC", float64(ms.NextGC))
	p.storeGaugeMetric("NumForcedGC", float64(ms.NumForcedGC))
	p.storeGaugeMetric("NumGC", float64(ms.NumGC))
	p.storeGaugeMetric("OtherSys", float64(ms.OtherSys))
	p.storeGaugeMetric("PauseTotalNs", float64(ms.PauseTotalNs))
	p.storeGaugeMetric("StackInuse", float64(ms.StackInuse))
	p.storeGaugeMetric("StackSys", float64(ms.StackSys))
	p.storeGaugeMetric("Sys", float64(ms.Sys))
	p.storeGaugeMetric("TotalAlloc", float64(ms.TotalAlloc))

	p.storeGaugeMetric("RandomValue", rand.Float64())
	p.storeCounterMetric("PollCount", 1)
}

func (p *Poller) execGopsPoll() {
	p.logger.Info("polling gops accumulation")

	vmStat, err := mem.VirtualMemory()
	if err != nil {
		p.logger.Errorw("failed to get gops memory accumulation", "error", err)
		return
	}

	p.storeGaugeMetric("TotalMemory", float64(vmStat.Total))
	p.storeGaugeMetric("FreeMemory", float64(vmStat.Free))

	cpuUtils, err := cpu.Percent(0, true)
	if err != nil {
		p.logger.Errorw("failed to get gops cpu accumulation", "error", err)
		return
	}

	for id, cpuUtil := range cpuUtils {
		p.storeGaugeMetric(fmt.Sprintf("CPUutilization%d", id+1), cpuUtil)
	}
}

func (p *Poller) storeCounterMetric(id string, delta int64) {
	ma := p.stor.GetOrNew(id)
	err := ma.AccumulateCounter(delta)
	if err != nil {
		p.logger.Errorw("failed to store counter metric",
			"metric", id,
			"reason", err.Error(),
		)
	}
}

func (p *Poller) storeGaugeMetric(id string, value float64) {
	ma := p.stor.GetOrNew(id)
	err := ma.AccumulateGauge(value)
	if err != nil {
		p.logger.Errorw("failed to store gauge metric",
			"metric", id,
			"reason", err.Error(),
		)
	}
}
