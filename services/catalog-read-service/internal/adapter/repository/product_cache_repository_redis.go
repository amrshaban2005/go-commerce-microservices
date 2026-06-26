package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/domain"
	"github.com/amrshaban2005/go-commerce-microservices/services/catalog-read-service/internal/port"
	"github.com/redis/go-redis/v9"
)

const productsCacheKey = "catalog-read:products:all"

type productCacheRepositoryRedis struct {
	client *redis.Client
	ttl    time.Duration
}

func NewProductCacheRepositoryRedis(client *redis.Client) port.ProductCacheRepository {
	return &productCacheRepositoryRedis{
		client: client,
		ttl:    5 * time.Minute,
	}
}

func (r *productCacheRepositoryRedis) GetProducts(ctx context.Context) ([]domain.Product, error) {
	value, err := r.client.Get(ctx, productsCacheKey).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var products []domain.Product
	if err := json.Unmarshal(value, &products); err != nil {
		return nil, err
	}

	return products, nil
}

func (r *productCacheRepositoryRedis) SetProducts(ctx context.Context, products []domain.Product) error {
	value, err := json.Marshal(products)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, productsCacheKey, value, r.ttl).Err()
}

func (r *productCacheRepositoryRedis) DeleteProducts(ctx context.Context) error {
	return r.client.Del(ctx, productsCacheKey).Err()
}