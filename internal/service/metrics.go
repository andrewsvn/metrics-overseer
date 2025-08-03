package service

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"html/template"
	"io"
)

//go:embed resources/metricspage.html
var metricspage string

type MetricsService struct {
	storage        repository.Storage
	allMetricsTmpl *template.Template
}

type MetricsPage struct {
	Metrics []*model.Metrics
}

func NewMetricsService(st repository.Storage) *MetricsService {
	return &MetricsService{
		storage: st,
	}
}

func (ms *MetricsService) AccumulateCounter(ctx context.Context, id string, inc int64) error {
	err := ms.storage.AddCounter(ctx, id, inc)
	if err != nil {
		return err
	}

	return nil
}

func (ms *MetricsService) SetGauge(ctx context.Context, id string, val float64) error {
	err := ms.storage.SetGauge(ctx, id, val)
	if err != nil {
		return err
	}

	return nil
}

func (ms *MetricsService) GetCounter(ctx context.Context, id string) (*int64, error) {
	return ms.storage.GetCounter(ctx, id)
}

func (ms *MetricsService) GetGauge(ctx context.Context, id string) (*float64, error) {
	return ms.storage.GetGauge(ctx, id)
}

func (ms *MetricsService) GetMetric(ctx context.Context, id, mtype string) (*model.Metrics, error) {
	metric, err := ms.storage.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if metric.MType != mtype {
		return nil, model.ErrIncorrectAccess
	}
	return metric, nil
}

func (ms *MetricsService) BatchSetMetrics(ctx context.Context, metrics []*model.Metrics) error {
	return ms.storage.BatchUpdate(ctx, metrics)
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
