package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/service"
)

type MetricsHandlers struct {
	msrv *service.MetricsService
}

func NewMetricsHandlers(ms *service.MetricsService) *MetricsHandlers {
	return &MetricsHandlers{
		msrv: ms,
	}
}

func (mh *MetricsHandlers) UpdateHandler() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		log.Printf("Update request received: method=%s, url=%s", r.Method, r.URL)

		if r.Method != http.MethodPost {
			http.Error(rw, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		params := strings.Split(strings.TrimPrefix(r.URL.String(), "/update/"), "/")
		if len(params) < 3 {
			http.Error(rw, "metric name and/or value not specified", http.StatusNotFound)
			return
		}

		mtype := params[0]
		id := params[1]
		svalue := params[2]
		log.Printf("Update metrics data: type=%s, id=%s, value=%s", mtype, id, svalue)

		switch mtype {
		case model.Counter:
			mh.processCounterValue(rw, id, svalue)
		case model.Gauge:
			mh.processGaugeValue(rw, id, svalue)
		default:
			http.Error(rw, "unsupported metric type", http.StatusBadRequest)
		}
	}
}

func (mh *MetricsHandlers) processCounterValue(
	rw http.ResponseWriter, id string, svalue string) {

	inc, err := strconv.Atoi(svalue)
	if err != nil {
		http.Error(rw, "invalid metric value", http.StatusBadRequest)
		return
	}
	err = mh.msrv.AccumulateCounter(id, int64(inc))
	if err != nil {
		if errors.Is(err, model.ErrMethodNotSupported) {
			http.Error(rw, "wrong metric type", http.StatusBadRequest)
			return
		}
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}

func (mh *MetricsHandlers) processGaugeValue(
	rw http.ResponseWriter, id string, svalue string) {

	value, err := strconv.ParseFloat(svalue, 64)
	if err != nil {
		http.Error(rw, "invalid metric value", http.StatusBadRequest)
		return
	}
	err = mh.msrv.SetGauge(id, value)
	if err != nil {
		if errors.Is(err, model.ErrMethodNotSupported) {
			http.Error(rw, "wrong metric type", http.StatusBadRequest)
			return
		}
		http.Error(rw, "internal error", http.StatusInternalServerError)
		return
	}

	rw.WriteHeader(http.StatusOK)
}
