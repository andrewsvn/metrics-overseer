package main

import (
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/dump"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"go.uber.org/zap"
	"log"
	"net/http"
	"strings"
	"time"

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
	mdumper := dump.NewStorageDumper(cfg.StorageFilePath, mstor, logger)

	if cfg.RestoreOnStartup {
		logger.Info("Restoring metrics on startup")
		mdumper.Load()
	}

	if cfg.StoreIntervalSec > 0 {
		storeInterval := time.Duration(cfg.StoreIntervalSec) * time.Second
		storeTicker := time.NewTicker(storeInterval)
		logger.Info("Scheduling metrics storing to file", zap.Duration("interval", storeInterval))
		go func(tc <-chan time.Time) {
			for {
				<-tc
				mdumper.Store()
			}
		}(storeTicker.C)
	}

	msrv := service.NewMetricsService(mstor)
	if cfg.StoreIntervalSec == 0 {
		msrv.AttachDumper(mdumper)
	}

	mhandlers := handler.NewMetricsHandlers(msrv, logger)
	r := mhandlers.GetRouter()

	addr := strings.Trim(cfg.Addr, "\"")
	logger.Info(fmt.Sprintf("Starting server on %s", addr))
	return http.ListenAndServe(addr, r)
}
