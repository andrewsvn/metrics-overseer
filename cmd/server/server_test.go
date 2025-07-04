package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/model"
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
				code: http.StatusMethodNotAllowed,
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
				code: http.StatusNotFound,
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

	srv := setupServer()
	defer srv.Close()

	for _, test := range tests {
		req, err := http.NewRequest(test.method, srv.URL+test.url, nil)
		require.NoError(t, err)

		res, err := srv.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, test.want.code, res.StatusCode)

		resBody, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		if test.want.response != "" {
			assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
		}
	}
}

func TestGetValueHandler(t *testing.T) {
	type want struct {
		code        int
		response    string
		contentType string
		resultType  string
	}
	tests := []struct {
		name   string
		method string
		url    string
		want   want
	}{
		{
			name:   "get_existing_counter",
			method: http.MethodGet,
			url:    "/value/counter/cnt1",
			want: want{
				code:        http.StatusOK,
				response:    "10",
				contentType: "text/plain",
				resultType:  model.Counter,
			},
		},
		{
			name:   "get_existing_gauge",
			method: http.MethodGet,
			url:    "/value/gauge/gauge1",
			want: want{
				code:        http.StatusOK,
				response:    "3.14",
				contentType: "text/plain",
				resultType:  model.Gauge,
			},
		},
		{
			name:   "get_nonexisting_counter",
			method: http.MethodGet,
			url:    "/value/counter/cnt10",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_nonexisting_gauge",
			method: http.MethodGet,
			url:    "/value/gauge/gauge10",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_gauge_as_counter",
			method: http.MethodGet,
			url:    "/value/counter/gauge1",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_counter_as_gauge",
			method: http.MethodGet,
			url:    "/value/gauge/cnt1",
			want: want{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_counter_wrong_method",
			method: http.MethodPost,
			url:    "/value/counter/cnt1",
			want: want{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "get_unknown_metric_type",
			method: http.MethodGet,
			url:    "/value/string/str1",
			want: want{
				code: http.StatusBadRequest,
			},
		},
	}

	srv := setupServer()
	defer srv.Close()

	for _, test := range tests {
		req, err := http.NewRequest(test.method, srv.URL+test.url, nil)
		require.NoError(t, err)

		res, err := srv.Client().Do(req)
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, test.want.code, res.StatusCode)
		if test.want.contentType != "" {
			assert.Equal(t, test.want.contentType, strings.Split(res.Header.Get("Content-Type"), ";")[0])
		}

		resBody, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		if test.want.response != "" {
			switch test.want.resultType {
			case model.Counter:
				exp, _ := strconv.ParseInt(test.want.response, 10, 64)
				act, err := strconv.ParseInt(string(resBody), 10, 64)
				assert.NoError(t, err)
				assert.Equal(t, exp, act)
			case model.Gauge:
				exp, _ := strconv.ParseFloat(test.want.response, 64)
				act, err := strconv.ParseFloat(string(resBody), 64)
				assert.NoError(t, err)
				assert.Equal(t, exp, act)
			default:
				assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
			}
		}
	}
}

func TestGetAllMetricsPage(t *testing.T) {
	srv := setupServer()
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/html", strings.Split(res.Header.Get("Content-Type"), ";")[0])

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`(?s)<html>.*<body>.*<tr>.*<td>cnt1</td>\s*<td>counter</td>\s*<td>10</td>`), string(resBody))
	assert.Regexp(t, regexp.MustCompile(`(?s)<tr>.*<td>gauge1</td>\s*<td>gauge</td>\s*<td>3.14</td>`), string(resBody))
}

func setupServer() *httptest.Server {
	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)
	msrv.AccumulateCounter("cnt1", 10)
	msrv.SetGauge("gauge1", 3.14)
	mhandlers := handler.NewMetricsHandlers(msrv)

	return httptest.NewServer(mhandlers.GetRouter())
}
