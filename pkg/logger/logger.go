package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(options Options, serviceName string) (*zap.Logger, error) {
	environment := os.Getenv("APP_ENV")
	if environment == "" {
		environment = "development"
	}

	var level zapcore.Level
	if err := level.UnmarshalText([]byte(options.Level)); err != nil {
		return nil, fmt.Errorf("parse log level: %w", err)
	}

	var config zap.Config
	if environment == "development" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	config.Level = zap.NewAtomicLevelAt(level)

	log, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("build logger: %w", err)
	}

	return log.With(
		zap.String("service", serviceName),
		zap.String("environment", environment),
	), nil
}
