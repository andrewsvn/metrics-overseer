package sender

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"go.uber.org/zap"
	"io"
	"net/http"
)

type RestSender struct {
	addr string

	// we use a custom http client here for further customization
	// and to enable connection reuse for sequential server calls
	cl *http.Client

	logger *zap.Logger
}

func NewRestSender(addr string, logger *zap.Logger) (*RestSender, error) {
	enrichedAddr, err := enrichServerAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("can't enrich address for sender to a proper format: %w", err)
	}

	logger.Info(fmt.Sprintf("Sender address for sending reports: %s", enrichedAddr))
	rs := &RestSender{
		addr:   enrichedAddr,
		cl:     &http.Client{},
		logger: logger,
	}
	return rs, nil
}

func (rs RestSender) ValueSendFunc() MetricValueSendFunc {
	return func(id string, mtype string, value string) error {
		req, err := http.NewRequest(http.MethodPost, rs.composePostMetricByPathURL(id, mtype, value), nil)
		if err != nil {
			return fmt.Errorf("can't construct metric send request: %w", err)
		}
		req.Header.Add("Content-Type", "text/plain")

		return rs.sendRequest(req)
	}
}

func (rs RestSender) StructSendFunc() MetricStructSendFunc {
	return func(metric *model.Metrics) error {
		body, err := json.Marshal(metric)
		if err != nil {
			return fmt.Errorf("can't construct metric send request: %w", err)
		}
		req, err := http.NewRequest(http.MethodPost, rs.addr+"/update", bytes.NewBufferString(string(body)))
		if err != nil {
			return fmt.Errorf("can't construct metric send request: %w", err)
		}
		req.Header.Add("Content-Type", "text/plain")

		return rs.sendRequest(req)
	}
}

func (rs RestSender) sendRequest(req *http.Request) error {
	resp, err := rs.cl.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request to server %s: %w", rs.addr, err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			rs.logger.Error("error closing response body", zap.Error(err))
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("can't read response body from metrics send operation: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error received from metric server (%d) %s", resp.StatusCode, body)
	}
	return nil
}

func (rs RestSender) composePostMetricByPathURL(id string, mtype string, value string) string {
	return fmt.Sprintf("%s/update/%s/%s/%s", rs.addr, mtype, id, value)
}
