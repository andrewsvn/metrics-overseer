package logging

import (
	"fmt"
	"go.uber.org/zap"
)

func NewZapLogger(logLevel string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return nil, fmt.Errorf("error parsing log level: %w", err)
	}

	lcfg := zap.NewProductionConfig()
	lcfg.Level = lvl

	logger, err := lcfg.Build()
	if err != nil {
		return nil, fmt.Errorf("error creating l: %w", err)
	}

	return logger, nil
}
