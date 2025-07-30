package dbcfg

import (
	"flag"
	"github.com/caarlos0/env/v6"
)

const (
	defaultPostgresConnString = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
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

	return cfg, nil
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.DBConnString, "d", defaultPostgresConnString,
		"postgres database connection string")
}
