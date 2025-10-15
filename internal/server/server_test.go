package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/db"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/mocks"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/mock"

	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testWant struct {
	code        int
	response    string
	contentType string
	resultType  string
}

type testCase struct {
	name   string
	method string
	url    string
	body   string
	want   testWant
}

func TestUpdateByPathHandler(t *testing.T) {
	tests := []testCase{
		{
			name:   "counter_metric",
			method: http.MethodPost,
			url:    "/update/counter/cnt1/10",
			want: testWant{
				code: http.StatusOK,
			},
		},
		{
			name:   "gauge_metric",
			method: http.MethodPost,
			url:    "/update/gauge/gauge1/1.05",
			want: testWant{
				code: http.StatusOK,
			},
		},
		{
			name:   "wrong_http_method",
			method: http.MethodPut,
			url:    "/update/counter/gauge1/1.05",
			want: testWant{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "wrong_metric_type",
			method: http.MethodPost,
			url:    "/update/value/val1/100",
			want: testWant{
				code:     http.StatusBadRequest,
				response: "unsupported metric type: value",
			},
		},
		{
			name:   "missing_metric_value",
			method: http.MethodPost,
			url:    "/update/counter/cnt1",
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "wrong_counter_metric_value",
			method: http.MethodPost,
			url:    "/update/counter/cnt1/10e",
			want: testWant{
				code:     http.StatusBadRequest,
				response: "invalid metric value: 10e",
			},
		},
		{
			name:   "wrong_gauge_metric_value",
			method: http.MethodPost,
			url:    "/update/gauge/gauge1/0x01",
			want: testWant{
				code:     http.StatusBadRequest,
				response: "invalid metric value: 0x01",
			},
		},
	}

	srv := setupServerWithMemStorage()
	defer srv.Close()

	for _, test := range tests {
		updateByPathHandlerSingleTest(t, test, srv)
	}
}

func updateByPathHandlerSingleTest(t *testing.T, test testCase, srv *httptest.Server) {
	req, err := http.NewRequest(test.method, srv.URL+test.url, nil)
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = res.Body.Close()
	}()

	assert.Equal(t, test.want.code, res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if test.want.response != "" {
		assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
	}
}

func TestUpdateValueByJSONHandler(t *testing.T) {
	tests := []testCase{
		{
			name:   "counter_metric",
			method: http.MethodPost,
			body:   `{"id": "cnt1", "type": "counter", "delta": 10}`,
			want: testWant{
				code: http.StatusOK,
			},
		},
		{
			name:   "gauge_metric",
			method: http.MethodPost,
			body:   `{"id": "gauge1", "type": "gauge", "value": 1.05}`,
			want: testWant{
				code: http.StatusOK,
			},
		},
		{
			name:   "wrong_http_method",
			method: http.MethodPut,
			body:   `{"id": "gauge1", "type": "gauge", "value": 1.05}`,
			want: testWant{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "wrong_metric_type",
			method: http.MethodPost,
			body:   `{"id": "gauge1", "type": "value", "value": 1.05}`,
			want: testWant{
				code:     http.StatusBadRequest,
				response: "unsupported metric type: value",
			},
		},
		{
			name:   "missing_counter_metric_value",
			method: http.MethodPost,
			body:   `{"id": "cnt1", "type": "counter", "value": 1.11}`,
			want: testWant{
				code:     http.StatusBadRequest,
				response: "missing counter metric value",
			},
		},
		{
			name:   "missing_gauge_metric_value",
			method: http.MethodPost,
			body:   `{"id": "gauge1", "type": "gauge", "delta": 1}`,
			want: testWant{
				code:     http.StatusBadRequest,
				response: "missing gauge metric value",
			},
		},
	}

	srv := setupServerWithMemStorage()
	defer srv.Close()

	for _, test := range tests {
		updateByJSONHandlerSingleTest(t, test, srv)
	}
}

func updateByJSONHandlerSingleTest(t *testing.T, test testCase, srv *httptest.Server) {
	req, err := http.NewRequest(test.method, srv.URL+"/update", bytes.NewBufferString(test.body))
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = res.Body.Close()
	}()

	assert.Equal(t, test.want.code, res.StatusCode)

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if test.want.response != "" {
		assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
	}
}

func TestGetPlainValueHandler(t *testing.T) {
	tests := []testCase{
		{
			name:   "get_existing_counter",
			method: http.MethodGet,
			url:    "/value/counter/cnt1",
			want: testWant{
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
			want: testWant{
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
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_nonexisting_gauge",
			method: http.MethodGet,
			url:    "/value/gauge/gauge10",
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_gauge_as_counter",
			method: http.MethodGet,
			url:    "/value/counter/gauge1",
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_counter_as_gauge",
			method: http.MethodGet,
			url:    "/value/gauge/cnt1",
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_counter_wrong_method",
			method: http.MethodPost,
			url:    "/value/counter/cnt1",
			want: testWant{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "get_unknown_metric_type",
			method: http.MethodGet,
			url:    "/value/string/str1",
			want: testWant{
				code: http.StatusBadRequest,
			},
		},
	}

	srv := setupServerWithMemStorage()
	defer srv.Close()

	for _, test := range tests {
		getPlainValueHandlerSingleTest(t, test, srv)
	}
}

func getPlainValueHandlerSingleTest(t *testing.T, test testCase, srv *httptest.Server) {
	req, err := http.NewRequest(test.method, srv.URL+test.url, nil)
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = res.Body.Close()
	}()

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
			require.NoError(t, err)
			assert.Equal(t, exp, act)
		case model.Gauge:
			exp, _ := strconv.ParseFloat(test.want.response, 64)
			act, err := strconv.ParseFloat(string(resBody), 64)
			require.NoError(t, err)
			assert.Equal(t, exp, act)
		default:
			assert.Equal(t, test.want.response, strings.TrimSpace(string(resBody)))
		}
	}
}

func TestGetJSONValueHandler(t *testing.T) {
	tests := []testCase{
		{
			name:   "get_existing_counter",
			method: http.MethodPost,
			body:   `{ "id": "cnt1", "type": "counter" }`,
			want: testWant{
				code:        http.StatusOK,
				response:    `{ "id": "cnt1", "type": "counter", "delta": 10 }`,
				contentType: "application/json",
			},
		},
		{
			name:   "get_existing_gauge",
			method: http.MethodPost,
			body:   `{ "id": "gauge1", "type": "gauge" }`,
			want: testWant{
				code:        http.StatusOK,
				response:    `{ "id": "gauge1", "type": "gauge", "value": 3.14 }`,
				contentType: "application/json",
			},
		},
		{
			name:   "get_nonexisting_counter",
			method: http.MethodPost,
			body:   `{ "id": "cnt10", "type": "counter" }`,
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_nonexisting_gauge",
			method: http.MethodPost,
			body:   `{ "id": "gauge10", "type": "gauge" }`,
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_gauge_as_counter",
			method: http.MethodPost,
			body:   `{ "id": "gauge1", "type": "counter" }`,
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_counter_as_gauge",
			method: http.MethodPost,
			body:   `{ "id": "cnt1", "type": "gauge" }`,
			want: testWant{
				code: http.StatusNotFound,
			},
		},
		{
			name:   "get_counter_wrong_method",
			method: http.MethodPut,
			body:   `{ "id": "cnt1", "type": "counter" }`,
			want: testWant{
				code: http.StatusMethodNotAllowed,
			},
		},
		{
			name:   "get_unknown_metric_type",
			method: http.MethodPost,
			body:   `{ "id": "cnt1", "type": "string" }`,
			want: testWant{
				code: http.StatusBadRequest,
			},
		},
	}

	srv := setupServerWithMemStorage()
	defer srv.Close()

	for _, test := range tests {
		getJSONValueHandlerSingleTest(t, test, srv)
	}
}

func getJSONValueHandlerSingleTest(t *testing.T, test testCase, srv *httptest.Server) {
	req, err := http.NewRequest(test.method, srv.URL+"/value", bytes.NewBufferString(test.body))
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = res.Body.Close()
	}()

	assert.Equal(t, test.want.code, res.StatusCode)
	if test.want.contentType != "" {
		assert.Equal(t, test.want.contentType, strings.Split(res.Header.Get("Content-Type"), ";")[0])
	}

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	if test.want.response != "" {
		assert.JSONEq(t, test.want.response, string(resBody))
	}
}

func TestGetAllMetricsPage(t *testing.T) {
	srv := setupServerWithMemStorage()
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = res.Body.Close()
	}()

	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, "text/html", strings.Split(res.Header.Get("Content-Type"), ";")[0])

	resBody, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	assert.Regexp(t, regexp.MustCompile(`(?s)<html>.*<body>.*<tr>.*<td>cnt1</td>\s*<td>counter</td>\s*<td>10</td>`),
		string(resBody))
	assert.Regexp(t, regexp.MustCompile(`(?s)<tr>.*<td>gauge1</td>\s*<td>gauge</td>\s*<td>3.14</td>`),
		string(resBody))
}

func TestDBConnectionPing(t *testing.T) {
	mconn := new(mocks.MockConnection)
	mconn.EXPECT().Pool().Return(&pgxpool.Pool{})

	srv := setupServerWithDummyDBStorage(mconn)
	defer srv.Close()

	mconn.EXPECT().Ping(mock.Anything).Return(nil).Once()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/ping", nil)
	require.NoError(t, err)

	res, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = res.Body.Close()
	}()
	assert.Equal(t, http.StatusOK, res.StatusCode)

	mconn.EXPECT().Ping(mock.Anything).Return(fmt.Errorf("no connection")).Once()

	res2, err := srv.Client().Do(req)
	require.NoError(t, err)
	defer func() {
		_ = res2.Body.Close()
	}()
	assert.Equal(t, http.StatusInternalServerError, res2.StatusCode)
}

func setupServerWithMemStorage() *httptest.Server {
	ctx := context.Background()
	logger, _ := logging.NewZapLogger("info")

	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)

	cm := model.NewCounterMetrics("cnt1")
	cm.AddCounter(10)
	_ = msrv.AccumulateMetric(ctx, cm, "")

	gm := model.NewGaugeMetrics("gauge1")
	gm.SetGauge(3.14)
	_ = msrv.AccumulateMetric(ctx, gm, "")

	mhandlers := handler.NewMetricsHandlers(msrv, &servercfg.SecurityConfig{}, logger)

	return httptest.NewServer(mhandlers.GetRouter())
}

func setupServerWithDummyDBStorage(conn db.Connection) *httptest.Server {
	logger, _ := logging.NewZapLogger("info")

	mstor := repository.NewPostgresDBStorage(conn, logger, &retrying.NoRetryPolicy{})
	msrv := service.NewMetricsService(mstor)
	mhandlers := handler.NewMetricsHandlers(msrv, &servercfg.SecurityConfig{}, logger)

	return httptest.NewServer(mhandlers.GetRouter())
}
