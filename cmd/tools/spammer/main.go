// Spammer is a simple tool that can spam metrics-overseer server with a lot of simultaneous requests
// with metric updates and optional metrics page requests
// see spammercfg.Config for configurable parameters and flags
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"time"

	"math/rand/v2"

	"github.com/andrewsvn/metrics-overseer/internal/agent/sending"
	"github.com/andrewsvn/metrics-overseer/internal/config/spammercfg"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

func main() {
	cfg := spammercfg.Read()

	l, err := logging.NewZapLogger("error")
	if err != nil {
		log.Fatalf("error initializing logger: %v", err)
	}

	sender, err := sending.NewRestSender(cfg.URL, &retrying.NoRetryPolicy{}, "", "", l)
	if err != nil {
		log.Fatalf("error initializing sender: %v", err)
	}

	ctx, done := context.WithCancel(context.Background())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	l.Info("starting metrics spammer", zap.String("url", cfg.URL), zap.Int("workersCount", cfg.NumWorkers),
		zap.Int("batchSize", cfg.BatchSize), zap.Int("sendIntervalMs", cfg.SendMetricsIntervalMs),
		zap.Int("getPageIntervalMs", cfg.GetMetricsPageIntervalMs))

	// senders - to check storage write
	for i := 0; i < cfg.NumWorkers; i++ {
		go worker(ctx, cfg, sender, l)
	}

	// checker - to check storage read, serialization and compressing
	if cfg.GetMetricsPageIntervalMs > 0 {
		go checker(ctx, cfg, l)
	}

	<-stop
	done()
	l.Info("stopping metrics spammer")
}

func worker(ctx context.Context, cfg *spammercfg.Config, sender *sending.RestSender, l *zap.Logger) {
	sendInterval := time.Duration(cfg.SendMetricsIntervalMs) * time.Millisecond
	rnd := rand.NewPCG(uint64(time.Now().UnixNano()), 0x8eed)

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if cfg.BatchSize > 1 {
			err := sendBatch(sender, rnd, cfg.BatchSize)
			if err != nil {
				l.Error("error sending batch", zap.Error(err))
			}
		} else {
			err := sendSingle(sender, rnd)
			if err != nil {
				l.Error("error sending metric", zap.Error(err))
			}
		}
		time.Sleep(sendInterval)
	}
}

func sendBatch(sender *sending.RestSender, rnd *rand.PCG, batchSize int) error {
	metrics := make([]*model.Metrics, batchSize)
	for i := 0; i < batchSize; i++ {
		metrics[i] = generateMetric(rnd)
	}
	return sender.SendMetricArray(metrics)
}

func sendSingle(sender *sending.RestSender, rnd *rand.PCG) error {
	metric := generateMetric(rnd)
	return sender.SendMetric(metric)
}

func generateMetric(rnd *rand.PCG) *model.Metrics {
	return model.NewCounterMetricsWithDelta(
		fmt.Sprintf("cnt_%d", rnd.Uint64()),
		int64(rnd.Uint64()%100),
	)
}

func checker(ctx context.Context, cfg *spammercfg.Config, l *zap.Logger) {
	getPageInterval := time.Duration(cfg.GetMetricsPageIntervalMs) * time.Millisecond
	cl := resty.New()

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		req := cl.R()
		// no gzip here to disable compress benchmarking
		_, err := req.Get(cfg.URL)
		if err != nil {
			l.Error("error checking metrics", zap.Error(err))
		}

		time.Sleep(getPageInterval)
	}
}
