package sender

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

type RestSender struct {
	addr string

	// we use a custom http client here for further customization
	// and to enable connection reuse for sequential server calls
	cl *http.Client
}

func NewRestSender(addr string) (*RestSender, error) {
	enrichedAddr, err := enrichServerAddress(addr)
	if err != nil {
		return nil, fmt.Errorf("can't enrich address for sender to a proper format: %w", err)
	}

	log.Printf("[INFO] Server address for sending reports: %s", enrichedAddr)
	rs := &RestSender{
		addr: enrichedAddr,
		cl:   &http.Client{},
	}
	return rs, nil
}

func (rs RestSender) MetricSendFunc() MetricSendFunc {
	return func(id string, mtype string, value string) error {
		req, err := http.NewRequest(http.MethodPost, rs.composePostMetricURL(id, mtype, value), nil)
		if err != nil {
			return fmt.Errorf("can't construct metric send request: %w", err)
		}

		req.Header.Add("Content-Type", "text/plain")
		resp, err := rs.cl.Do(req)
		if err != nil {
			return fmt.Errorf("error sending request to server %s: %w", rs.addr, err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("can't read response body from metrics send operation: %w", err)
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
