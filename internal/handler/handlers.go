package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/compress"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/encrypt"
	"github.com/andrewsvn/metrics-overseer/internal/handler/errorhandling"
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
	msrv        *service.MetricsService
	decomp      *compress.Decompressor
	securityCfg *servercfg.SecurityConfig

	baseLogger *zap.Logger
	logger     *zap.SugaredLogger
}

const (
	logErrorWriteBody = "error writing response body"
	logErrorGenHTML   = "error generating metrics html"
)

func NewMetricsHandlers(
	ms *service.MetricsService,
	securityCfg *servercfg.SecurityConfig,
	logger *zap.Logger,
) *MetricsHandlers {
	mhLogger := logger.Sugar().With(zap.String("component", "metrics-handlers"))
	return &MetricsHandlers{
		msrv:        ms,
		decomp:      compress.NewDecompressor(logger, compress.NewGzipReadEngine()),
		baseLogger:  logger,
		logger:      mhLogger,
		securityCfg: securityCfg,
	}
}

func (mh *MetricsHandlers) GetRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(
		middleware.NewHTTPLogging(mh.baseLogger).Middleware,
		middleware.NewAuthorization(mh.baseLogger, mh.securityCfg.SecretKey).Middleware,
		middleware.NewCompressing(mh.baseLogger).Middleware,
	)

	r.Post("/update/{mtype}/{id}/{value}", mh.updateByPathHandler())
	r.Route("/update", func(r chi.Router) {
		r.Post("/", mh.updateByBodyHandler())
	})
	r.Route("/updates", func(r chi.Router) {
		r.Post("/", mh.updateBatchHandler())
	})
	r.Route("/value", func(r chi.Router) {
		r.Post("/", mh.getJSONValueHandler())
	})
	r.Route("/ping", func(r chi.Router) {
		r.Get("/", mh.pingStorageHandler())
	})
	r.Get("/value/{mtype}/{id}", mh.getPlainValueHandler())
	r.Get("/", mh.showMetricsPage())

	return r
}

func (mh *MetricsHandlers) showMetricsPage() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		pageWriter := new(bytes.Buffer)
		err := mh.msrv.GenerateAllMetricsHTML(r.Context(), pageWriter)
		if err != nil {
			mh.logger.Error(logErrorGenHTML, zap.Error(err))
			http.Error(rw, "unable to render metrics page", http.StatusInternalServerError)
			return
		}

		payload := pageWriter.Bytes()

		encrypt.AddSignature([]byte(mh.securityCfg.SecretKey), payload, rw.Header())
		rw.Header().Add("Content-Type", "text/html")
		rw.WriteHeader(http.StatusOK)
		_, err = rw.Write(payload)
		if err != nil {
			mh.logger.Error(logErrorWriteBody, zap.Error(err))
		}
	}
}

func (mh *MetricsHandlers) updateByPathHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		svalue := chi.URLParam(r, "value")
		mh.logger.Debugw("Trying to update metric",
			"mtype", mtype,
			"id", id,
			"value", svalue,
		)

		metric, he := mh.buildMetric(id, mtype, svalue)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}
		he = mh.processUpdateMetric(r.Context(), metric)
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
			errorhandling.NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		metric := &model.Metrics{}
		if err := json.Unmarshal(body, &metric); err != nil {
			errorhandling.NewValidationHandlerError(fmt.Sprintf("error unmarshalling body: %v", err)).Render(rw)
			return
		}

		mh.logger.Debugw("Trying to update metric",
			"metric", metric,
		)
		he := mh.validateMetric(metric)
		if he != nil {
			if he.Error != nil {
				mh.logger.Error(he.Message, zap.Error(he.Error))
			}
			he.Render(rw)
			return
		}
		he = mh.processUpdateMetric(r.Context(), metric)
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

func (mh *MetricsHandlers) updateBatchHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		body, err := mh.decomp.ReadRequestBody(r)
		if err != nil {
			errorhandling.NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		metrics := make([]*model.Metrics, 0)
		if err := json.Unmarshal(body, &metrics); err != nil {
			errorhandling.NewValidationHandlerError(fmt.Sprintf("error unmarshalling body: %v", err)).Render(rw)
			return
		}

		mh.logger.Debugw("Trying to update metrics",
			"count", len(metrics),
		)
		err = mh.msrv.BatchSetMetrics(r.Context(), metrics)
		if err != nil {
			if errors.Is(err, model.ErrIncorrectAccess) {
				errorhandling.NewValidationHandlerError(err.Error()).Render(rw)
				return
			}
		}
		rw.WriteHeader(http.StatusOK)
	}
}

func (mh *MetricsHandlers) getPlainValueHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		mh.logger.Debug("Fetching metric",
			zap.String("mtype", mtype),
			zap.String("id", id),
		)

		metric, he := mh.getMetric(r.Context(), id, mtype)
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
			errorhandling.NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		metric := &model.Metrics{}
		if err := json.Unmarshal(body, &metric); err != nil {
			errorhandling.NewValidationHandlerError(fmt.Sprintf("error decoding body: %v", err)).Render(rw)
			return
		}

		mh.logger.Debugw("Fetching metric",
			"id", metric.ID,
			"mtype", metric.MType,
		)
		metric, he := mh.getMetric(r.Context(), metric.ID, metric.MType)
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

func (mh *MetricsHandlers) pingStorageHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		err := mh.msrv.PingStorage(r.Context())
		if err != nil {
			mh.logger.Error("failed to ping storage", zap.Error(err))
			http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		rw.WriteHeader(http.StatusOK)
	}
}

func (mh *MetricsHandlers) processUpdateMetric(ctx context.Context, metric *model.Metrics) *errorhandling.Error {
	var err error

	switch metric.MType {
	case model.Counter:
		err = mh.msrv.AccumulateCounter(ctx, metric.ID, *metric.Delta)
	case model.Gauge:
		err = mh.msrv.SetGauge(ctx, metric.ID, *metric.Value)
	default:
		return errorhandling.NewValidationHandlerError("unsupported metric type: " + metric.MType)
	}

	if err != nil {
		if errors.Is(err, repository.ErrStore) {
			// no impact on main flow, only log this
			mh.logger.Error("metrics store error", zap.Error(err))
		}
		if errors.Is(err, model.ErrIncorrectAccess) {
			return errorhandling.NewValidationHandlerError("wrong metric type")
		}
		return errorhandling.NewInternalServerError(fmt.Errorf("error updating metric: %w", err))
	}
	return nil
}

func (mh *MetricsHandlers) buildMetric(id, mtype, svalue string) (*model.Metrics, *errorhandling.Error) {
	var delta *int64
	var value *float64

	switch mtype {
	case model.Counter:
		dval, err := strconv.ParseInt(svalue, 10, 64)
		if err != nil {
			return nil, errorhandling.NewValidationHandlerError("invalid metric value: " + svalue)
		}
		delta = &dval
	case model.Gauge:
		fval, err := strconv.ParseFloat(svalue, 64)
		if err != nil {
			return nil, errorhandling.NewValidationHandlerError("invalid metric value: " + svalue)
		}
		value = &fval
	default:
		return nil, errorhandling.NewValidationHandlerError("unsupported metric type: " + mtype)
	}

	return model.NewMetrics(id, mtype, delta, value), nil
}

func (mh *MetricsHandlers) validateMetric(metric *model.Metrics) *errorhandling.Error {
	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			return errorhandling.NewValidationHandlerError("missing counter metric value")
		}
	case model.Gauge:
		if metric.Value == nil {
			return errorhandling.NewValidationHandlerError("missing gauge metric value")
		}
	default:
		return errorhandling.NewValidationHandlerError("unsupported metric type: " + metric.MType)
	}
	return nil
}

func (mh *MetricsHandlers) getMetric(
	ctx context.Context,
	id, mtype string,
) (*model.Metrics, *errorhandling.Error) {

	if len(strings.TrimSpace(id)) == 0 {
		return nil, errorhandling.NewValidationHandlerError("missing metric id")
	}
	if mtype != model.Counter && mtype != model.Gauge {
		return nil, errorhandling.NewValidationHandlerError("unsupported metric type: " + mtype)
	}

	metric, err := mh.msrv.GetMetric(ctx, id, mtype)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) || errors.Is(err, model.ErrIncorrectAccess) {
			return nil, errorhandling.NewNotFoundHandlerError("metric not found")
		}
		return nil, errorhandling.NewInternalServerError(fmt.Errorf("error getting metric: %w", err))
	}
	return metric, nil
}

func (mh *MetricsHandlers) renderMetricValue(rw http.ResponseWriter, metric *model.Metrics) {
	var payload []byte
	switch metric.MType {
	case model.Counter:
		if metric.Delta == nil {
			payload = []byte("nil")
		} else {
			payload = strconv.AppendInt(make([]byte, 0), *metric.Delta, 10)
		}
	case model.Gauge:
		if metric.Value == nil {
			payload = []byte("nil")
		} else {
			payload = strconv.AppendFloat(make([]byte, 0), *metric.Value, 'f', -1, 64)
		}
	}

	encrypt.AddSignature([]byte(mh.securityCfg.SecretKey), payload, rw.Header())
	rw.Header().Add("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	_, err := rw.Write(payload)
	if err != nil {
		mh.logger.Error(logErrorWriteBody, zap.Error(err))
	}
}

func (mh *MetricsHandlers) renderMetricJSON(rw http.ResponseWriter, metric *model.Metrics) {
	payload, err := json.MarshalIndent(metric, "", "  ")
	if err != nil {
		mh.logger.Error(logErrorWriteBody, zap.Error(err))
		rw.WriteHeader(http.StatusInternalServerError)
		return
	}

	encrypt.AddSignature([]byte(mh.securityCfg.SecretKey), payload, rw.Header())
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, err = rw.Write(payload)
	if err != nil {
		mh.logger.Error(logErrorWriteBody, zap.Error(err))
	}
}
