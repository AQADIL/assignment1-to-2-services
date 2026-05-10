package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"order-service/internal/domain"
)

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr string) (*RedisCache, error) {
	opt, err := redis.ParseURL(addr)
	if err != nil {
		return nil, err
	}
	client := redis.NewClient(opt)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RedisCache{client: client}, nil
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) Get(ctx context.Context, id string) (domain.Order, bool, error) {
	val, err := c.client.Get(ctx, orderKey(id)).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return domain.Order{}, false, nil
		}
		return domain.Order{}, false, err
	}
	var order domain.Order
	if err := json.Unmarshal(val, &order); err != nil {
		return domain.Order{}, false, err
	}
	return order, true, nil
}

func (c *RedisCache) Set(ctx context.Context, order domain.Order, ttl time.Duration) error {
	b, err := json.Marshal(order)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, orderKey(order.ID), b, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, id string) error {
	return c.client.Del(ctx, orderKey(id)).Err()
}

func orderKey(id string) string {
	return "order:" + id
}
