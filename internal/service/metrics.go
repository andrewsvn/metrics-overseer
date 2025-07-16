package service

import (
	_ "embed"
	"fmt"
	"html/template"
	"io"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
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

func (ms *MetricsService) AccumulateCounter(id string, inc int64) error {
	return ms.storage.AddCounter(id, inc)
}

func (ms *MetricsService) SetGauge(id string, val float64) error {
	return ms.storage.SetGauge(id, val)
}

func (ms *MetricsService) GetCounter(id string) (*int64, error) {
	return ms.storage.GetCounter(id)
}

func (ms *MetricsService) GetGauge(id string) (*float64, error) {
	return ms.storage.GetGauge(id)
}

func (ms *MetricsService) GetMetric(id, mtype string) (*model.Metrics, error) {
	metric, err := ms.storage.GetByID(id)
	if err != nil {
		return nil, err
	}
	if metric.MType != mtype {
		return nil, model.ErrIncorrectAccess
	}
	return metric, nil
}

func (ms *MetricsService) GenerateAllMetricsHTML(w io.Writer) error {
	if ms.allMetricsTmpl == nil {
		tmpl := template.New("metricspage")
		tmpl, err := tmpl.Parse(metricspage)
		if err != nil {
			return fmt.Errorf("error parsing page template: %w", err)
		}
		ms.allMetricsTmpl = tmpl
	}

	metrics, err := ms.storage.GetAllSorted()
	if err != nil {
		return fmt.Errorf("can't get all metrics from storage: %w", err)
	}

	page := MetricsPage{
		Metrics: metrics,
	}
	return ms.allMetricsTmpl.Execute(w, page)
}
