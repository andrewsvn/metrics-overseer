package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/andrewsvn/metrics-overseer/internal/audit"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/andrewsvn/metrics-overseer/internal/db"
	"github.com/andrewsvn/metrics-overseer/internal/handling/grpcsrv"
	"github.com/andrewsvn/metrics-overseer/internal/handling/restsrv"
	"github.com/andrewsvn/metrics-overseer/internal/logging"
	"github.com/andrewsvn/metrics-overseer/internal/repository"
	"github.com/andrewsvn/metrics-overseer/internal/retrying"
	"github.com/andrewsvn/metrics-overseer/internal/service"
	"github.com/andrewsvn/metrics-overseer/migrations"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	_ "net/http/pprof"
)

func Run() error {
	cfg, err := servercfg.Read()
	if err != nil {
		return fmt.Errorf("can't read server config: %w", err)
	}

	logger, err := logging.NewZapLogger(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("can't initialize logger: %w", err)
	}
	sl := logger.Sugar()

	stor, err := InitializeStorage(cfg, logger)
	if err != nil {
		return fmt.Errorf("can't initialize storage: %w", err)
	}
	defer func() {
		err := stor.Close()
		if err != nil {
			sl.Errorw("Failed to close metrics storage", "error", err)
		}
	}()

	msrv := service.NewMetricsService(stor, logger)
	var fw *audit.FileWriter
	if cfg.AuditFilePath != "" {
		sl.Infow("subscribing file auditor", "path", cfg.AuditFilePath)
		fw = audit.NewFileWriter(cfg.AuditFilePath, cfg.AuditFileWriteIntervalSec, logger)
		msrv.SubscribeAuditor(fw)
	}
	if cfg.AuditURL != "" {
		sl.Infow("subscribing http service auditor", "url", cfg.AuditURL)
		msrv.SubscribeAuditor(audit.NewHTTPWriter(cfg.AuditURL))
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	// metrics rest server
	mhandlers, err := restsrv.NewMetricsHandlers(msrv, &cfg.SecurityConfig, logger)
	if err != nil {
		return fmt.Errorf("can't initialize metrics handlers: %w", err)
	}

	r := mhandlers.GetRouter()
	addr := strings.Trim(cfg.Addr, "\"")
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Sugar().Infow("starting metric-overseer server",
			"address", addr,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	// metrics grpc server
	var gsrv *grpc.Server
	if cfg.GRPCAddr != "" {
		listen, err := net.Listen("tcp", ":3200")
		if err != nil {
			log.Fatal(err)
		}
		metricsGsrv, err := grpcsrv.NewMetricsServer(msrv, &cfg.SecurityConfig, logger)
		if err != nil {
			return fmt.Errorf("can't initialize gRPC metrics server: %w", err)
		}

		gsrv = metricsGsrv.GetGRPCServer()
		go func() {
			logger.Sugar().Infow("starting gRPC metrics server", "address", cfg.GRPCAddr)
			if err := gsrv.Serve(listen); err != nil {
				logger.Fatal("failed to start gRPC metrics server", zap.Error(err))
			}
		}()
	}

	// pprof server
	if cfg.PprofAddr != "" {
		go func() {
			logger.Sugar().Infow("starting pprof handlers",
				"address", cfg.PprofAddr)
			if err := http.ListenAndServe(cfg.PprofAddr, nil); err != nil {
				logger.Fatal("failed to start pprof", zap.Error(err))
			}
		}()
	}

	<-ctx.Done()

	logger.Info("shutting down metric-overseer server...")
	ctx, cancel = context.WithTimeout(context.Background(), time.Duration(cfg.GracePeriodSec)*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown server: %w", err)
	}

	if gsrv != nil {
		gsrv.GracefulStop()
	}

	err = stor.Close()
	if err != nil {
		logger.Error("failed to close storage", zap.Error(err))
	}

	if fw != nil {
		fw.Close()
	}

	logger.Info("metric-overseer server shutdown complete")
	return nil
}

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
