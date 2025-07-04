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

type EnrichedNetAddress string

func (ena *EnrichedNetAddress) String() string {
	return string(*ena)
}

func (ena *EnrichedNetAddress) Set(addr string) error {
	if addr == "" {
		*ena = defaultServerAddr
		return nil
	}

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
	*ena = EnrichedNetAddress(strings.Join(parts[1:], ""))
	return nil
}

type Config struct {
	addr              EnrichedNetAddress
	PollIntervalSec   int64
	ReportIntervalSec int64
}

func (cfg Config) ServerAddr() string {
	return string(cfg.addr)
}

func DefaultConfig() *Config {
	return &Config{
		addr:              defaultServerAddr,
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
	}
}

func (c *Config) BindFlags() {
	flag.Var(&c.addr, "a", fmt.Sprintf("server address in form of host:port (default: %s)", defaultServerAddr))
	flag.Int64Var(&c.PollIntervalSec, "p", 2, "metrics polling interval, seconds (default: 2)")
	flag.Int64Var(&c.ReportIntervalSec, "r", 10, "metrics reporting interval, seconds (default: 10)")
}
