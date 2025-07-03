package service

import (
	"html/template"
	"io"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
)

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

func (ms *MetricsService) GenerateAllMetricsHtml(w io.Writer) error {
	if ms.allMetricsTmpl == nil {
		tmpl, err := template.ParseFiles("resources/html/metricspage.html")
		if err != nil {
			return err
		}
		ms.allMetricsTmpl = tmpl
	}

	metrics, err := ms.storage.GetAllSorted()
	if err != nil {
		return err
	}

	page := MetricsPage{
		Metrics: metrics,
	}
	return ms.allMetricsTmpl.Execute(w, page)
}
