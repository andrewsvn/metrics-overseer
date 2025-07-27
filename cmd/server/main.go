package main

import (
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"log"
	"net/http"
	"strings"
)

func main() {
	log.Fatal(run())
}

func run() error {
	cfg, err := servercfg.Read()
	if err != nil {
		return fmt.Errorf("can't read server config: %w", err)
	}

	logger, err := logging.NewZapLogger(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("can't initialize logger: %w", err)
	}

	fstor := repository.NewFileStorage(&cfg.StoreConfig, logger)
	// store on shutdown - not implemented yet
	defer fstor.Close()

	msrv := service.NewMetricsService(fstor)
	mhandlers := handler.NewMetricsHandlers(msrv, logger)
	r := mhandlers.GetRouter()

	addr := strings.Trim(cfg.Addr, "\"")
	logger.Sugar().Infow("starting server",
		"address", addr,
	)
	return http.ListenAndServe(addr, r)
}
