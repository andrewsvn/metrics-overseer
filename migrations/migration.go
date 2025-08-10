package migrations

import (
	"embed"
	"errors"
	"fmt"
	"github.com/andrewsvn/metrics-overseer/internal/config/servercfg"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"

	_ "github.com/golang-migrate/migrate/v4/database/postgres"
)

//go:embed *.sql
var migrationFS embed.FS

func MigrateDB(cfg *servercfg.DatabaseConfig, logger *zap.Logger) error {
	fs, err := iofs.New(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("can't find migration files: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", fs, cfg.DBConnString)
	if err != nil {
		return fmt.Errorf("can't initialize database migration: %w", err)
	}

	err = m.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			logger.Sugar().Info("database schema is up to date")
		} else {
			return err
		}
	}
	return nil
}
