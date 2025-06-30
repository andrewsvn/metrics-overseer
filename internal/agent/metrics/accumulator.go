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
	if ma.Delta == nil {
		ma.Delta = &inc
	} else {
		*ma.Delta += inc
	}
	return nil
}

func (ma *MetricAccumulator) AccumulateGauge(value float64) error {
	ma.Values = append(ma.Values, value)
	return nil
}

func (ma *MetricAccumulator) ExtractAndSend(ms sender.MetricSendFunc) error {
	switch ma.MType {
	case model.Counter:
		return ma.extractAndSendCounter(ms)
	case model.Gauge:
		return ma.extractAndSendGauge(ms)
	default:
		return model.ErrMethodNotSupported
	}
}

func (ma *MetricAccumulator) extractAndSendCounter(ms sender.MetricSendFunc) error {
	if ma.Delta == nil {
		return nil
	}

	total := *ma.Delta
	err := ms(ma.ID, strconv.FormatInt(total, 10))
	if err != nil {
		return err
	}

	// if send is successful, remove sent values
	*ma.Delta -= total
	return nil
}

func (ma *MetricAccumulator) extractAndSendGauge(ms sender.MetricSendFunc) error {
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

	err := ms(ma.ID, strconv.FormatFloat(total, 'f', 6, 64))
	if err != nil {
		return err
	}

	// if send is successful, remove sent values
	ma.Values = ma.Values[count:]
	return nil
}
