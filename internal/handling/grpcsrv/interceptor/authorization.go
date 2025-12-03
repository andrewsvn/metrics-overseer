package interceptor

import (
	"context"
	"net"

	"github.com/andrewsvn/metrics-overseer/internal/handling"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	ContextIPAddrKey = "ipAddr"
)

type Authorization struct {
	trustedSubnet *net.IPNet
}

func NewAuthorization(subnet *net.IPNet) *Authorization {
	return &Authorization{
		trustedSubnet: subnet,
	}
}

func (s *Authorization) UnaryInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "missing X-Real-IP metadata")
	}

	ipAddr := md.Get("X-Real-IP")
	if len(ipAddr) == 0 {
		return nil, status.Error(codes.Unauthenticated, "missing X-Real-IP address")
	}

	err := handling.VerifyTrustedIPAddress(s.trustedSubnet, ipAddr[0])
	if err != nil {
		return nil, status.Error(codes.PermissionDenied, "X-Real-IP is invalid or not trusted")
	}

	return handler(ctx, req)
}
