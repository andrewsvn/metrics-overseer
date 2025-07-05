package servercfg

import (
	"flag"
	"fmt"
)

const (
	defaultAddr = ":8080"
)

type Config struct {
	Addr string
}

func DefaultConfig() *Config {
	return &Config{
		Addr: defaultAddr,
	}
}

func ReadFromCLArgs() *Config {
	cfg := &Config{}
	cfg.bindFlags()
	flag.Parse()
	return cfg
}

func (cfg *Config) bindFlags() {
	flag.StringVar(&cfg.Addr, "a", defaultAddr, fmt.Sprintf("server address in form of host:port (default: %s)", defaultAddr))
}
