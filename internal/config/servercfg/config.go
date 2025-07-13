package servercfg

import (
	"flag"
	"fmt"
	"github.com/caarlos0/env/v6"
)

const (
	defaultAddr = ":8080"
)

type Config struct {
	Addr     string `env:"ADDRESS"`
	LogLevel string `env:"SERVER_LOG_LEVEL" default:"info"`
}

func Read() *Config {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	_ = env.Parse(cfg)
	return cfg
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.Addr, "a", defaultAddr,
		fmt.Sprintf("server address in form of host:port (default: %s)", defaultAddr))
}
