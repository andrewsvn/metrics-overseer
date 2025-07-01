package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateHandler(t *testing.T) {
	type want struct {
		code     int
		response string
	}
	tests := []struct {
		name   string
		method string
		url    string
		want   want
	}{
		{
			name:   "counter_metric",
			method: http.MethodPost,
			url:    "/update/counter/cnt1/10",
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:   "gauge_metric",
			method: http.MethodPost,
			url:    "/update/gauge/gauge1/1.05",
			want: want{
				code: http.StatusOK,
			},
		},
		{
			name:   "wrong_http_method",
			method: http.MethodPut,
			url:    "/update/counter/gauge1/1.05",
			want: want{
				code:     http.StatusMethodNotAllowed,
				response: "method not allowed",
			},
		},
		{
			name:   "wrong_metric_type",
			method: http.MethodPost,
			url:    "/update/value/val1/100",
			want: want{
				code:     http.StatusBadRequest,
				response: "unsupported metric type",
			},
		},
		{
			name:   "missing_metric_value",
			method: http.MethodPost,
			url:    "/update/counter/cnt1",
			want: want{
				code:     http.StatusNotFound,
				response: "metric name and/or value not specified",
			},
		},
		{
			name:   "wrong_counter_metric_value",
			method: http.MethodPost,
			url:    "/update/counter/cnt1/10e",
			want: want{
				code:     http.StatusBadRequest,
				response: "invalid metric value",
			},
		},
		{
			name:   "wrong_gauge_metric_value",
			method: http.MethodPost,
			url:    "/update/gauge/gauge1/0x01",
			want: want{
				code:     http.StatusBadRequest,
				response: "invalid metric value",
			},
		},
	}

	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)
	mhandlers := NewMetricsHandlers(msrv)

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.url, nil)
			w := httptest.NewRecorder()

			mhandlers.UpdateHandler()(w, req)
			res := w.Result()
			assert.Equal(t, test.want.code, res.StatusCode)
			defer res.Body.Close()

			resBody, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			if test.want.response != "" {
				assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
			}
		})
	}
}
