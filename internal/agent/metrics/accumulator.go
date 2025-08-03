package metrics

import (
	"errors"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"sync"
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

	mutex        sync.Mutex
	isStaged     bool
	stagedDelta  int64
	stagedValues []float64
}

var (
	ErrUnknownMetricType = errors.New("unknown metric type")
	ErrWrongStagingState = errors.New("incorrect operation for current staging state")
)

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

	ma.mutex.Lock()
	defer ma.mutex.Unlock()

	if ma.Delta == nil {
		ma.Delta = &inc
	} else {
		*ma.Delta += inc
	}
	return nil
}

func (ma *MetricAccumulator) AccumulateGauge(value float64) error {
	if ma.MType != model.Gauge {
		return fmt.Errorf("%w: expected gauge, got %v", model.ErrIncorrectAccess, ma.MType)
	}

	ma.mutex.Lock()
	defer ma.mutex.Unlock()

	ma.Values = append(ma.Values, value)
	return nil
}

// StageChanges prepares accumulated metric for sending to server
// if no values were accumulated then nil is returned, so a caller must explicitly check the returned metric for nil
func (ma *MetricAccumulator) StageChanges() (*model.Metrics, error) {
	if ma.isStaged {
		return nil, ErrWrongStagingState
	}

	ma.mutex.Lock()
	defer ma.mutex.Unlock()

	switch ma.MType {
	case model.Counter:
		return ma.stageCounterChanges(), nil
	case model.Gauge:
		return ma.stageGaugeChanges(), nil
	}
	return nil, ErrUnknownMetricType
}

func (ma *MetricAccumulator) stageCounterChanges() *model.Metrics {
	if ma.Delta == nil {
		return nil
	}

	ma.isStaged = true
	ma.stagedDelta = *ma.Delta
	ma.Delta = nil
	return model.NewMetrics(ma.ID, ma.MType, &ma.stagedDelta, nil)
}

func (ma *MetricAccumulator) stageGaugeChanges() *model.Metrics {
	if len(ma.Values) == 0 {
		return nil
	}

	ma.isStaged = true
	ma.stagedValues = append([]float64{}, ma.Values...)
	ma.Values = ma.Values[:0]

	// we take average value from accumulated metric values (suggestion: maybe use only last)
	var total float64
	count := len(ma.stagedValues)
	for _, v := range ma.stagedValues {
		total += v
	}
	total /= float64(count)
	return model.NewMetrics(ma.ID, ma.MType, nil, &total)
}

func (ma *MetricAccumulator) RollbackStaged() error {
	if !ma.isStaged {
		return nil
	}

	ma.mutex.Lock()
	defer ma.mutex.Unlock()

	switch ma.MType {
	case model.Counter:
		ma.rollbackStagedCounter()
	case model.Gauge:
		ma.rollbackStagedGauge()
	default:
		ma.isStaged = false
	}
	return nil
}

func (ma *MetricAccumulator) rollbackStagedCounter() {
	ma.isStaged = false
	if ma.Delta == nil {
		delta := ma.stagedDelta
		ma.Delta = &delta
	} else {
		*ma.Delta += ma.stagedDelta
	}
	ma.stagedDelta = 0
}

func (ma *MetricAccumulator) rollbackStagedGauge() {
	ma.isStaged = false
	ma.Values = append(ma.stagedValues, ma.Values...)
	ma.stagedValues = ma.stagedValues[:0]
}

func (ma *MetricAccumulator) CommitStaged() error {
	if !ma.isStaged {
		return ErrWrongStagingState
	}

	ma.mutex.Lock()
	defer ma.mutex.Unlock()

	switch ma.MType {
	case model.Counter:
		ma.stagedDelta = 0
	case model.Gauge:
		ma.stagedValues = ma.stagedValues[:0]
	}
	ma.isStaged = false
	return nil
}
