package service

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"go.uber.org/zap"
)

//go:embed resources/metricspage.html
var metricspage string

type MetricsService struct {
	storage        repository.Storage
	auditors       []Auditor
	allMetricsTmpl *template.Template
	logger         *zap.SugaredLogger
}

var (
	ErrUnsupportedMetricType  = errors.New("unsupported metric type")
	ErrMetricValueNotProvided = errors.New("metric value not provided")
)

type MetricsPage struct {
	Metrics []*model.Metrics
}

func NewMetricsService(st repository.Storage, l *zap.Logger) *MetricsService {
	return &MetricsService{
		storage: st,
		logger:  l.Sugar().With("component", "metrics-service"),
	}
}

func (ms *MetricsService) SubscribeAuditor(auditor Auditor) {
	ms.auditors = append(ms.auditors, auditor)
}

// AccumulateMetric is an aggregated method of updating metric value based on metric type provided
// for Counter metric it adds delta value to existing metric value (or creates a new one in storage if not exists)
// for Gauge metric it simply stores gauge value, overwriting an existing one
func (ms *MetricsService) AccumulateMetric(ctx context.Context, metric *model.Metrics, ipAddr string) error {
	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			return ErrMetricValueNotProvided
		}
		if err := ms.storage.AddCounter(ctx, metric.ID, *metric.Delta); err != nil {
			return fmt.Errorf("unable to update metric: %w", err)
		}
	case model.Gauge:
		if metric.Value == nil {
			return ErrMetricValueNotProvided
		}
		if err := ms.storage.SetGauge(ctx, metric.ID, *metric.Value); err != nil {
			return fmt.Errorf("unable to update metric: %w", err)
		}
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedMetricType, metric.MType)
	}

	ms.notifyAuditors(ipAddr, metric)
	return nil
}

func (ms *MetricsService) GetMetric(ctx context.Context, id, mtype string) (*model.Metrics, error) {
	mi, err := ms.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if mi.MType != mtype {
		return nil, repository.ErrIncorrectAccess
	}
	return mi, nil
}

func (ms *MetricsService) BatchAccumulateMetrics(ctx context.Context, metrics []*model.Metrics, ipAddr string) error {
	err := ms.storage.BatchUpdate(ctx, metrics)
	if err != nil {
		return fmt.Errorf("failed to store metric values: %w", err)
	}

	ms.notifyAuditors(ipAddr, metrics...)
	return nil
}

func (ms *MetricsService) GenerateAllMetricsHTML(ctx context.Context, w io.Writer) error {
	if ms.allMetricsTmpl == nil {
		tmpl := template.New("metricspage")
		tmpl, err := tmpl.Parse(metricspage)
		if err != nil {
			return fmt.Errorf("error parsing page template: %w", err)
		}
		ms.allMetricsTmpl = tmpl
	}

	metrics, err := ms.storage.GetAllSorted(ctx)
	if err != nil {
		return fmt.Errorf("can't get all metrics from storage: %w", err)
	}

	page := MetricsPage{
		Metrics: metrics,
	}
	return ms.allMetricsTmpl.Execute(w, page)
}

func (ms *MetricsService) PingStorage(ctx context.Context) error {
	return ms.storage.Ping(ctx)
}

func (ms *MetricsService) notifyAuditors(ipAddr string, metrics ...*model.Metrics) {
	ts := time.Now()
	for _, auditor := range ms.auditors {
		err := auditor.OnMetricsUpdate(ts, ipAddr, metrics...)
		if err != nil {
			ms.logger.Errorw("error performing metrics audit", "error", err)
		}
	}
}
