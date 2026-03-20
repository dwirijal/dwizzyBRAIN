package storage

import (
	"context"
	"fmt"
	"os"
	"strings"

	redis "github.com/redis/go-redis/v9"
)

func NewValkeyClientFromEnv(ctx context.Context) (*redis.Client, error) {
	rawURL := strings.TrimSpace(os.Getenv("VALKEY_URL"))
	if rawURL == "" {
		return nil, fmt.Errorf("VALKEY_URL is required")
	}

	return NewValkeyClient(ctx, rawURL)
}

func NewValkeyClient(ctx context.Context, rawURL string) (*redis.Client, error) {
	options, err := redis.ParseURL(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse VALKEY_URL: %w", err)
	}

	client := redis.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping valkey: %w", err)
	}

	return client, nil
}
