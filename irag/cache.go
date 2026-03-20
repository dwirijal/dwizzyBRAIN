package irag

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	redis "github.com/redis/go-redis/v9"
)

type Cache interface {
	Get(ctx context.Context, key string) (CachedResponse, bool, error)
	Set(ctx context.Context, key string, value CachedResponse, ttl time.Duration) error
}

type CachedResponse struct {
	Status      int            `json:"status"`
	ContentType string         `json:"content_type"`
	Body        []byte         `json:"body"`
	Raw         bool           `json:"raw"`
	Provider    string         `json:"provider"`
	Meta        map[string]any `json:"meta,omitempty"`
}

type RedisCache struct {
	client redis.Cmdable
}

func NewRedisCache(client redis.Cmdable) *RedisCache {
	if client == nil {
		return nil
	}
	return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string) (CachedResponse, bool, error) {
	if c == nil || c.client == nil {
		return CachedResponse{}, false, nil
	}

	raw, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return CachedResponse{}, false, nil
		}
		return CachedResponse{}, false, err
	}

	var cached CachedResponse
	if err := json.Unmarshal(raw, &cached); err != nil {
		return CachedResponse{}, false, err
	}

	return cached, true, nil
}

func (c *RedisCache) Set(ctx context.Context, key string, value CachedResponse, ttl time.Duration) error {
	if c == nil || c.client == nil || ttl <= 0 {
		return nil
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, key, raw, ttl).Err()
}
