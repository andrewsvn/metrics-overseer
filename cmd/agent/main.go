package main

import (
	"log"

	"github.com/andrewsvn/metrics-overseer/internal/buildinfo"
	"github.com/andrewsvn/metrics-overseer/internal/logging"

	"github.com/andrewsvn/metrics-overseer/internal/agent"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

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
