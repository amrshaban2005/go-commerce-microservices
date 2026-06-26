package database

import (
	"fmt"

	"github.com/amrshaban2005/go-commerce-microservices/pkg/configloader"
)

type RedisOptions struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func LoadRedisOptions() (*RedisOptions, error) {
	options, err := configloader.BindKey[RedisOptions](
		"redisOptions",
		map[string]string{
			"addr":     "REDIS_ADDR",
			"password": "REDIS_PASSWORD",
			"db":       "REDIS_DB",
		},
	)
	if err != nil {
		return nil, err
	}

	if options.Addr == "" {
		return nil, fmt.Errorf("redisOptions.addr is required")
	}

	return options, nil
}
