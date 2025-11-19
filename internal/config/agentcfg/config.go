// Package agentcfg contains possible customization for metrics-overseer agent behavior
// in form of environment variables and flags
package agentcfg

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/caarlos0/env/v6"
	flag "github.com/spf13/pflag"
)

const (
	defaultServerAddr        = "http://localhost:8080"
	defaultPollIntervalSec   = 2
	defaultReportIntervalSec = 10
	defaultGracePeriodSec    = 30
	defaultLogLevel          = "info"

	defaultReportMaxRetries             = 3
	defaultReportInitialRetryDelaySec   = 1
	defaultReportRetryDelayIncrementSec = 2
)

// ReportRetryConfig contains retry policy configuration for metrics reporting to metrics-overseer server
type ReportRetryConfig struct {
	MaxRetryCount          int `env:"REPORT_MAX_RETRY_COUNT" json:"report_max_retry_count"`
	InitialRetryDelaySec   int `env:"REPORT_INITIAL_RETRY_DELAY" json:"report_initial_retry_delay_sec"`
	RetryDelayIncrementSec int `env:"REPORT_RETRY_DELAY_INCREMENT" json:"report_retry_delay_increment_sec"`
}

// ReportingConfig contains settings for simultaneous sending of metrics to metrics-overseer server
// used by agent/reporting package methods
type ReportingConfig struct {
	MaxNumberOfRequests int `env:"RATE_LIMIT" json:"rate_limit"`
}

// Config embeds all agent configuration properties to be set by env.Parse or flag.Parse and be used in agent code
type Config struct {
	ReportingConfig
	ReportRetryConfig

	ServerAddr        string `env:"ADDRESS" json:"address"`
	PollIntervalSec   int    `env:"POLL_INTERVAL" json:"poll_interval_sec"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL" json:"report_interval_sec"`
	GracePeriodSec    int    `env:"AGENT_GRACE_PERIOD" json:"grace_period_sec"`
	SecretKey         string `env:"KEY" json:"key"`
	PublicKeyPath     string `env:"CRYPTO_KEY" json:"crypto_key"`
	LogLevel          string `env:"AGENT_LOG_LEVEL" json:"agent_log_level"`

	ConfigFile string `env:"AGENT_CONFIG"`
}

// Read is used to initialize agent Config from environment variables and/or flags. Environment variables have higher
// priority than flags, so they are parsed later
func Read() (*Config, error) {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.ConfigFile != "" {
		fileCfg, err := NewConfigFromJSONFile(cfg.ConfigFile)
		if err != nil {
			return nil, err
		}
		cfg.FillOutEmptyValues(fileCfg)
	}
	cfg.FillOutEmptyValues(NewDefaultConfig())

	return cfg, nil
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.ServerAddr, "a", "",
		fmt.Sprintf("server address in form of host:port (default: %s)", defaultServerAddr))
	flag.IntVar(&cfg.PollIntervalSec, "p", 0,
		fmt.Sprintf("accumulation polling interval, seconds (default: %d)", defaultPollIntervalSec))
	flag.IntVar(&cfg.ReportIntervalSec, "r", 0,
		fmt.Sprintf("accumulation reporting interval, seconds (default: %d)", defaultReportIntervalSec))
	flag.IntVar(&cfg.GracePeriodSec, "gs", 0,
		fmt.Sprintf("accumulation agent graceful shutdown period, seconds (default: %d)", defaultGracePeriodSec))

	flag.IntVar(&cfg.MaxNumberOfRequests, "l", 0,
		fmt.Sprintf("maximum number of simultaneous reporting requests (default: 0). "+
			"If 0, single-thread batching is used"))

	flag.StringVar(&cfg.SecretKey, "k", "",
		"secret key for request signing")
	flag.StringVar(&cfg.PublicKeyPath, "crypto-key", "",
		"path to PEM file with RSA public key for encrypting requests (no encryption if empty)")

	flag.StringVarP(&cfg.ConfigFile, "config", "c", "", "path to JSON config file with default configuration")
}

// FillOutEmptyValues for each missing cfg value tries tp get corresponding value from nextCfg and fill it
// can be used to fill out missing values using config from JSON file and then some default config
func (cfg *Config) FillOutEmptyValues(nextCfg *Config) {
	if cfg.MaxRetryCount == 0 {
		cfg.MaxRetryCount = nextCfg.MaxRetryCount
	}
	if cfg.InitialRetryDelaySec == 0 {
		cfg.InitialRetryDelaySec = nextCfg.InitialRetryDelaySec
	}
	if cfg.RetryDelayIncrementSec == 0 {
		cfg.RetryDelayIncrementSec = nextCfg.RetryDelayIncrementSec
	}

	if cfg.MaxNumberOfRequests == 0 {
		cfg.MaxNumberOfRequests = nextCfg.MaxNumberOfRequests
	}

	if cfg.LogLevel == "" {
		cfg.LogLevel = nextCfg.LogLevel
	}
	if cfg.ServerAddr == "" {
		cfg.ServerAddr = nextCfg.ServerAddr
	}
	if cfg.GracePeriodSec == 0 {
		cfg.GracePeriodSec = nextCfg.GracePeriodSec
	}
	if cfg.PollIntervalSec == 0 {
		cfg.PollIntervalSec = nextCfg.PollIntervalSec
	}
	if cfg.ReportIntervalSec == 0 {
		cfg.ReportIntervalSec = nextCfg.ReportIntervalSec
	}
	if cfg.SecretKey == "" {
		cfg.SecretKey = nextCfg.SecretKey
	}
	if cfg.PublicKeyPath == "" {
		cfg.PublicKeyPath = nextCfg.PublicKeyPath
	}
}

func NewConfigFromJSONFile(path string) (*Config, error) {
	cfgBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("can't read configuration file: %w", err)
	}

	cfg := &Config{}
	err = json.Unmarshal(cfgBytes, cfg)
	if err != nil {
		return nil, fmt.Errorf("can't parse configuration file: %w", err)
	}

	return cfg, nil
}

func NewDefaultConfig() *Config {
	cfg := &Config{
		ReportRetryConfig: ReportRetryConfig{
			MaxRetryCount:          defaultReportMaxRetries,
			InitialRetryDelaySec:   defaultReportInitialRetryDelaySec,
			RetryDelayIncrementSec: defaultReportRetryDelayIncrementSec,
		},
		ServerAddr:        defaultServerAddr,
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
		GracePeriodSec:    defaultGracePeriodSec,
		LogLevel:          defaultLogLevel,
	}
	return cfg
}
