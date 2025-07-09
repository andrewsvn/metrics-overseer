package main

import (
	"log"

	"github.com/andrewsvn/metrics-overseer/internal/agent"
	"github.com/andrewsvn/metrics-overseer/internal/config/agentcfg"
)

func main() {
	cfg := agentcfg.Read()
	a, err := agent.NewAgent(cfg)
	if err != nil {
		log.Fatalf("Can't initialize agent: %v", err)
		return
	}
	a.Run()
}
