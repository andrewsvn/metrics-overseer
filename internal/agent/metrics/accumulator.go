package metrics

import (
	"strconv"

	"github.com/andrewsvn/metrics-overseer/internal/agent/sender"
	"github.com/andrewsvn/metrics-overseer/internal/model"
)

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
		return model.ErrMethodNotSupported
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
		return model.ErrMethodNotSupported
	}

	ma.Values = append(ma.Values, value)
	return nil
}

func (ma *MetricAccumulator) ExtractAndSend(sendfunc sender.MetricSendFunc) error {
	switch ma.MType {
	case model.Counter:
		return ma.extractAndSendCounter(sendfunc)
	case model.Gauge:
		return ma.extractAndSendGauge(sendfunc)
	default:
		return model.ErrMethodNotSupported
	}
}

func (ma *MetricAccumulator) extractAndSendCounter(sendfunc sender.MetricSendFunc) error {
	if ma.Delta == nil {
		return nil
	}

	total := *ma.Delta
	err := sendfunc(ma.ID, ma.MType, strconv.FormatInt(total, 10))
	if err != nil {
		return err
	}

	// if send is successful, remove sent values
	if *ma.Delta == total {
		ma.Delta = nil
	} else {
		*ma.Delta -= total
	}
	return nil
}

func (ma *MetricAccumulator) extractAndSendGauge(sendfunc sender.MetricSendFunc) error {
	if len(ma.Values) == 0 {
		return nil
	}

	// we take average value from accumulated metric values (? maybe use only last)
	var total float64
	count := len(ma.Values)
	for _, v := range ma.Values {
		total += v
	}
	total /= float64(count)

	err := sendfunc(ma.ID, ma.MType, strconv.FormatFloat(total, 'f', 6, 64))
	if err != nil {
		return err
	}

	// if send is successful, remove sent values
	ma.Values = ma.Values[count:]
	return nil
}
