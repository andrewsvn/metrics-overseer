package main

import "github.com/andrewsvn/metrics-overseer/internal/agent"

func main() {
	agent.NewAgent().Start()
}
