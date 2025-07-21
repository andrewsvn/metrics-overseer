package main

import (
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"log"

	"github.com/andrewsvn/metrics-overseer/internal/agent"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
)

func main() {
	cfg, err := agentcfg.Read()
	if err != nil {
		log.Fatalf("Can't read agent configuration: %v", err)
		return
	}

	logger, err := logging.NewZapLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("can't initialize logger: %v", err)
	}

	a, err := agent.NewAgent(cfg, logger)
	if err != nil {
		log.Fatalf("Can't initialize agent: %v", err)
		return
	}

	a.Run()
}
