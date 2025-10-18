package audit

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

type Payload struct {
	Timestamp   int64    `json:"ts"`
	MetricNames []string `json:"metrics"`
	IPAddress   string   `json:"ip_address"`
}

func NewPayload(ts time.Time, ipAddr string, metrics ...*model.Metrics) *Payload {
	names := make([]string, 0)
	for _, m := range metrics {
		names = append(names, m.ID)
	}

	return &Payload{
		Timestamp:   ts.Unix(),
		MetricNames: names,
		IPAddress:   ipAddr,
	}
}

func (p *Payload) Serialize() ([]byte, error) {
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("error serializing audit payload: %s", err)
	}
	return b, nil
}
