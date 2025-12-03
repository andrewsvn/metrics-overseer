package grpcsrv

import (
	"context"
	"fmt"
	"net"

	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/handling/grpcsrv/interceptor"
	"github.com/andrewsvn/metrics-overseer/internal/model"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/andrewsvn/metrics-overseer/internal/proto"
)

type MetricsServer struct {
	proto.UnimplementedMetricsServer

	msrv          *service.MetricsService
	trustedSubnet *net.IPNet

	baseLogger *zap.Logger
	logger     *zap.SugaredLogger
}

func NewMetricsServer(
	ms *service.MetricsService,
	securityCfg *servercfg.SecurityConfig,
	logger *zap.Logger,
) (*MetricsServer, error) {
	mhLogger := logger.Sugar().With(zap.String("component", "grpcsrv-metrics-server"))

	var err error

	var trustedSubnet *net.IPNet
	if securityCfg.TrustedSubnet != "" {
		_, trustedSubnet, err = net.ParseCIDR(securityCfg.TrustedSubnet)
		if err != nil {
			return nil, fmt.Errorf("error parsing trusted subnet: %w", err)
		}
		mhLogger.Infow("using trusted subnet for client remote address validation: %s", securityCfg.TrustedSubnet)
	}

	return &MetricsServer{
		msrv:          ms,
		trustedSubnet: trustedSubnet,
		baseLogger:    logger,
		logger:        mhLogger,
	}, nil
}

// GetGRPCServer constructs grpc Server with attached instance of MetricsServer and all needed interceptors
func (s *MetricsServer) GetGRPCServer() *grpc.Server {
	gs := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor.NewAuthorization(s.trustedSubnet).UnaryInterceptor),
	)

	proto.RegisterMetricsServer(gs, s)
	return gs
}

// UpdateMetrics accepts one or more metrics in a request from an agent and performs bulk update
func (s *MetricsServer) UpdateMetrics(
	ctx context.Context,
	request *proto.UpdateMetricsRequest,
) (*proto.UpdateMetricsResponse, error) {
	reqMetrics := request.GetMetrics()
	metrics := make([]*model.Metrics, 0, len(reqMetrics))
	for _, m := range reqMetrics {
		switch m.GetType() {
		case proto.Metric_GAUGE:
			metrics = append(metrics, model.NewGaugeMetricsWithValue(m.GetId(), m.GetValue()))
		case proto.Metric_COUNTER:
			metrics = append(metrics, model.NewCounterMetricsWithDelta(m.GetId(), m.GetDelta()))
		default:
			return nil, status.Error(codes.InvalidArgument, "unknown metric type")
		}
	}

	ipAddr := ctx.Value(interceptor.ContextIPAddrKey).(string)
	err := s.msrv.BatchAccumulateMetrics(ctx, metrics, ipAddr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &proto.UpdateMetricsResponse{}, nil
}
