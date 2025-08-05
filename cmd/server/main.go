package main

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/db"
	"github.com/andrewsvn/metrics-overseer/internal/handler"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/andrewsvn/metrics-overseer/migrations"
	"go.uber.org/zap"
	"log"
	"net/http"
	"strings"
	"time"
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

	dbconn, err := initializeDB(&cfg.DatabaseConfig, logger)
	if err != nil {
		return err
	}
	// no need to close here since postgres repository will handle this

	stor := initializeStorage(cfg, dbconn, logger)
	defer func() {
		err := stor.Close()
		if err != nil {
			logger.Error("failed to close storage", zap.Error(err))
		}
	}()

	msrv := service.NewMetricsService(stor)
	mhandlers := handler.NewMetricsHandlers(msrv, dbconn, logger)
	r := mhandlers.GetRouter()

	addr := strings.Trim(cfg.Addr, "\"")
	logger.Sugar().Infow("starting server",
		"address", addr,
	)
	return http.ListenAndServe(addr, r)
}

func initializeStorage(cfg *servercfg.Config, dbconn *db.PostgresDB, logger *zap.Logger) repository.Storage {
	if dbconn != nil {
		logger.Info("initializing postgres storage")
		policy := retrying.NewLinearPolicy(
			cfg.MaxRetryCount,
			time.Duration(cfg.InitialRetryDelaySec)*time.Second,
			time.Duration(cfg.RetryDelayIncrementSec)*time.Second,
		)
		return repository.NewPostgresDBStorage(dbconn.Pool(), logger, policy)
	}

	if cfg.FileStorageConfig.IsSetUp() {
		logger.Info("initializing file storage")
		return repository.NewFileStorage(&cfg.FileStorageConfig, logger)
	}

	logger.Info("initializing memory storage")
	return repository.NewMemStorage()
}

func initializeDB(dbcfg *servercfg.DatabaseConfig, logger *zap.Logger) (*db.PostgresDB, error) {
	if !dbcfg.IsSetUp() {
		return nil, nil
	}

	err := migrations.MigrateDB(dbcfg, logger)
	if err != nil {
		return nil, err
	}

	dbconn, err := db.NewPostgresDB(context.Background(), dbcfg)
	if err != nil {
		return nil, fmt.Errorf("can't create postgres database connection pool: %w", err)
	}

	logger.Sugar().Infow("initialized postgres database connection pool",
		"DSN", dbcfg.DBConnString)
	return dbconn, nil
}
