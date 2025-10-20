package repository

import (
	"context"
	"fmt"

	"github.com/andrewsvn/metrics-overseer/internal/model"
)

func ExampleMemStorage_AddCounter() {
	ms := NewMemStorage()
	ctx := context.Background()

	// set new metric value and check it afterward
	_ = ms.AddCounter(ctx, "foo", 1)
	fmt.Println("Added 1 to foo counter")
	m, _ := ms.GetByID(ctx, "foo")
	// metric delta is a pointer so it should be checked for nil
	fmt.Printf("Foo delta: %d\n", *m.Delta)

	// accumulate this metric further
	_ = ms.AddCounter(ctx, "foo", 2)
	fmt.Println("Added 2 to foo counter")
	m, _ = ms.GetByID(ctx, "foo")
	fmt.Printf("Foo delta: %d\n", *m.Delta)

	// Output:
	// Added 1 to foo counter
	// Foo delta: 1
	// Added 2 to foo counter
	// Foo delta: 3
}

func ExampleMemStorage_SetGauge() {
	ms := NewMemStorage()
	ctx := context.Background()

	// set new metric value and check it afterward
	_ = ms.SetGauge(ctx, "foo", 1.0)
	fmt.Println("Set 1 to foo gauge")
	m, _ := ms.GetByID(ctx, "foo")
	// metric value is a pointer so it should be checked for nil
	fmt.Printf("Foo value: %f\n", *m.Value)

	// accumulate this metric further
	_ = ms.SetGauge(ctx, "foo", 1.5)
	fmt.Println("Set 1.5 to foo gauge")
	m, _ = ms.GetByID(ctx, "foo")
	fmt.Printf("Foo value: %f\n", *m.Value)

	// Output:
	// Set 1 to foo gauge
	// Foo value: 1.000000
	// Set 1.5 to foo gauge
	// Foo value: 1.500000
}

func ExampleMemStorage_BatchUpdate() {
	ms := NewMemStorage()
	ctx := context.Background()

	metrics := []*model.Metrics{
		model.NewCounterMetricsWithDelta("foo1", 10),
		model.NewCounterMetricsWithDelta("foo2", 5),
		model.NewCounterMetricsWithDelta("foo3", 7),
		model.NewGaugeMetricsWithValue("bar1", 3.14),
		model.NewGaugeMetricsWithValue("bar2", 4.56),
		model.NewGaugeMetricsWithValue("bar3", 3.33),
	}

	_ = ms.BatchUpdate(ctx, metrics)

	metrics = []*model.Metrics{
		model.NewCounterMetricsWithDelta("foo1", 2),
		model.NewCounterMetricsWithDelta("foo2", 3),
		model.NewGaugeMetricsWithValue("bar1", 2.72),
		model.NewGaugeMetricsWithValue("bar2", 0.0),
	}
	_ = ms.BatchUpdate(ctx, metrics)

	metrics, _ = ms.GetAllSorted(ctx)
	fmt.Printf("Total number of metrics: %d\n", len(metrics))
	for _, m := range metrics {
		if m.MType == model.Gauge {
			fmt.Printf("Metric %s (type %s): %f\n", m.ID, m.MType, *m.Value)
		} else {
			fmt.Printf("Metric %s (type %s): %d\n", m.ID, m.MType, *m.Delta)
		}
	}

	// Output:
	// Total number of metrics: 6
	// Metric bar1 (type gauge): 2.720000
	// Metric bar2 (type gauge): 0.000000
	// Metric bar3 (type gauge): 3.330000
	// Metric foo1 (type counter): 12
	// Metric foo2 (type counter): 8
	// Metric foo3 (type counter): 7
}
