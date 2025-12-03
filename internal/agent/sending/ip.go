package sending

import (
	"net"

	"go.uber.org/zap"
)

// IPSender is a helper entity for embedding into higher-level protocol senders
// while providing logging for helper methods
type IPSender struct {
	logger *zap.SugaredLogger
}

func (ns *IPSender) getHostIPAddr() string {
	const loopback = "127.0.0.1"

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		ns.logger.Errorw("error getting host IP addresses", zap.Error(err))
		return loopback
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	// loopback address as a fail case
	ns.logger.Warnw("no host IP address found - use loopback instead")
	return loopback
}
