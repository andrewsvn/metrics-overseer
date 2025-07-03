package model

import (
	"crypto/md5"
	"errors"
	"fmt"
	"strconv"
)

const (
	Counter = "counter"
	Gauge   = "gauge"
)

// store Delta and Value as pointers to support uninitialized state
// separated from default value without additional flags
type Metrics struct {
	ID    string   `json:"id"`
	MType string   `json:"type"`
	Delta *int64   `json:"delta,omitempty"`
	Value *float64 `json:"value,omitempty"`
	Hash  string   `json:"hash,omitempty"`
}

var (
	ErrMethodNotSupported = errors.New("access method not supported for metric")
)

func NewGaugeMetrics(id string) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Gauge,
	}
	m.UpdateHash()
	return m
}

func NewCounterMetrics(id string) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Counter,
	}
	m.UpdateHash()
	return m
}

func (m *Metrics) Reset() {
	m.Delta = nil
	m.Value = nil
	m.UpdateHash()
}

func (m Metrics) GetGauge() (*float64, error) {
	if m.MType != Gauge {
		return nil, ErrMethodNotSupported
	}
	return m.Value, nil
}

func (m Metrics) GetCounter() (*int64, error) {
	if m.MType != Counter {
		return nil, ErrMethodNotSupported
	}
	return m.Delta, nil
}

func (m *Metrics) SetGauge(value float64) error {
	if m.MType != Gauge {
		return ErrMethodNotSupported
	}
	m.Value = &value
	m.UpdateHash()
	return nil
}

func (m *Metrics) AddCounter(delta int64) error {
	if m.MType != Counter {
		return ErrMethodNotSupported
	}

	if m.Delta == nil {
		m.Delta = &delta
	} else {
		*m.Delta += delta
	}
	m.UpdateHash()
	return nil
}

func (m *Metrics) UpdateHash() {
	bytes := fmt.Appendf(nil, "%s#%s", m.ID, m.MType)
	if m.Delta != nil {
		bytes = fmt.Appendf(bytes, "#%d", *m.Delta)
	} else {
		bytes = fmt.Append(bytes, "#nil")
	}
	if m.Value != nil {
		bytes = fmt.Appendf(bytes, "#%f", *m.Value)
	} else {
		bytes = fmt.Append(bytes, "#nil")
	}

	m.Hash = string(md5.New().Sum(bytes))
}

func (m Metrics) StringValue() string {
	const NotAvailable = "N/A"
	switch m.MType {
	case Counter:
		if m.Delta == nil {
			return NotAvailable
		}
		return strconv.FormatInt(*m.Delta, 10)
	case Gauge:
		if m.Value == nil {
			return NotAvailable
		}
		return strconv.FormatFloat(*m.Value, 'f', -1, 64)
	default:
		return NotAvailable
	}
}
