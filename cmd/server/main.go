package main

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/dbcfg"
	"github.com/andrewsvn/metrics-overseer/internal/db"
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
	cfg, err := servercfg.Read()
	if err != nil {
		return fmt.Errorf("can't read server config: %w", err)
	}

	logger, err := logging.NewZapLogger(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("can't initialize logger: %w", err)
	}

	mstor := repository.NewMemStorage()
	mdumper := dump.NewStorageDumper(cfg.StorageFilePath, mstor, logger)

	if cfg.RestoreOnStartup {
		err := mdumper.Load()
		if err != nil {
			logger.Error("failed to load metrics on startup", zap.Error(err))
		}
	}

	// store on shutdown - not implemented yet
	defer func() {
		err := mdumper.Store()
		if err != nil {
			logger.Error("failed to store metrics on shutdown", zap.Error(err))
		}
	}()

	if cfg.StoreIntervalSec > 0 {
		// subscribing on store timer
		storeInterval := time.Duration(cfg.StoreIntervalSec) * time.Second
		storeTicker := time.NewTicker(storeInterval)
		logger.Info("Scheduling metrics storing to file", zap.Duration("interval", storeInterval))
		go func() {
			for {
				<-storeTicker.C
				err := mdumper.Store()
				if err != nil {
					logger.Error("failed to store metrics", zap.Error(err))
				}
			}
		}()
	}

	dbconf, err := dbcfg.Read()
	if err != nil {
		return fmt.Errorf("can't read database config: %w", err)
	}

	dbconn, err := db.NewPostgresDB(context.Background(), dbconf)
	if err != nil {
		return fmt.Errorf("can't create postgres database connection pool: %w", err)
	}
	defer dbconn.Close()

	msrv := service.NewMetricsService(mstor)
	if cfg.StoreIntervalSec == 0 {
		msrv.AttachDumper(mdumper)
	}

	mhandlers := handler.NewMetricsHandlers(msrv, dbconn, logger)
	r := mhandlers.GetRouter()

	addr := strings.Trim(cfg.Addr, "\"")
	logger.Sugar().Infow("starting server",
		"address", addr,
	)
	return http.ListenAndServe(addr, r)
}
