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

func (c *Config) BindFlags() {
	flag.StringVar(&c.Addr, "a", defaultAddr, fmt.Sprintf("server address in form of host:port (default: %s)", defaultAddr))
}
