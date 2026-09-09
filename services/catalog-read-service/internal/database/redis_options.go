package database

import (
	"fmt"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RedisOptions struct {
	Addr              string        `mapstructure:"addr"`
	Password          string        `mapstructure:"password"`
	DB                int           `mapstructure:"db"`
	ConnectionTimeout time.Duration `mapstructure:"connectionTimeout"`
}

func LoadRedisOptions() (*RedisOptions, error) {
	options, err := configloader.BindKey[RedisOptions](
		"redisOptions",
		map[string]string{
			"addr":              "REDIS_ADDR",
			"password":          "REDIS_PASSWORD",
			"db":                "REDIS_DB",
			"connectionTimeout": "REDIS_CONNECTION_TIMEOUT",
		},
	)
	if err != nil {
		return nil, err
	}

	if options.Addr == "" {
		return nil, fmt.Errorf("redisOptions.addr is required")
	}
	if options.ConnectionTimeout <= 0 {
		return nil, fmt.Errorf("redisOptions.connectionTimeout must be greater than zero")
	}

	return options, nil
}
