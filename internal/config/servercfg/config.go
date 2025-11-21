// Package servercfg contains possible customization for metrics-overseer server behavior
// in form of environment variables and flags
// Since server provides three types of storages, they are configurable independently
// but only one can be used depending on what settings are provided (see FileStorageConfig, DatabaseConfig)
package servercfg

import (
	"encoding/json"
	"fmt"
	"os"

	"dario.cat/mergo"
	"github.com/caarlos0/env/v6"
	flag "github.com/spf13/pflag"
)

const (
	defaultAddr                  = ":8080"
	defaultServerLogLevel        = "info"
	defaultAuditWriteIntervalSec = 30
	defaultGracePeriodSec        = 30
	defaultStoreIntervalSec      = 300
	defaultRestoreOnStartup      = false

	defaultPGMaxRetryCount       = 3
	defaultPGInitialRetryDelay   = 1
	defaultPGRetryDelayIncrement = 2
)

// FileStorageConfig contains settings related to in-memory metrics storage with file dumping and restoring at startup
type FileStorageConfig struct {
	StorageFilePath  string `env:"FILE_STORAGE_PATH" json:"file_storage_path"`
	StoreIntervalSec int    `env:"STORE_INTERVAL" json:"store_interval"`
	RestoreOnStartup bool   `env:"RESTORE" json:"restore"`
}

// IsSetUp method checks that file storage mode can be chosen on server start - if not,
// server should fall back to memory storage mode
func (fscfg *FileStorageConfig) IsSetUp() bool {
	return fscfg.StorageFilePath != ""
}

// DatabaseConfig contains settings related to postgres-based metrics storage
type DatabaseConfig struct {
	DBConnString string `env:"DATABASE_DSN" json:"database_dsn"`
}

// IsSetUp method checks that database storage mode can be chosen on server start.
// In this case no checks related to FileStorageConfig are performed
func (dbcfg *DatabaseConfig) IsSetUp() bool {
	return dbcfg.DBConnString != ""
}

// PostgresRetryConfig contains retry policy settings related to retrying postgres database queries in case of
// temporarily unaccessible database
type PostgresRetryConfig struct {
	MaxRetryCount          int `env:"PG_MAX_RETRY_COUNT" json:"pg_max_retry_count"`
	InitialRetryDelaySec   int `env:"PG_INITIAL_RETRY_DELAY" json:"pg_initial_retry_delay_sec"`
	RetryDelayIncrementSec int `env:"PG_RETRY_DELAY_INCREMENT" json:"pg_retry_delay_increment_sec"`
}

// SecurityConfig contains settings related to HTTP authentication for clients of metrics-overseer server
// including metrics-overseer agent
type SecurityConfig struct {
	SecretKey      string `env:"KEY" json:"key"`
	PrivateKeyPath string `env:"CRYPTO_KEY" json:"crypto_key"`
}

// AuditConfig contains settings related to audit of metrics updates - it can be forwarded to a file and/or
// remote http server - if a corresponding setting is provided
type AuditConfig struct {
	AuditFilePath             string `env:"AUDIT_FILE" json:"audit_file"`
	AuditFileWriteIntervalSec int    `env:"AUDIT_FILE_WRITE_INTERVAL" json:"audit_file_write_interval"`
	AuditURL                  string `env:"AUDIT_URL" json:"audit_url"`
}

// Config embeds all server configuration properties to be set by env.Parse or flag.Parse and be used in server code
type Config struct {
	FileStorageConfig
	DatabaseConfig
	PostgresRetryConfig
	SecurityConfig
	AuditConfig

	LogLevel       string `env:"SERVER_LOG_LEVEL" json:"server_log_level"`
	Addr           string `env:"ADDRESS" json:"address"`
	GracePeriodSec int    `env:"SERVER_GRACE_PERIOD" json:"server_grace_period"`
	PprofAddr      string `env:"PPROF_ADDRESS" json:"pprof_address"`

	// ConfigFile specifies path to application config in JSON format, if specified
	// it will be parsed to extract mapping that can be used if neither flag nor environment variable is not specified
	ConfigFile string `env:"SERVER_CONFIG"`
}

// Read is used to initialize server Config from environment variables and/or flags. Environment variables have higher
// // priority than flags, so they are parsed later
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
		err = mergo.Merge(cfg, fileCfg)
		if err != nil {
			return nil, fmt.Errorf("unable to merge server configs: %w", err)
		}
	}
	err = mergo.Merge(cfg, NewDefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("unable to merge server configs: %w", err)
	}

	return cfg, nil
}

func (cfg *Config) bindFlags() {
	flag.StringVarP(&cfg.Addr, "addr", "a", "",
		fmt.Sprintf("server address in form of host:port (default: %s)", defaultAddr))
	flag.IntVar(&cfg.GracePeriodSec, "grace-period", 0,
		fmt.Sprintf("server grace period in seconds (default: %d)", defaultGracePeriodSec))
	flag.StringVar(&cfg.PprofAddr, "pprof", "",
		"pprof endpoints address in form of host:port, must be different from server address "+
			"(pprof disabled if not specified)")

	flag.StringVarP(&cfg.StorageFilePath, "store-file", "f", "",
		"metrics storage file path (should be specified to enable file storage)")
	flag.IntVarP(&cfg.StoreIntervalSec, "store-interval", "i", 0,
		"metrics storing interval in seconds (0 for synchronous store)")
	flag.BoolVarP(&cfg.RestoreOnStartup, "restore", "r", false,
		"flag for restoring metrics on startup")

	flag.StringVarP(&cfg.DBConnString, "database-dsn", "d", "",
		"postgres database connection string (should be specified to enable postgres storage)")

	flag.StringVarP(&cfg.SecretKey, "secret-key", "k", "",
		"secret key for verifying requests and signing responses")
	flag.StringVar(&cfg.PrivateKeyPath, "crypto-key", "",
		"path to PEM file with RSA private key for decrypting requests (no decryption if empty)")

	flag.StringVar(&cfg.AuditFilePath, "audit-file", "",
		"audit file path (should be specified to enable file audit)")
	flag.StringVar(&cfg.AuditURL, "audit-url", "",
		"audit url (should be specified to enable http service audit)")

	flag.StringVarP(&cfg.ConfigFile, "config", "c", "", "path to JSON config file with default configuration")
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
	return &Config{
		FileStorageConfig: FileStorageConfig{
			StoreIntervalSec: defaultStoreIntervalSec,
			RestoreOnStartup: defaultRestoreOnStartup,
		},
		DatabaseConfig: DatabaseConfig{},
		PostgresRetryConfig: PostgresRetryConfig{
			MaxRetryCount:          defaultPGMaxRetryCount,
			InitialRetryDelaySec:   defaultPGInitialRetryDelay,
			RetryDelayIncrementSec: defaultPGRetryDelayIncrement,
		},
		AuditConfig: AuditConfig{
			AuditFileWriteIntervalSec: defaultAuditWriteIntervalSec,
		},
		Addr:           defaultAddr,
		LogLevel:       defaultServerLogLevel,
		GracePeriodSec: defaultGracePeriodSec,
	}
}
