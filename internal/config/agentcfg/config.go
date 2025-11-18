// Package agentcfg contains possible customization for metrics-overseer agent behavior
// in form of environment variables and flags
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
	defaultGracePeriodSec    = 30
	defaultLogLevel          = "info"
)

// ReportRetryConfig contains retry policy configuration for metrics reporting to metrics-overseer server
type ReportRetryConfig struct {
	MaxRetryCount          int `env:"REPORT_RETRY_COUNT" envDefault:"3"`
	InitialRetryDelaySec   int `env:"REPORT_INITIAL_RETRY_DELAY" envDefault:"1"`
	RetryDelayIncrementSec int `env:"REPORT_RETRY_DELAY_INCREMENT" envDefault:"2"`
}

// ReportingConfig contains settings for simultaneous sending of metrics to metrics-overseer server
// used by agent/reporting package methods
type ReportingConfig struct {
	MaxNumberOfRequests int `env:"RATE_LIMIT"`
}

// Config embeds all agent configuration properties to be set by env.Parse or flag.Parse and be used in agent code
type Config struct {
	ReportingConfig
	ReportRetryConfig

	ServerAddr        string `env:"ADDRESS"`
	PollIntervalSec   int    `env:"POLL_INTERVAL"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL"`
	GracePeriodSec    int    `env:"AGENT_GRACE_PERIOD"`
	SecretKey         string `env:"KEY"`
	PublicKeyPath     string `env:"CRYPTO_KEY"`
	LogLevel          string `env:"AGENT_LOG_LEVEL" default:"info"`
}

// Read is used to initialize agent Config from environment variables and/or flags. Environment variables have higher
// priority than flags, so they are parsed later
func Read() (*Config, error) {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	err := env.Parse(cfg)
	return cfg, err
}

// Default config can be used for tests that require to initialize certain agent components with config values
func Default() *Config {
	cfg := &Config{
		ReportRetryConfig: ReportRetryConfig{
			MaxRetryCount:          3,
			InitialRetryDelaySec:   1,
			RetryDelayIncrementSec: 2,
		},
		ServerAddr:        defaultServerAddr,
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
		GracePeriodSec:    defaultGracePeriodSec,
		LogLevel:          defaultLogLevel,
	}
	return cfg
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.ServerAddr, "a", defaultServerAddr,
		fmt.Sprintf("server address in form of host:port (default: %s)", defaultServerAddr))
	flag.IntVar(&cfg.PollIntervalSec, "p", defaultPollIntervalSec,
		fmt.Sprintf("accumulation polling interval, seconds (default: %d)", defaultPollIntervalSec))
	flag.IntVar(&cfg.ReportIntervalSec, "r", defaultReportIntervalSec,
		fmt.Sprintf("accumulation reporting interval, seconds (default: %d)", defaultReportIntervalSec))
	flag.IntVar(&cfg.GracePeriodSec, "gs", defaultGracePeriodSec,
		fmt.Sprintf("accumulation agent graceful shutdown period, seconds (default: %d)", defaultGracePeriodSec))

	flag.IntVar(&cfg.MaxNumberOfRequests, "l", 0,
		fmt.Sprintf("maximum number of simultaneous reporting requests (default: 0). "+
			"If 0, single-thread batching is used"))

	flag.StringVar(&cfg.SecretKey, "k", "",
		"secret key for request signing")
	flag.StringVar(&cfg.PublicKeyPath, "crypto-key", "",
		"path to PEM file with RSA public key for encrypting requests (no encryption if empty)")
}
