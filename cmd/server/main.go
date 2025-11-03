package main

import (
	"log"

	"github.com/andrewsvn/metrics-overseer/internal/buildinfo"
	"github.com/andrewsvn/metrics-overseer/internal/server"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	buildinfo.Print(buildVersion, buildDate, buildCommit)

	err := server.Run()
	if err != nil {
		log.Fatal(err)
	}
}
