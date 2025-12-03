package handling

import (
	"net"

	"github.com/pkg/errors"
)

var (
	ErrNonTrustedIPAddress = errors.New("IP address is not in trusted subnet")
)

func VerifyTrustedIPAddress(subnet *net.IPNet, ipStr string) error {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return ErrNonTrustedIPAddress
	}
	sub := ip.Mask(subnet.Mask)
	if !sub.Equal(subnet.IP) {
		return ErrNonTrustedIPAddress
	}
	return nil
}
