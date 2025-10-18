package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/agent/sending"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"
)

var (
	url             string
	nWorkers        int
	batchSize       int
	sendIntervalMs  int
	sendInterval    time.Duration
	checkIntervalMs int
	checkInterval   time.Duration
)

func main() {
	flag.StringVar(&url, "url", "http://localhost:8080", "URL to send metrics to")
	flag.IntVar(&nWorkers, "w", 10, "Number of workers")
	flag.IntVar(&batchSize, "b", 10, "Batch size")
	flag.IntVar(&sendIntervalMs, "i", 500, "Interval between sends in milliseconds")
	flag.IntVar(&checkIntervalMs, "c", 500, "Interval between checks in milliseconds")
	flag.Parse()

	sendInterval = time.Duration(sendIntervalMs) * time.Millisecond
	checkInterval = time.Duration(checkIntervalMs) * time.Millisecond

	l, err := logging.NewZapLogger("error")
	if err != nil {
		log.Fatalf("error initializing logger: %v", err)
	}

	sender, err := sending.NewRestSender(url, &retrying.NoRetryPolicy{}, "", l)
	if err != nil {
		log.Fatalf("error initializing sender: %v", err)
	}

	ctx, done := context.WithCancel(context.Background())

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	l.Info("starting metrics spammer", zap.String("url", url), zap.Int("workersCount", nWorkers),
		zap.Int("batchSize", batchSize), zap.Int("sendIntervalMs", sendIntervalMs))

	// senders - to check storage write
	for i := 0; i < nWorkers; i++ {
		go worker(ctx, sender, l)
	}

	// checker - to check storage read, serialization and compressing
	go checker(ctx, l)

	<-stop
	done()
	l.Info("stopping metrics spammer")
}

func worker(ctx context.Context, sender *sending.RestSender, l *zap.Logger) {
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if batchSize > 1 {
			err := sendBatch(sender, rnd, batchSize)
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

func sendBatch(sender *sending.RestSender, rnd *rand.Rand, batchSize int) error {
	metrics := make([]*model.Metrics, batchSize)
	for i := 0; i < batchSize; i++ {
		metrics[i] = generateMetric(rnd)
	}
	return sender.SendMetricArray(metrics)
}

func sendSingle(sender *sending.RestSender, rnd *rand.Rand) error {
	metric := generateMetric(rnd)
	return sender.SendMetric(metric)
}

func generateMetric(rnd *rand.Rand) *model.Metrics {
	return model.NewCounterMetricsWithDelta(
		fmt.Sprintf("cnt_%d", rnd.Int63()),
		rnd.Int63n(100),
	)
}

func checker(ctx context.Context, l *zap.Logger) {
	cl := resty.New()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		req := cl.R()
		req.Header.Add("Accept-Encoding", "gzip")
		_, err := req.Get(url)
		if err != nil {
			l.Error("error checking metrics", zap.Error(err))
		}

		time.Sleep(checkInterval)
	}
}
