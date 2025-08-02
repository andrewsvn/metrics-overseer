package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/compress"
	"github.com/andrewsvn/metrics-overseer/internal/db"
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
	dbconn db.Connection
	decomp *compress.Decompressor
	logger *zap.SugaredLogger
}

const (
	logErrorWriteBody = "error writing response body"
	logErrorGenHTML   = "error generating metrics html"
)

func NewMetricsHandlers(ms *service.MetricsService, dbconn db.Connection,
	logger *zap.Logger) *MetricsHandlers {
	mhLogger := logger.Sugar().With(zap.String("component", "metrics-handlers"))
	return &MetricsHandlers{
		msrv:   ms,
		dbconn: dbconn,
		decomp: compress.NewDecompressor(logger, compress.NewGzipReadEngine()),
		logger: mhLogger,
	}
}

func (mh *MetricsHandlers) GetRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(
		middleware.NewHTTPLogging(mh.logger.Desugar()).Middleware,
		middleware.NewCompressing(mh.logger.Desugar()).Middleware,
	)

	r.Post("/update/{mtype}/{id}/{value}", mh.updateByPathHandler())
	r.Route("/update", func(r chi.Router) {
		r.Post("/", mh.updateByBodyHandler())
	})
	r.Route("/value", func(r chi.Router) {
		r.Post("/", mh.getJSONValueHandler())
	})
	r.Route("/ping", func(r chi.Router) {
		r.Get("/", mh.pingDatabaseHandler())
	})
	r.Get("/value/{mtype}/{id}", mh.getPlainValueHandler())
	r.Get("/", mh.showMetricsPage())

	return r
}

func (mh *MetricsHandlers) showMetricsPage() http.HandlerFunc {
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

func (mh *MetricsHandlers) updateByPathHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		svalue := chi.URLParam(r, "value")
		mh.logger.Info("Trying to update metric",
			zap.String("mtype", mtype), zap.String("id", id), zap.String("value", svalue))

		metric, he := mh.buildMetric(id, mtype, svalue)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}
		he = mh.processUpdateMetric(metric)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func (mh *MetricsHandlers) updateByBodyHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		body, err := mh.decomp.ReadRequestBody(r)
		if err != nil {
			NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		metric := &model.Metrics{}
		if err := json.Unmarshal(body, &metric); err != nil {
			NewValidationHandlerError(fmt.Sprintf("error unmarshalling body: %v", err)).Render(rw)
			return
		}
		he := mh.validateMetric(metric)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}
		he = mh.processUpdateMetric(metric)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}

func (mh *MetricsHandlers) getPlainValueHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		mh.logger.Info("Fetching metric", zap.String("mtype", mtype), zap.String("id", id))

		metric, he := mh.getMetric(id, mtype)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}
		mh.renderMetricValue(rw, metric)
	}
}

func (mh *MetricsHandlers) getJSONValueHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		body, err := mh.decomp.ReadRequestBody(r)
		if err != nil {
			NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		metric := &model.Metrics{}
		if err := json.Unmarshal(body, &metric); err != nil {
			NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		metric, he := mh.getMetric(metric.ID, metric.MType)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}
		mh.renderMetricJSON(rw, metric)
	}
}

func (mh *MetricsHandlers) pingDatabaseHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		if mh.dbconn == nil {
			mh.logger.Error("database connection not set up - unable to ping")
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		err := mh.dbconn.Ping(r.Context())
		if err != nil {
			mh.logger.Error("failed to ping postgres database connection", zap.Error(err))
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	}
}

func (mh *MetricsHandlers) processUpdateMetric(metric *model.Metrics) *HandlingError {
	var err error

	switch metric.MType {
	case model.Counter:
		err = mh.msrv.AccumulateCounter(metric.ID, *metric.Delta)
	case model.Gauge:
		err = mh.msrv.SetGauge(metric.ID, *metric.Value)
	default:
		return NewValidationHandlerError("unsupported metric type: " + metric.MType)
	}

	if err != nil {
		if errors.Is(err, repository.ErrStore) {
			// no impact on main flow, only log this
			mh.logger.Error("metrics store error", zap.Error(err))
		}
		if errors.Is(err, model.ErrIncorrectAccess) {
			return NewValidationHandlerError("wrong metric type")
		}
		return NewInternalServerError(fmt.Errorf("error updating metric: %w", err))
	}
	return nil
}

func (mh *MetricsHandlers) buildMetric(id, mtype, svalue string) (*model.Metrics, *HandlingError) {
	var delta *int64
	var value *float64

	switch mtype {
	case model.Counter:
		dval, err := strconv.ParseInt(svalue, 10, 64)
		if err != nil {
			return nil, NewValidationHandlerError("invalid metric value: " + svalue)
		}
		delta = &dval
	case model.Gauge:
		fval, err := strconv.ParseFloat(svalue, 64)
		if err != nil {
			return nil, NewValidationHandlerError("invalid metric value: " + svalue)
		}
		value = &fval
	default:
		return nil, NewValidationHandlerError("unsupported metric type: " + mtype)
	}

	return model.NewMetrics(id, mtype, delta, value), nil
}

func (mh *MetricsHandlers) validateMetric(metric *model.Metrics) *HandlingError {
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
		return NewValidationHandlerError("unsupported metric type: " + metric.MType)
	}
	return nil
}

func (mh *MetricsHandlers) getMetric(id, mtype string) (*model.Metrics, *HandlingError) {
	if len(strings.TrimSpace(id)) == 0 {
		return nil, NewValidationHandlerError("missing metric id")
	}
	if mtype != model.Counter && mtype != model.Gauge {
		return nil, NewValidationHandlerError("unsupported metric type: " + mtype)
	}

	metric, err := mh.msrv.GetMetric(id, mtype)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) || errors.Is(err, model.ErrIncorrectAccess) {
			return nil, NewNotFoundHandlerError("metric not found")
		}
		return nil, NewInternalServerError(fmt.Errorf("error getting metric: %w", err))
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
