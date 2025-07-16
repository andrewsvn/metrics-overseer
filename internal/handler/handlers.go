package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/handler/middleware"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"net/http"
	"strconv"
	"strings"
)

type MetricsHandlers struct {
	msrv   *service.MetricsService
	logger *zap.Logger
}

const (
	logErrorWriteBody    = "Error writing response body"
	logErrorCloseBody    = "Error closing request body"
	logErrorDecodeBody   = "Error decoding request body"
	logErrorGenHTML      = "Error generating metrics html"
	logErrorUpdateMetric = "Error updating metric"
	logErrorGetMetric    = "Error getting metric"
)

func NewMetricsHandlers(ms *service.MetricsService, logger *zap.Logger) *MetricsHandlers {
	return &MetricsHandlers{
		msrv:   ms,
		logger: logger,
	}
}

func (mh *MetricsHandlers) GetRouter() *chi.Mux {
	r := chi.NewRouter()

	lg := middleware.NewLoggable(mh.logger)
	r.Use(lg.Middleware)

	r.Post("/update/{mtype}/{id}/{value}", mh.UpdateByPathHandler())
	r.Post("/update", mh.UpdateByBodyHandler())
	r.Post("/value", mh.GetJSONValueHandler())
	r.Get("/value/{mtype}/{id}", mh.GetPlainValueHandler())
	r.Get("/", mh.ShowMetricsPage())

	return r
}

func (mh *MetricsHandlers) ShowMetricsPage() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Add("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)

		err := mh.msrv.GenerateAllMetricsHTML(rw)
		if err != nil {
			mh.logger.Error(logErrorGenHTML, zap.Error(err))
			http.Error(rw, "unable to render metrics page", http.StatusInternalServerError)
			return
		}
	}
}

func (mh *MetricsHandlers) UpdateByPathHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		svalue := chi.URLParam(r, "value")
		mh.logger.Info("Trying to update metric",
			zap.String("mtype", mtype), zap.String("id", id), zap.String("value", svalue))

		metric, he := mh.buildMetric(id, mtype, svalue)
		if he != nil {
			he.Render(rw)
			return
		}
		he = mh.processUpdateMetric(metric)
		if he != nil {
			he.Render(rw)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func (mh *MetricsHandlers) UpdateByBodyHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		defer func() {
			err := r.Body.Close()
			if err != nil {
				mh.logger.Error(logErrorCloseBody, zap.Error(err))
			}
		}()

		metric := &model.Metrics{}
		if err := json.NewDecoder(r.Body).Decode(metric); err != nil {
			mh.logger.Error(logErrorDecodeBody, zap.Error(err))
			NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}
		he := mh.validateMetric(metric)
		if he != nil {
			he.Render(rw)
			return
		}
		he = mh.processUpdateMetric(metric)
		if he != nil {
			he.Render(rw)
			return
		}

		rw.WriteHeader(http.StatusOK)

	}
}

func (mh *MetricsHandlers) GetPlainValueHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		mh.logger.Info("Fetching metric", zap.String("mtype", mtype), zap.String("id", id))

		metric, he := mh.getMetric(id, mtype)
		if he != nil {
			he.Render(rw)
			return
		}
		mh.renderMetricValue(rw, metric)
	}
}

func (mh *MetricsHandlers) GetJSONValueHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		metric := &model.Metrics{}
		if err := json.NewDecoder(r.Body).Decode(metric); err != nil {
			mh.logger.Error(logErrorDecodeBody, zap.Error(err))
			NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		metric, he := mh.getMetric(metric.ID, metric.MType)
		if he != nil {
			he.Render(rw)
			return
		}
		mh.renderMetricJSON(rw, metric)
	}
}

func (mh *MetricsHandlers) processUpdateMetric(metric *model.Metrics) *HandlerError {
	var err error

	switch metric.MType {
	case model.Counter:
		err = mh.msrv.AccumulateCounter(metric.ID, *metric.Delta)
	case model.Gauge:
		err = mh.msrv.SetGauge(metric.ID, *metric.Value)
	default:
		return NewValidationHandlerError("unsupported metric type")
	}

	if err != nil {
		if errors.Is(err, model.ErrIncorrectAccess) {
			return NewValidationHandlerError("wrong metric type")
		}
		mh.logger.Error(logErrorUpdateMetric, zap.Error(err))
		return InternalError
	}
	return nil
}

func (mh *MetricsHandlers) buildMetric(id, mtype, svalue string) (*model.Metrics, *HandlerError) {
	var delta *int64
	var value *float64

	switch mtype {
	case model.Counter:
		dval, err := strconv.ParseInt(svalue, 10, 64)
		if err != nil {
			return nil, NewValidationHandlerError("invalid metric value")
		}
		delta = &dval
	case model.Gauge:
		fval, err := strconv.ParseFloat(svalue, 64)
		if err != nil {
			return nil, NewValidationHandlerError("invalid metric value")
		}
		value = &fval
	default:
		return nil, NewValidationHandlerError("unsupported metric type")
	}

	return model.NewMetrics(id, mtype, delta, value), nil
}

func (mh *MetricsHandlers) validateMetric(metric *model.Metrics) *HandlerError {
	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			return NewValidationHandlerError("missing counter metric value")
		}
	case model.Gauge:
		if metric.Value == nil {
			return NewValidationHandlerError("missing gauge metric value")
		}
	default:
		return NewValidationHandlerError("unsupported metric type")
	}
	return nil
}

func (mh *MetricsHandlers) getMetric(id, mtype string) (*model.Metrics, *HandlerError) {
	if len(strings.TrimSpace(id)) == 0 {
		return nil, NewValidationHandlerError("missing metric id")
	}
	if mtype != model.Counter && mtype != model.Gauge {
		return nil, NewValidationHandlerError("unsupported metric type")
	}

	metric, err := mh.msrv.GetMetric(id, mtype)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) || errors.Is(err, model.ErrIncorrectAccess) {
			return nil, NewNotFoundHandlerError("metric not found")
		}
		mh.logger.Error(logErrorGetMetric, zap.Error(err))
		return nil, InternalError
	}
	return metric, nil
}

func (mh *MetricsHandlers) renderMetricValue(rw http.ResponseWriter, metric *model.Metrics) {
	rw.Header().Add("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)

	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			_, err := rw.Write([]byte("nil"))
			if err != nil {
				mh.logger.Error(logErrorWriteBody, zap.Error(err))
			}
		} else {
			_, err := rw.Write(strconv.AppendInt(make([]byte, 0), *metric.Delta, 10))
			if err != nil {
				mh.logger.Error(logErrorWriteBody, zap.Error(err))
			}
		}
	case model.Gauge:
		if metric.Value == nil {
			_, err := rw.Write([]byte("nil"))
			if err != nil {
				mh.logger.Error(logErrorWriteBody, zap.Error(err))
			}
		} else {
			_, err := rw.Write(strconv.AppendFloat(make([]byte, 0), *metric.Value, 'f', -1, 64))
			if err != nil {
				mh.logger.Error(logErrorWriteBody, zap.Error(err))
			}
		}
	}
}

func (mh *MetricsHandlers) renderMetricJSON(rw http.ResponseWriter, metric *model.Metrics) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	bytes, err := json.MarshalIndent(metric, "", "  ")
	if err != nil {
		mh.logger.Error(logErrorWriteBody, zap.Error(err))
	}
	_, err = rw.Write(bytes)
	if err != nil {
		mh.logger.Error(logErrorWriteBody, zap.Error(err))
	}
}
