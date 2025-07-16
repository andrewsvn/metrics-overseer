package metrics

import (
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

// Accumulator manages metric lifecycle between pollings and sendings
// accumulated values are stored in incremental counter for "counter" type metrics and in values list for "gauge" type metrics
// for "counter" type metrics:
//	each poll calls @AccumulateCounter method to increment stored value or initialize new in case nothing accumulated yet
//  each report calls @ExtractAndSend method which gets accumulated value and tries to send it to the server,
//    then cleans it in success case to eliminate double accumulation on agent and server sides
//  cleanup is done by decrementing metric by sent value to correctly handle multi-threading and incrementing counter from another thread
// for "gauge" type metrics:
//  each poll calls @AccumulateGauge method to add new collected value to the list
//  each report calls @ExtractAndSend method which takes average value of accumulated ones and tries to send it to the server,
//    then removes processed slice of list so old values don't impact next sends

type MetricAccumulator struct {
	ID     string
	MType  string
	Delta  *int64
	Values []float64
}

func NewMetricAccumulator(id string, mtype string) *MetricAccumulator {
	return &MetricAccumulator{
		ID:    id,
		MType: mtype,
	}
}

func (ma *MetricAccumulator) AccumulateCounter(inc int64) error {
	if ma.MType != model.Counter {
		return fmt.Errorf("%w: expected counter, got %v", model.ErrIncorrectAccess, ma.MType)
	}

	if ma.Delta == nil {
		ma.Delta = &inc
	} else {
		*ma.Delta += inc
	}
	return nil
}

func (ma *MetricAccumulator) AccumulateGauge(value float64) error {
	if ma.MType != model.Gauge {
		return model.ErrIncorrectAccess
	}

	ma.Values = append(ma.Values, value)
	return nil
}

func (ma *MetricAccumulator) ExtractAndSend(sendfunc sender.MetricStructSendFunc) error {
	switch ma.MType {
	case model.Counter:
		return ma.extractAndSendCounter(sendfunc)
	case model.Gauge:
		return ma.extractAndSendGauge(sendfunc)
	default:
		return fmt.Errorf("unknown metric type")
	}
}

func (ma *MetricAccumulator) extractAndSendCounter(sendfunc sender.MetricStructSendFunc) error {
	if ma.Delta == nil {
		return nil
	}

	total := *ma.Delta
	err := sendfunc(model.NewMetrics(ma.ID, ma.MType, ma.Delta, nil))
	if err != nil {
		return fmt.Errorf("error sending accumulated metric id=%s value=%d to server: %w", ma.ID, total, err)
	}

	// if send is successful, remove sent values
	if *ma.Delta == total {
		ma.Delta = nil
	} else {
		*ma.Delta -= total
	}
	return nil
}

func (ma *MetricAccumulator) extractAndSendGauge(sendfunc sender.MetricStructSendFunc) error {
	if len(ma.Values) == 0 {
		return nil
	}

	// we take average value from accumulated metric values (suggestion: maybe use only last)
	var total float64
	count := len(ma.Values)
	for _, v := range ma.Values {
		total += v
	}
	total /= float64(count)

	err := sendfunc(model.NewMetrics(ma.ID, ma.MType, nil, &total))
	if err != nil {
		return fmt.Errorf("error sending accumulated metric id=%s value=%f to server: %w", ma.ID, total, err)
	}

	// if send is successful, remove sent values
	ma.Values = ma.Values[count:]
	return nil
}
