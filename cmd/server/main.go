package main

import (
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/dump"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"go.uber.org/zap"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
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

	// channel used to trigger data storage and passing exit flag
	storeTriggerChan := make(chan bool, 1)
	go func() {
		for {
			isExit := <-storeTriggerChan
			mdumper.Store()
			if isExit {
				// graceful shutdown (not implemented yet)
				os.Exit(0)
			}
		}
	}()

	// subscribing on shutdown events
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for {
			<-exitChan
			storeTriggerChan <- true
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
				storeTriggerChan <- false
			}
		}()
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
