package agent

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/andrewsvn/metrics-overseer/internal/agent/accumulation"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
)

type Agent struct {
	gracePeriod time.Duration

	pollr *Poller
	repr  *Reporter

	logger *zap.SugaredLogger
}

func NewAgent(cfg *agentcfg.Config, l *zap.Logger) (*Agent, error) {
	agentLogger := l.Sugar().With(zap.String("component", "agent"))
	agentLogger.Infow("initializing accumulation-overseer agent",
		"Start interval (sec)", cfg.PollIntervalSec,
		"report interval (sec)", cfg.ReportIntervalSec,
		"parallel report requests", cfg.MaxNumberOfRequests)

	stor := accumulation.NewAccumulatorStorage()
	pollr := NewPoller(cfg, stor, l)
	repr, err := NewReporter(cfg, stor, l)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric reporter: %w", err)
	}

	a := &Agent{
		gracePeriod: time.Duration(cfg.GracePeriodSec) * time.Second,
		pollr:       pollr,
		repr:        repr,
		logger:      agentLogger,
	}
	return a, nil
}

func (a *Agent) Run() {
	a.logger.Info("starting accumulation-overseer agent")

	//ctx, done := context.WithCancel(context.Background())
	ctx, cancelCtx := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer done()

	wg := &sync.WaitGroup{}
	a.pollr.Start(ctx, wg)
	a.repr.Start(ctx, wg)

	<-ctx.Done()

	a.logger.Info("shutting down accumulation-overseer agent...")
	_, shutdownCancel := context.WithTimeout(context.Background(), a.gracePeriod)
	defer shutdownCancel()

	wg.Wait()
	a.logger.Info("accumulation-overseer agent successfully stopped")
}
