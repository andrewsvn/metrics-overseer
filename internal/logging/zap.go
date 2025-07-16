package logging

import (
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewZapLogger(logLevel string) (*zap.Logger, error) {
	lvl, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return nil, fmt.Errorf("error parsing log level: %w", err)
	}

	lcfg := zap.NewProductionConfig()
	lcfg.DisableCaller = true
	lcfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	lcfg.Level = lvl

	logger, err := lcfg.Build()
	if err != nil {
		return nil, fmt.Errorf("error creating logger: %w", err)
	}

	return logger, nil
}
