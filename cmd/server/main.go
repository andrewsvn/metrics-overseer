package main

import (
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/server"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"go.uber.org/zap"
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

	stor, err := server.InitializeStorage(cfg, logger)
	if err != nil {
		return fmt.Errorf("can't initialize storage: %w", err)
	}
	defer func() {
		err := stor.Close()
		if err != nil {
			logger.Error("failed to close storage", zap.Error(err))
		}
	}()

	msrv := service.NewMetricsService(stor)
	mhandlers := handler.NewMetricsHandlers(msrv, logger)
	r := mhandlers.GetRouter()

	addr := strings.Trim(cfg.Addr, "\"")
	logger.Sugar().Infow("starting server",
		"address", addr,
	)
	return http.ListenAndServe(addr, r)
}
