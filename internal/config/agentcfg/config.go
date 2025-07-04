package agentcfg

import (
	"flag"
	"fmt"
)

const (
	defaultServerAddr        = "http://localhost:8080"
	defaultPollIntervalSec   = 2
	defaultReportIntervalSec = 10
)

type Config struct {
	ServerAddr        string
	PollIntervalSec   int64
	ReportIntervalSec int64
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:        defaultServerAddr,
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
	}
}

func (c *Config) BindFlags() {
	flag.StringVar(&c.ServerAddr, "a", ":8080",
		fmt.Sprintf("server address in form of host:port (default: %s)", defaultServerAddr))
	flag.Int64Var(&c.PollIntervalSec, "p", 2, "metrics polling interval, seconds (default: 2)")
	flag.Int64Var(&c.ReportIntervalSec, "r", 10, "metrics reporting interval, seconds (default: 10)")
}
