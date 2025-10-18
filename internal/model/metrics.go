package model

import (
	"crypto/md5"
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
	Hash  string   `json:"-"`
}

func NewMetrics(id string, mType string, delta *int64, value *float64) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: mType,
		Delta: delta,
		Value: value,
	}
	m.UpdateHash()
	return m
}

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

func NewGaugeMetricsWithValue(id string, value float64) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Gauge,
		Value: &value,
	}
	m.UpdateHash()
	return m
}

func NewCounterMetricsWithDelta(id string, delta int64) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Counter,
		Delta: &delta,
	}
	m.UpdateHash()
	return m
}

func (m *Metrics) SetGauge(value float64) {
	m.Value = &value
	m.UpdateHash()
}

func (m *Metrics) AddCounter(delta int64) {
	if m.Delta == nil {
		m.Delta = &delta
	} else {
		*m.Delta += delta
	}
	m.UpdateHash()
}

func (m *Metrics) Reset() {
	m.Delta = nil
	m.Value = nil
	m.UpdateHash()
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

func (m *Metrics) StringValue() string {
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
