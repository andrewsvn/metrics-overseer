// Package servercfg contains possible customization for metrics-overseer server behavior
// in form of environment variables and flags
// Since server provides three types of storages, they are configurable independently
// but only one can be used depending on what settings are provided (see FileStorageConfig, DatabaseConfig)
package servercfg

import (
	"flag"
	"fmt"

	"github.com/caarlos0/env/v6"
)

const (
	defaultAddr             = ":8080"
	defaultGracePeriodSec   = 30
	defaultStoreIntervalSec = 300
	defaultRestoreOnStartup = false
)

// FileStorageConfig contains settings related to in-memory metrics storage with file dumping and restoring at startup
type FileStorageConfig struct {
	StorageFilePath  string `env:"FILE_STORAGE_PATH"`
	StoreIntervalSec int    `env:"STORE_INTERVAL"`
	RestoreOnStartup bool   `env:"RESTORE"`
}

// IsSetUp method checks that file storage mode can be chosen on server start - if not,
// server should fall back to memory storage mode
func (fscfg *FileStorageConfig) IsSetUp() bool {
	return fscfg.StorageFilePath != ""
}

// DatabaseConfig contains settings related to postgres-based metrics storage
type DatabaseConfig struct {
	DBConnString string `env:"DATABASE_DSN"`
}

// IsSetUp method checks that database storage mode can be chosen on server start.
// In this case no checks related to FileStorageConfig are performed
func (dbcfg *DatabaseConfig) IsSetUp() bool {
	return dbcfg.DBConnString != ""
}

// PostgresRetryConfig contains retry policy settings related to retrying postgres database queries in case of
// temporarily unaccessible database
type PostgresRetryConfig struct {
	MaxRetryCount          int `env:"PG_MAX_RETRY_COUNT" envDefault:"3"`
	InitialRetryDelaySec   int `env:"PG_INITIAL_RETRY_DELAY" envDefault:"1"`
	RetryDelayIncrementSec int `env:"PG_RETRY_DELAY_INCREMENT" envDefault:"2"`
}

// SecurityConfig contains settings related to HTTP authentication for clients of metrics-overseer server
// including metrics-overseer agent
type SecurityConfig struct {
	SecretKey      string `env:"KEY"`
	PrivateKeyPath string `env:"CRYPTO_KEY"`
}

// AuditConfig contains settings related to audit of metrics updates - it can be forwarded to a file and/or
// remote http server - if a corresponding setting is provided
type AuditConfig struct {
	AuditFilePath             string `env:"AUDIT_FILE"`
	AuditFileWriteIntervalSec int    `env:"AUDIT_FILE_WRITE_INTERVAL" envDefault:"30"`
	AuditURL                  string `env:"AUDIT_URL"`
}

// Config embeds all server configuration properties to be set by env.Parse or flag.Parse and be used in server code
type Config struct {
	FileStorageConfig
	DatabaseConfig
	PostgresRetryConfig
	SecurityConfig
	AuditConfig

	LogLevel       string `env:"SERVER_LOG_LEVEL" default:"info"`
	Addr           string `env:"ADDRESS"`
	GracePeriodSec int    `env:"SERVER_GRACE_PERIOD"`
	PprofAddr      string `env:"PPROF_ADDRESS"`
}

// Read is used to initialize server Config from environment variables and/or flags. Environment variables have higher
// // priority than flags, so they are parsed later
func Read() (*Config, error) {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	err := env.Parse(cfg)
	return cfg, err
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.Addr, "a", defaultAddr,
		fmt.Sprintf("server address in form of host:port (default: %s)", defaultAddr))
	flag.IntVar(&cfg.GracePeriodSec, "gs", defaultGracePeriodSec,
		fmt.Sprintf("server grace period in seconds (default: %d)", defaultGracePeriodSec))
	flag.StringVar(&cfg.PprofAddr, "pprof", "",
		"pprof endpoints address in form of host:port, must be different from server address "+
			"(pprof disabled if not specified)")

	flag.StringVar(&cfg.StorageFilePath, "f", "",
		"metrics storage file path (should be specified to enable file storage)")
	flag.IntVar(&cfg.StoreIntervalSec, "i", defaultStoreIntervalSec,
		"metrics storing interval in seconds (0 for synchronous store)")
	flag.BoolVar(&cfg.RestoreOnStartup, "r", defaultRestoreOnStartup,
		"flag for restoring metrics on startup")

	flag.StringVar(&cfg.DBConnString, "d", "",
		"postgres database connection string (should be specified to enable postgres storage)")

	flag.StringVar(&cfg.SecretKey, "k", "",
		"secret key for verifying requests and signing responses")
	flag.StringVar(&cfg.PrivateKeyPath, "crypto-key", "",
		"path to PEM file with RSA private key for decrypting requests (no decryption if empty)")

	flag.StringVar(&cfg.AuditFilePath, "audit-file", "",
		"audit file path (should be specified to enable file audit)")
	flag.StringVar(&cfg.AuditURL, "audit-url", "",
		"audit url (should be specified to enable http service audit)")
}
