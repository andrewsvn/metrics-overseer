package server

import (
	"context"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/db"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/andrewsvn/metrics-overseer/migrations"
	"go.uber.org/zap"
	"time"
)

func InitializeStorage(cfg *servercfg.Config, logger *zap.Logger) (repository.Storage, error) {
	if cfg.DatabaseConfig.IsSetUp() {
		dbcfg := &cfg.DatabaseConfig

		logger.Info("migrating database schema")
		err := migrations.MigrateDB(dbcfg, logger)
		if err != nil {
			return nil, err
		}

		logger.Info("creating postgres connection pool")
		dbconn, err := db.NewPostgresDB(context.Background(), dbcfg)
		if err != nil {
			return nil, fmt.Errorf("can't create postgres database connection pool: %w", err)
		}

		logger.Info("initializing postgres storage")
		policy := retrying.NewLinearPolicy(
			cfg.MaxRetryCount,
			time.Duration(cfg.InitialRetryDelaySec)*time.Second,
			time.Duration(cfg.RetryDelayIncrementSec)*time.Second,
		)
		return repository.NewPostgresDBStorage(dbconn, logger, policy), nil
	}

	if cfg.FileStorageConfig.IsSetUp() {
		logger.Info("initializing file storage")
		return repository.NewFileStorage(&cfg.FileStorageConfig, logger), nil
	}

	logger.Info("initializing memory storage")
	return repository.NewMemStorage(), nil
}
