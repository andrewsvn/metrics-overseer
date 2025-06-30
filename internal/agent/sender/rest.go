package sender

import (
	"fmt"
	"io"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type RestSender struct {
	host string
	port int16

	// we use custom http client here for further customization
	// and to enable connection reuse for sequential server calls
	cl *http.Client
}

func NewRestSender(host string, port int16) *RestSender {
	return &RestSender{
		host: host,
		port: port,

		cl: &http.Client{},
	}
}

func (rs RestSender) CounterMetricSendFunc() MetricSendFunc {
	return func(name string, value string) error {
		return rs.sendMetric(model.Counter, name, value)
	}
}

func (rs RestSender) GaugeMetricSendFunc() MetricSendFunc {
	return func(name string, value string) error {
		return rs.sendMetric(model.Gauge, name, value)
	}
}

func (rs RestSender) sendMetric(mtype string, name string, value string) error {
	req, err := http.NewRequest(http.MethodPost,
		rs.composePostMetricUrl(mtype, name, value), nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Content-Type", "text/plain")
	resp, err := rs.cl.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("error received from metric server (%d) %s",
			resp.StatusCode, body)
	}
	return nil
}

func (rs RestSender) composePostMetricUrl(mtype string, name string, value string) string {
	return fmt.Sprintf("%s:%d/update/%s/%s/%s",
		rs.host, rs.port, mtype, name, value)
}
