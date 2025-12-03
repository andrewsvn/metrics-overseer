package sending

import (
	"context"
	"fmt"

	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/proto"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type GRPCSender struct {
	IPSender

	cl      proto.MetricsClient
	retrier *retrying.Executor
	logger  *zap.SugaredLogger
}

func NewGRPCSender(
	grpcAddr string,
	retryPolicy retrying.Policy,
	logger *zap.Logger,
) (*GRPCSender, error) {
	grpcLogger := logger.Sugar().With(zap.String("component", "grpcsrv-sender"))
	retrier := retrying.NewExecutorBuilder(retryPolicy).
		WithLogger(grpcLogger, "sending metrics").
		Build()

	conn, err := grpc.NewClient(grpcAddr)
	if err != nil {
		return nil, fmt.Errorf("unable to create gRPC client: %w", err)
	}

	gs := &GRPCSender{
		IPSender: IPSender{
			logger: grpcLogger,
		},
		cl:      proto.NewMetricsClient(conn),
		retrier: retrier,
		logger:  grpcLogger,
	}
	return gs, nil
}

func (gs *GRPCSender) SendMetricValue(id string, mtype string, value string) error {
	return gs.SendMetricArray([]*model.Metrics{
		model.NewMetricsFromStringValue(id, mtype, value),
	})
}

func (gs *GRPCSender) SendMetric(metric *model.Metrics) error {
	return gs.SendMetricArray([]*model.Metrics{metric})
}

func (gs *GRPCSender) SendMetricArray(metrics []*model.Metrics) error {
	return gs.retrier.Run(func() error {
		return gs.sendRequest(metrics)
	})
}

func (gs *GRPCSender) sendRequest(metrics []*model.Metrics) error {
	md := metadata.New(map[string]string{"X-Real-IP": gs.getHostIPAddr()})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	mlist := make([]*proto.Metric, 0, len(metrics))
	for _, metric := range metrics {
		mlist = append(mlist, buildGRPCMetricForUpdate(metric))
	}

	req := proto.UpdateMetricsRequest_builder{
		Metrics: mlist,
	}.Build()

	_, err := gs.cl.UpdateMetrics(ctx, req)
	if err != nil {
		grpcErr, ok := status.FromError(err)
		if !ok {
			// consider internal sender error as retryable
			return retrying.NewRetryableError(err)
		}

		// check for retryable remote errors
		switch grpcErr.Code() {
		case codes.DeadlineExceeded:
		case codes.Unavailable:
			return retrying.NewRetryableError(err)
		}

		// all other codes considered as non-retryable
		return err
	}

	return nil
}

func buildGRPCMetricForUpdate(metrics *model.Metrics) *proto.Metric {
	switch metrics.MType {
	case model.Counter:
		var delta int64
		if metrics.Delta != nil {
			delta = *metrics.Delta
		}
		return proto.Metric_builder{
			Id:    metrics.ID,
			Type:  proto.Metric_COUNTER,
			Delta: delta,
		}.Build()
	case model.Gauge:
		var value float64
		if metrics.Value != nil {
			value = *metrics.Value
		}
		return proto.Metric_builder{
			Id:    metrics.ID,
			Type:  proto.Metric_GAUGE,
			Value: value,
		}.Build()
	}

	// this should not happen
	return nil
}
