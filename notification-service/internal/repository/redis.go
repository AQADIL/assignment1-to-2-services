package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisStore(addr string, ttl time.Duration) (*RedisStore, error) {
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
	return &RedisStore{client: client, ttl: ttl}, nil
}

func (s *RedisStore) Close() error {
	return s.client.Close()
}

func (s *RedisStore) IsProcessed(ctx context.Context, eventID string) (bool, error) {
	exists, err := s.client.Exists(ctx, idempotencyKey(eventID)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	return exists > 0, nil
}

func (s *RedisStore) MarkProcessed(ctx context.Context, eventID string) error {
	return s.client.Set(ctx, idempotencyKey(eventID), 1, s.ttl).Err()
}

func idempotencyKey(eventID string) string {
	return "notif:processed:" + eventID
}
