package servercfg

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
)

const (
	defaultAddr             = ":8080"
	defaultStoreIntervalSec = 300
	defaultStorageFilePath  = "./metrics.json"
	defaultRestoreOnStartup = false
)

type StoreConfig struct {
	StoreIntervalSec int    `env:"STORE_INTERVAL"`
	StorageFilePath  string `env:"FILE_STORAGE_PATH"`
	RestoreOnStartup bool   `env:"RESTORE"`
}

type Config struct {
	StoreConfig
	LogLevel string `env:"SERVER_LOG_LEVEL" default:"info"`
	Addr     string `env:"ADDRESS"`
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
	flag.IntVar(&cfg.StoreIntervalSec, "i", defaultStoreIntervalSec,
		"metrics storing interval in seconds (0 for synchronous store)")
	flag.StringVar(&cfg.StorageFilePath, "f", defaultStorageFilePath,
		"metrics storage file path")
	flag.BoolVar(&cfg.RestoreOnStartup, "r", defaultRestoreOnStartup,
		"flag for restoring metrics on startup")
}
