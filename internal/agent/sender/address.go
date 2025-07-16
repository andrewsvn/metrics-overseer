package sender

import (
	"fmt"
	"regexp"
	"strings"
)

func enrichServerAddress(addr string) (string, error) {
	// regexp to accept server address in 3 possible formats:
	// - protocol://host:port
	// - host:port
	// - :port
	re, err := regexp.Compile(`^(?:((?:http|https)://)?([^:]+))?(:\d+)$`)
	if err != nil {
		return "", fmt.Errorf("can't compile regexp for network address enrichment: %w", err)
	}
	parts := re.FindStringSubmatch(addr)
	if parts == nil {
		return "", fmt.Errorf("incorrect network address: %s", addr)
	}

	if parts[1] == "" {
		parts[1] = "http://"
	}
	if parts[2] == "" {
		parts[2] = "localhost"
	}
	return strings.Join(parts[1:], ""), nil
}
