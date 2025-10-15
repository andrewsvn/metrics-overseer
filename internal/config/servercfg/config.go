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

type FileStorageConfig struct {
	StorageFilePath  string `env:"FILE_STORAGE_PATH"`
	StoreIntervalSec int    `env:"STORE_INTERVAL"`
	RestoreOnStartup bool   `env:"RESTORE"`
}

func (fscfg *FileStorageConfig) IsSetUp() bool {
	return fscfg.StorageFilePath != ""
}

type DatabaseConfig struct {
	DBConnString string `env:"DATABASE_DSN"`
}

func (dbcfg *DatabaseConfig) IsSetUp() bool {
	return dbcfg.DBConnString != ""
}

type PostgresRetryConfig struct {
	MaxRetryCount          int `env:"PG_MAX_RETRY_COUNT" envDefault:"3"`
	InitialRetryDelaySec   int `env:"PG_INITIAL_RETRY_DELAY" envDefault:"1"`
	RetryDelayIncrementSec int `env:"PG_RETRY_DELAY_INCREMENT" envDefault:"2"`
}

type SecurityConfig struct {
	SecretKey string `env:"KEY"`
}

type AuditConfig struct {
	AuditFilePath string `env:"AUDIT_FILE"`
	AuditURL      string `env:"AUDIT_URL"`
}

type Config struct {
	FileStorageConfig
	DatabaseConfig
	PostgresRetryConfig
	SecurityConfig
	AuditConfig

	LogLevel       string `env:"SERVER_LOG_LEVEL" default:"info"`
	Addr           string `env:"ADDRESS"`
	GracePeriodSec int    `env:"SERVER_GRACE_PERIOD"`
}

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

	flag.StringVar(&cfg.AuditFilePath, "audit-file", "",
		"audit file path (should be specified to enable file audit)")
	flag.StringVar(&cfg.AuditURL, "audit-url", "",
		"audit url (should be specified to enable http service audit)")
}
