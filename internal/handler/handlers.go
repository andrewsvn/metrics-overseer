package handler

import (
	"errors"
	"github.com/andrewsvn/metrics-overseer/internal/handler/middleware"
	"log"
	"net/http"
	"strconv"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/go-chi/chi/v5"
)

type MetricsHandlers struct {
	msrv *service.MetricsService
	lg   *middleware.Loggable
}

func NewMetricsHandlers(ms *service.MetricsService) *MetricsHandlers {
	lg, err := middleware.NewLoggable("INFO", "metrics-handler")
	if err != nil {
		panic(err)
	}

	return &MetricsHandlers{
		msrv: ms,
		lg:   lg,
	}
}

func (mh *MetricsHandlers) GetRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(mh.lg.Middleware)
	r.Post("/update/{mtype}/{id}/{value}", mh.UpdateHandler())
	r.Get("/value/{mtype}/{id}", mh.GetValueHandler())
	r.Get("/", mh.ShowMetricsPage())

	return r
}

func (mh *MetricsHandlers) UpdateHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		svalue := chi.URLParam(r, "value")
		log.Printf("Updating metric of type %s: id=%s, value=%s", mtype, id, svalue)

		switch mtype {
		case model.Counter:
			mh.processUpdateCounterValue(rw, id, svalue)
		case model.Gauge:
			mh.processUpdateGaugeValue(rw, id, svalue)
		default:
			http.Error(rw, "unsupported metric type", http.StatusBadRequest)
		}
	}
}

func (mh *MetricsHandlers) GetValueHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		mtype := chi.URLParam(r, "mtype")
		id := chi.URLParam(r, "id")
		log.Printf("Fetching metric value of type %s: id=%s", mtype, id)

		switch mtype {
		case model.Counter:
			mh.processGetCounterValue(rw, id)
		case model.Gauge:
			mh.processGetGaugeValue(rw, id)
		default:
			http.Error(rw, "unsupported metric type", http.StatusBadRequest)
		}
	}
}

func (mh *MetricsHandlers) ShowMetricsPage() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		err := mh.msrv.GenerateAllMetricsHTML(rw)
		if err != nil {
			log.Printf("[ERROR] unable to render metrics page: %v", err)
			http.Error(rw, "unable to render metrics page", http.StatusInternalServerError)
			return
		}

		rw.Header().Add("Content-Type", "text/html")
	}
}

func (mh *MetricsHandlers) processUpdateCounterValue(rw http.ResponseWriter, id string, svalue string) {
	inc, err := strconv.ParseInt(svalue, 10, 64)
	if err != nil {
		http.Error(rw, "invalid metric value", http.StatusBadRequest)
		return
	}
	err = mh.msrv.AccumulateCounter(id, int64(inc))
	if err != nil {
		if errors.Is(err, model.ErrIncorrectAccess) {
			http.Error(rw, "wrong metric type", http.StatusBadRequest)
			return
		}
		log.Printf("[ERROR] unable to update counter value: %v", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (mh *MetricsHandlers) processUpdateGaugeValue(rw http.ResponseWriter, id string, svalue string) {
	value, err := strconv.ParseFloat(svalue, 64)
	if err != nil {
		http.Error(rw, "invalid metric value", http.StatusBadRequest)
		return
	}
	err = mh.msrv.SetGauge(id, value)
	if err != nil {
		if errors.Is(err, model.ErrIncorrectAccess) {
			http.Error(rw, "wrong metric type", http.StatusBadRequest)
			return
		}
		log.Printf("[ERROR] unable to update gauge value: %v", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (mh *MetricsHandlers) processGetCounterValue(rw http.ResponseWriter, id string) {
	pval, err := mh.msrv.GetCounter(id)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) || errors.Is(err, model.ErrIncorrectAccess) {
			http.Error(rw, "metric not found", http.StatusNotFound)
			return
		}
		log.Printf("[ERROR] unable to get counter value: %v", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Add("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	if pval == nil {
		_, _ = rw.Write([]byte("nil"))
	} else {
		_, _ = rw.Write(strconv.AppendInt(make([]byte, 0), *pval, 10))
	}
}

func (mh *MetricsHandlers) processGetGaugeValue(rw http.ResponseWriter, id string) {
	pval, err := mh.msrv.GetGauge(id)
	if err != nil {
		if errors.Is(err, repository.ErrMetricNotFound) || errors.Is(err, model.ErrIncorrectAccess) {
			http.Error(rw, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		log.Printf("[ERROR] unable to get gauge value: %v", err)
		http.Error(rw, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	rw.Header().Add("Content-Type", "text/plain")
	rw.WriteHeader(http.StatusOK)
	if pval == nil {
		_, _ = rw.Write([]byte("nil"))
	} else {
		_, _ = rw.Write(strconv.AppendFloat(make([]byte, 0), *pval, 'f', -1, 64))
	}
}
