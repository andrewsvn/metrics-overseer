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
	PollIntervalSec   int
	ReportIntervalSec int
}

func DefaultConfig() *Config {
	return &Config{
		ServerAddr:        defaultServerAddr,
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
	}
}

func ReadFromCLArgs() *Config {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	return cfg
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.ServerAddr, "a", defaultServerAddr,
		fmt.Sprintf("server address in form of host:port (default: %s)", defaultServerAddr))
	flag.IntVar(&cfg.PollIntervalSec, "p", defaultPollIntervalSec,
		fmt.Sprintf("metrics polling interval, seconds (default: %d)", defaultPollIntervalSec))
	flag.IntVar(&cfg.ReportIntervalSec, "r", defaultReportIntervalSec,
		fmt.Sprintf("metrics reporting interval, seconds (default: %d)", defaultReportIntervalSec))
}
