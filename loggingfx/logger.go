// Package loggingfx provides a configured Zap logger and Fx event logger.
package loggingfx

import (
	"context"
	"errors"
	"fmt"
	"syscall"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a production or development Zap logger.
func New(config Config) (*zap.Logger, error) {
	var zapConfig zap.Config
	if config.Development {
		zapConfig = zap.NewDevelopmentConfig()
	} else {
		zapConfig = zap.NewProductionConfig()
	}

	if config.Level != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(config.Level)); err != nil {
			return nil, fmt.Errorf("parse log level %q: %w", config.Level, err)
		}
		zapConfig.Level = zap.NewAtomicLevelAt(level)
	}

	if config.Encoding != "" {
		zapConfig.Encoding = config.Encoding
	}

	if zapConfig.InitialFields == nil {
		zapConfig.InitialFields = make(map[string]any)
	}
	if config.Service != "" {
		zapConfig.InitialFields["service"] = config.Service
	}
	if config.Environment != "" {
		zapConfig.InitialFields["environment"] = config.Environment
	}
	for key, value := range config.Fields {
		zapConfig.InitialFields[key] = value
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return logger, nil
}

// NewFxEventLogger routes Fx's internal events through the application logger.
func NewFxEventLogger(logger *zap.Logger) fxevent.Logger {
	return &fxevent.ZapLogger{Logger: logger}
}

func registerLifecycle(lifecycle fx.Lifecycle, logger *zap.Logger) {
	lifecycle.Append(fx.Hook{
		OnStop: func(context.Context) error {
			if err := logger.Sync(); err != nil && !errors.Is(err, syscall.EINVAL) {
				return fmt.Errorf("sync logger: %w", err)
			}
			return nil
		},
	})
}
