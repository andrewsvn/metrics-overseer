package dbcfg

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	DBConnString string `env:"DATABASE_DSN"`
}

func Read() (*Config, error) {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	err := env.Parse(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.DBConnString == "" {
		return nil, fmt.Errorf("DB connection string is not set")
	}
	return cfg, nil
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.DBConnString, "d", "",
		"postgres database connection string")
}
