package agentcfg

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
)

const (
	defaultServerAddr        = "http://localhost:8080"
	defaultPollIntervalSec   = 2
	defaultReportIntervalSec = 10
	defaultLogLevel          = "info"
)

type Config struct {
	ServerAddr        string `env:"ADDRESS"`
	PollIntervalSec   int    `env:"POLL_INTERVAL"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL"`
	LogLevel          string `env:"AGENT_LOG_LEVEL" default:"info"`
}

func Read() (*Config, error) {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	err := env.Parse(cfg)
	return cfg, err
}

func Default() *Config {
	cfg := &Config{
		ServerAddr:        defaultServerAddr,
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
		LogLevel:          defaultLogLevel,
	}
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
