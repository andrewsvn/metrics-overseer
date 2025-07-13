package main

import (
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
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
	cfg := servercfg.Read()
	logger, err := logging.NewZapLogger(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("can't initialize logger: %w", err)
	}

	mstor := repository.NewMemStorage()
	msrv := service.NewMetricsService(mstor)
	mhandlers := handler.NewMetricsHandlers(msrv, logger)

	r := mhandlers.GetRouter()

	logger.Info(fmt.Sprintf("Starting server on %s\n", cfg.Addr))
	return http.ListenAndServe(cfg.Addr, r)
}
