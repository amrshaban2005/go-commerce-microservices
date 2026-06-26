package database

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

func NewRedisClient(
	options *RedisOptions,
	lifecycle fx.Lifecycle,
	logger *zap.Logger,
) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     options.Addr,
		Password: options.Password,
		DB:       options.DB,
	})

	lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("connecting to redis", zap.String("addr", options.Addr))
			return client.Ping(ctx).Err()
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("closing redis connection")
			return client.Close()
		},
	})

	return client
}
