package agent

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/agent/metrics"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	config "github.com/andrewsvn/metrics-overseer/internal/config/agent"
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type Agent struct {
	serverHost     string
	serverPort     int32
	pollInterval   int64
	reportInterval int64

	accums   map[string]*metrics.MetricAccumulator
	sendfunc sender.MetricSendFunc
}

func NewAgent() *Agent {
	sndr := sender.NewRestSender(config.ServerHost, config.ServerPort)

	return &Agent{
		serverHost:     config.ServerHost,
		serverPort:     config.ServerPort,
		pollInterval:   config.PollIntervalMs,
		reportInterval: config.ReportIntervalMs,

		accums:   make(map[string]*metrics.MetricAccumulator),
		sendfunc: sndr.MetricSendFunc(),
	}
}

func (a *Agent) Start() {
	log.Printf("[INFO] Starting metrics-overseer agent")
	log.Printf("[INFO] Agent poll interval = %d ms, report interval = %d ms",
		a.pollInterval, a.reportInterval)
	log.Printf("[INFO] Agent reporting server: %s:%d", a.serverHost, a.serverPort)

	go a.poll()
	go a.report()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch
}

func (a *Agent) poll() {
	for {
		time.Sleep(time.Duration(a.pollInterval) * time.Millisecond)
		a.execPoll()
	}
}

func (a *Agent) execPoll() {
	log.Printf("[INFO] Polling metrics")

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

func (a *Agent) report() {
	for {
		time.Sleep(time.Duration(a.reportInterval) * time.Millisecond)
		a.execReport()
	}
}

func (a *Agent) execReport() {
	log.Printf("[INFO] Reporting metrics to server")
	for name, ma := range a.accums {
		err := ma.ExtractAndSend(a.sendfunc)
		if err != nil {
			log.Printf("[ERROR] unable to send metric %s to server", name)
		}
	}
}

func (a *Agent) storeCounterMetric(id string, delta int64) {
	// TODO: debug
	log.Printf("[DEBUG] Storing metric %s, value %d", id, delta)

	ma, exist := a.accums[id]
	if !exist {
		ma = metrics.NewMetricAccumulator(id, model.Counter)
		a.accums[id] = ma
	}
	err := ma.AccumulateCounter(delta)
	if err != nil {
		log.Printf("[ERROR] failed to store metric '%s', reason: %v\n", id, err)
	}
}

func (a *Agent) storeGaugeMetric(id string, value float64) {
	// TODO: debug
	log.Printf("[DEBUG] Storing metric %s, value %f", id, value)

	ma, exist := a.accums[id]
	if !exist {
		ma = metrics.NewMetricAccumulator(id, model.Gauge)
		a.accums[id] = ma
	}
	err := ma.AccumulateGauge(value)
	if err != nil {
		log.Printf("[ERROR] failed to store metric '%s', reason: %v\n", id, err)
	}
}
