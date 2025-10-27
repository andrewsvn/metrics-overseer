package model

import (
	"crypto/md5"
	"strconv"
	"strings"
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
	m.updateHash()
	return m
}

func NewGaugeMetrics(id string) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Gauge,
	}
	m.updateHash()
	return m
}

func NewCounterMetrics(id string) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Counter,
	}
	m.updateHash()
	return m
}

func NewGaugeMetricsWithValue(id string, value float64) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Gauge,
		Value: &value,
	}
	m.updateHash()
	return m
}

func NewCounterMetricsWithDelta(id string, delta int64) *Metrics {
	m := &Metrics{
		ID:    id,
		MType: Counter,
		Delta: &delta,
	}
	m.updateHash()
	return m
}

func (m *Metrics) SetGauge(value float64) {
	m.Value = &value
	m.updateHash()
}

func (m *Metrics) AddCounter(delta int64) {
	if m.Delta == nil {
		m.Delta = &delta
	} else {
		*m.Delta += delta
	}
	m.updateHash()
}

func (m *Metrics) Reset() {
	m.Delta = nil
	m.Value = nil
	m.updateHash()
}

func (m *Metrics) updateHash() {
	parts := make([]string, 4)
	parts[0] = m.ID
	parts[1] = m.MType
	parts[2] = "nil"
	if m.Delta != nil {
		parts[2] = strconv.FormatInt(*m.Delta, 10)
	}
	parts[3] = "nil"
	if m.Value != nil {
		parts[3] = strconv.FormatFloat(*m.Value, 'f', -1, 64)
	}

	hash := md5.New()
	hash.Write([]byte(strings.Join(parts, "#")))
	m.Hash = string(hash.Sum(nil))
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
