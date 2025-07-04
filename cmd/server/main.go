package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
)

func main() {
	log.Fatal(run())
}

func run() error {
	cfg := readConfig()

	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)
	mhandlers := handler.NewMetricsHandlers(msrv)

	r := mhandlers.GetRouter()

	log.Printf("Starting server on %s\n", cfg.Addr)
	return http.ListenAndServe(cfg.Addr, r)
}

func readConfig() *servercfg.Config {
	cfg := &servercfg.Config{}
	cfg.BindFlags()

	flag.Parse()
	return cfg
}
