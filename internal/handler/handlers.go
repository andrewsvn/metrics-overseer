package handler

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/encrypt"
	"github.com/andrewsvn/metrics-overseer/internal/handler/errorhandling"
	"github.com/andrewsvn/metrics-overseer/internal/handler/middleware"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// @Title Metrics Overseer API
// @Description Service for collecting and providing performance metrics from other services
// @Version 1.0

// @BasePath /
// @Host metrics-overseer:8080

// @SecurityDefinitions.apikey SecretKeyAuth
// @In Header
// @Name signAuth

// @Tag.name Maintenance
// @Tag.description endpoints group for controlling and providing inner service state

// @Tag.name Metrics
// @Tag.description endpoints group for working with metrics

// @Tag.name UI
// @Tag.description endpoints group for rendering HTML pages

type MetricsHandlers struct {
	msrv          *service.MetricsService
	securityCfg   *servercfg.SecurityConfig
	trustedSubnet *net.IPNet
	decrypter     encrypt.Decrypter

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
) (*MetricsHandlers, error) {
	mhLogger := logger.Sugar().With(zap.String("component", "metrics-handlers"))

	var err error

	var privKey *rsa.PrivateKey
	if securityCfg.PrivateKeyPath != "" {
		privKey, err = encrypt.ReadRSAPrivateKeyFromFile(securityCfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("error reading private key for decryption: %w", err)
		}
		mhLogger.Infow("using RSA private key for request decryption")
	}

	var trustedSubnet *net.IPNet
	if securityCfg.TrustedSubnet != "" {
		_, trustedSubnet, err = net.ParseCIDR(securityCfg.TrustedSubnet)
		if err != nil {
			return nil, fmt.Errorf("error parsing trusted subnet: %w", err)
		}
		mhLogger.Infow("using trusted subnet for request decryption: %s", securityCfg.TrustedSubnet)
	}

	return &MetricsHandlers{
		msrv:          ms,
		decrypter:     encrypt.NewRSAEngineBuilder().PrivateKey(privKey).Build(),
		baseLogger:    logger,
		logger:        mhLogger,
		securityCfg:   securityCfg,
		trustedSubnet: trustedSubnet,
	}, nil
}

func (mh *MetricsHandlers) GetRouter() *chi.Mux {
	r := chi.NewRouter()

	// For secured requests middlewares applied in a given order - so a client which applies multiple transformations
	// and checks to their request must apply them in the corresponding order (only for the body part):
	// - encrypt request body with an RSA public key if it is specified
	// - compress the body if needed
	// - sign the body if secret key is available
	secureR := r.With(
		middleware.NewHTTPLogging(mh.baseLogger).Middleware,
		middleware.NewAuthorization(mh.baseLogger, mh.securityCfg.SecretKey, mh.trustedSubnet).Middleware,
		middleware.NewCompressing(mh.baseLogger).Middleware,
		middleware.NewDecryption(mh.baseLogger, mh.decrypter).Middleware,
	)

	// For non-secured requests (status, metrics reading) sign verification and authentication are disabled
	plainR := r.With(
		middleware.NewHTTPLogging(mh.baseLogger).Middleware,
		middleware.NewCompressing(mh.baseLogger).Middleware,
	)

	// secure routes
	secureR.Post("/update/{mtype}/{id}/{value}", mh.updateByPathHandler())
	secureR.Route("/update", func(r chi.Router) {
		r.Post("/", mh.updateByBodyHandler())
	})
	secureR.Route("/updates", func(r chi.Router) {
		r.Post("/", mh.updateBatchHandler())
	})

	// unsecure routes
	plainR.Route("/value", func(r chi.Router) {
		r.Post("/", mh.getJSONValueHandler())
	})
	plainR.Get("/value/{mtype}/{id}", mh.getPlainValueHandler())

	// UI
	plainR.Get("/", mh.showMetricsPageHandler())

	// ping storage
	plainR.Route("/ping", func(r chi.Router) {
		r.Get("/", mh.pingStorageHandler())
	})

	return r
}

// @Tags UI
// @Summary Render overall collected metrics page
// @Description Renders metrics page containing table with all collected metrics and their values - sorted alphabetically
// @ID uiMetricsPage
// @Produce html
// @Success 200 {object} service.MetricsPage
// @Failure 500 {string} string "Internal server error"
// @Security SecretKeyAuth
// @Router / [get]
func (mh *MetricsHandlers) showMetricsPageHandler() http.HandlerFunc {
	return mh.showMetricsPage
}

// showMetricsPage writes into response HTML page consisting of a table with all collected metrics
// with corresponding types and values.
// in case page can't be rendered, HTTP code 500 is set to response.
func (mh *MetricsHandlers) showMetricsPage(rw http.ResponseWriter, r *http.Request) {
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

// @Tags Metrics
// @Summary Accumulate single metric value provided by path parameters
// @Description Accumulate metric value for ID and metric type provided in parameters
// @ID updateMetricByPath
// @Param mtype path string true "Metric Type" Enums(Counter, Gauge)
// @Param id path string true "Metric ID"
// @Param value path string true "Metric Value"
// @Success 200
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Security SecretKeyAuth
// @Router /update/{mtype}/{id}/{value} [post]
func (mh *MetricsHandlers) updateByPathHandler() http.HandlerFunc {
	return mh.updateByPath
}

// updateByPath takes mtype, id and value parameters from request path and tries to update corresponding metric
// in storage (or create new if it doesn't exist).
// in successful case HTTP code 200 is written into response
// in case provided metric data is invaild or any of input fields are not provided, HTTP code 400 is written into response
// in any other case error is considered unprocessable and HTTP code 500 is written
func (mh *MetricsHandlers) updateByPath(rw http.ResponseWriter, r *http.Request) {
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

	he = mh.processUpdateMetric(r.Context(), metric, mh.extractRemoteIPAddress(r))
	if he != nil {
		if he.Error != nil {
			mh.logger.Error(he.Message, zap.Error(he.Error))
		}
		he.Render(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// @Tags Metrics
// @Summary Accumulate single metric value provided in JSON body
// @Description Accumulate metric value for id and metric type provided in body
// @ID updateMetricByBody
// @Accept json
// @Body {object} model.Metrics
// @Success 200
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Security SecretKeyAuth
// @Router /update [post]
func (mh *MetricsHandlers) updateByBodyHandler() http.HandlerFunc {
	return mh.updateByBody
}

// updateByPath reads inbound request body, unmarshals it to model.Metrics and tries to update corresponding metric
// in storage (or create new if it doesn't exist).
// Valid input body must have "id" and "mtype" fields filled (mtype can be either "counter" or "gauge") and
// either "delta" set to some integer value (for counter-type metric)
// or "value" set to some decimal value (for gauge-type metric).
// Also, metric considered as not valid if it is already stored in a storage with a different mtype.
// in successful case HTTP code 200 is written into response
// in case body JSON can't be unmarshalled or required data for update is missing, HTTP code 400 is written into response
// in any other case error is considered unprocessable and HTTP code 500 is written
func (mh *MetricsHandlers) updateByBody(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorhandling.NewValidationHandlerError(fmt.Sprintf("error reading request body: %v", err)).Render(rw)
		return
	}

	metric := &model.Metrics{}
	if err := json.Unmarshal(body, &metric); err != nil {
		errorhandling.NewValidationHandlerError(fmt.Sprintf("error unmarshalling request body: %v", err)).Render(rw)
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
	he = mh.processUpdateMetric(r.Context(), metric, mh.extractRemoteIPAddress(r))
	if he != nil {
		if he.Error != nil {
			mh.logger.Error(he.Message, zap.Error(he.Error))
		}
		he.Render(rw)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

// @Tags Metrics
// @Summary Accumulate multiple metric value provided in JSON array
// @Description Accumulate batch of metric values for corresponding ids and metric types provided in body as array
// @ID updateMetricBatch
// @Accept json
// @Body {array} model.Metrics
// @Success 200
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Security SecretKeyAuth
// @Router /updates [post]
func (mh *MetricsHandlers) updateBatchHandler() http.HandlerFunc {
	return mh.updateBatch
}

// updateBatch reads inbound request body, unmarshals it to array of model.Metrics and tries to update all provided
// metrics in storage (or create new for those not existing).
// Input body must contain only valid model.Metrics input objects (see updateByBody description for validation explanation).
// in successful case HTTP code 200 is written into response
// in case body JSON can't be unmarshalled or any metric is invalid, HTTP code 400 is written into response
// in any other case error is considered unprocessable and HTTP code 500 is written
func (mh *MetricsHandlers) updateBatch(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorhandling.NewValidationHandlerError(fmt.Sprintf("error reading request body: %v", err)).Render(rw)
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
	err = mh.msrv.BatchAccumulateMetrics(r.Context(), metrics, mh.extractRemoteIPAddress(r))
	if err != nil {
		if errors.Is(err, repository.ErrIncorrectAccess) {
			errorhandling.NewValidationHandlerError(err.Error()).Render(rw)
			return
		}
	}
	rw.WriteHeader(http.StatusOK)
}

// @Tags Metrics
// @Summary Return metric value by ID and type
// @Description Return metric value for ID and metric type provided in path parameters if it exists
// @ID getMetricByPath
// @Produce plain
// @Param mtype path string true "Metric Type" Enums(Counter, Gauge)
// @Param id path string true "Metric ID"
// @Success 200 {string} metric value
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Metric not found"
// @Failure 500 {string} string "Internal server error"
// @Security SecretKeyAuth
// @Router /value/{mtype}/{id} [get]
func (mh *MetricsHandlers) getPlainValueHandler() http.HandlerFunc {
	return mh.getPlainValue
}

// getPlainValue gets metric ID and type from request path parameters and tries to fetch existing metric from
// the server storage. If metric with given parameters exists, its value is written into response body as plain string.
// in success case, HTTP code 200 is written into response
// in case metric doesn't exist in the storage, HTTP code 404 is written into response
// in any other case error is considered unprocessable and HTTP code 500 is written
func (mh *MetricsHandlers) getPlainValue(rw http.ResponseWriter, r *http.Request) {
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

// @Tags Metrics
// @Summary Return metric value by JSON parameters
// @Description Return metric value for ID and metric type provided in JSON body if it exists
// @ID getMetricByBody
// @Produce json
// @Accept json
// @Body {object} model.Metrics
// @Success 200 {string} metric value
// @Failure 400 {string} string "Bad request"
// @Failure 401 {string} string "Unauthorized"
// @Failure 404 {string} string "Metric not found"
// @Failure 500 {string} string "Internal server error"
// @Security SecretKeyAuth
// @Router /value [post]
func (mh *MetricsHandlers) getJSONValueHandler() http.HandlerFunc {
	return mh.getJSONValue
}

// getJSONValue reads inbound request body, unmarshals it to model.Metrics and gets metric ID and type,
// then tries to fetch existing metric from the server storage.
// If metric with given parameters exists, its value is written into response body as plain string.
// in success case, HTTP code 200 is written into response
// in case body JSON can't be unmarshalled or required parameters ("id", "mtype") not present,
// HTTP code 400 is written into response
// in case metric doesn't exist in the storage, HTTP code 404 is written into response
// in any other case error is considered unprocessable and HTTP code 500 is written
func (mh *MetricsHandlers) getJSONValue(rw http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		errorhandling.NewValidationHandlerError(fmt.Sprintf("error reading request body: %v", err)).Render(rw)
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

// @Tags Maintenance
// @Summary Check storage for availability
// @Description Returns success response in case underlying storage database connection is working.
//
//	For memory-based storage types always returns success.
//
// @ID pingStorage
// @Success 200 {string} metric value
// @Failure 401 {string} string "Unauthorized"
// @Failure 500 {string} string "Internal server error"
// @Security SecretKeyAuth
// @Router /ping [get]
func (mh *MetricsHandlers) pingStorageHandler() http.HandlerFunc {
	return mh.pingStorage
}

// pingStorage is a simple method to check if metrics storage is accessible at the moment.
// For file-based or memory-based storage this check is always successful.
// For database storage actual ping of DB connection is performed.
// in case of success, HTTP code 200 is written into response
// in case of failure, HTTP code 500 is written
func (mh *MetricsHandlers) pingStorage(rw http.ResponseWriter, r *http.Request) {
	err := mh.msrv.PingStorage(r.Context())
	if err != nil {
		mh.logger.Error("failed to ping storage", zap.Error(err))
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	rw.WriteHeader(http.StatusOK)
}

func (mh *MetricsHandlers) processUpdateMetric(
	ctx context.Context,
	metric *model.Metrics,
	ipAddr string,
) *errorhandling.Error {
	err := mh.msrv.AccumulateMetric(ctx, metric, ipAddr)
	if err != nil {
		if errors.Is(err, service.ErrUnsupportedMetricType) {
			return errorhandling.NewValidationHandlerError(err.Error())
		}
		if errors.Is(err, repository.ErrStore) {
			// no impact on main flow, only log this
			mh.logger.Error("metrics store error", zap.Error(err))
		}
		if errors.Is(err, repository.ErrIncorrectAccess) {
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
		if errors.Is(err, repository.ErrMetricNotFound) || errors.Is(err, repository.ErrIncorrectAccess) {
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

func (mh *MetricsHandlers) extractRemoteIPAddress(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		mh.logger.Warnw("error extracting remote IP address", zap.Error(err))
		return "N/A"
	}
	return ip
}
