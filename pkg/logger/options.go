package logger

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type Options struct {
	Level string `mapstructure:"level"`
}

func LoadOptions() (*Options, error) {
	return configloader.BindKey[Options](
		"logOptions",
		map[string]string{
			"level": "LOG_LEVEL",
		},
	)
}

func (o Options) Validate() error {
	switch o.Level {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("unsupported log level: %s", o.Level)
	}
}
