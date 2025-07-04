package main

import (
	"flag"

	"github.com/andrewsvn/metrics-overseer/internal/agent"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
)

func main() {
	cfg := readConfig()
	agent.NewAgent(cfg).Run()
}

func readConfig() *agentcfg.Config {
	cfg := &agentcfg.Config{}
	cfg.BindFlags()

	flag.Parse()
	return cfg
}
