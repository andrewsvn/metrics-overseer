package sender

import (
	"fmt"
	"io"
	"net/http"
)

type RestSender struct {
	addr string

	// we use custom http client here for further customization
	// and to enable connection reuse for sequential server calls
	cl *http.Client
}

func NewRestSender(addr string) *RestSender {
	return &RestSender{
		addr: addr,
		cl:   &http.Client{},
	}
}

func (rs RestSender) MetricSendFunc() MetricSendFunc {
	return func(id string, mtype string, value string) error {
		req, err := http.NewRequest(http.MethodPost, rs.composePostMetricURL(id, mtype, value), nil)
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
			return fmt.Errorf("error received from metric server (%d) %s", resp.StatusCode, body)
		}
		return nil
	}
}

func (rs RestSender) composePostMetricURL(id string, mtype string, value string) string {
	return fmt.Sprintf("%s/update/%s/%s/%s", rs.addr, mtype, id, value)
}
