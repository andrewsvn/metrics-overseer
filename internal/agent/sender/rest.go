package sender

import (
	"fmt"
	"io"
	"net/http"
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

func (rs RestSender) MetricSendFunc() MetricSendFunc {
	return func(id string, mtype string, value string) error {
		req, err := http.NewRequest(http.MethodPost,
			rs.composePostMetricURL(mtype, id, value), nil)
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
}

func (rs RestSender) composePostMetricURL(id string, mtype string, value string) string {
	return fmt.Sprintf("%s:%d/update/%s/%s/%s",
		rs.host, rs.port, mtype, id, value)
}
