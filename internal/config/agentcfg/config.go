package agentcfg

import (
	"flag"
	"fmt"
	"regexp"
	"strings"
)

const (
	defaultServerAddr        = "http://localhost:8080"
	defaultPollIntervalSec   = 2
	defaultReportIntervalSec = 10
)

type enrichedServerAddress string

func (ena *enrichedServerAddress) String() string {
	if *ena == "" {
		return defaultServerAddr
	}
	return string(*ena)
}

func (ena *enrichedServerAddress) Set(addr string) error {
	re := regexp.MustCompile(`^(?:((?:http|https)://)?([^:]+))?(:\d+)$`)
	parts := re.FindStringSubmatch(addr)
	if parts == nil {
		return fmt.Errorf("incorrect network address format")
	}

	if parts[1] == "" {
		parts[1] = "http://"
	}
	if parts[2] == "" {
		parts[2] = "localhost"
	}
	*ena = enrichedServerAddress(strings.Join(parts[1:], ""))
	return nil
}

type Config struct {
	addr              enrichedServerAddress
	PollIntervalSec   int64
	ReportIntervalSec int64
}

func (cfg Config) ServerAddr() string {
	return cfg.addr.String()
}

func DefaultConfig() *Config {
	return &Config{
		addr:              "",
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
	}
}

func (cfg *Config) BindFlags() {
	flag.Var(&cfg.addr, "a", fmt.Sprintf("server address in form of host:port (default: %s)", defaultServerAddr))
	flag.Int64Var(&cfg.PollIntervalSec, "p", 2, "metrics polling interval, seconds (default: 2)")
	flag.Int64Var(&cfg.ReportIntervalSec, "r", 10, "metrics reporting interval, seconds (default: 10)")
}
