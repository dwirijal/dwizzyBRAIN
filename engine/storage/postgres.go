package storage

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPoolFromEnv(ctx context.Context) (*pgxpool.Pool, error) {
	rawURL := strings.TrimSpace(os.Getenv("POSTGRES_URL"))
	if rawURL == "" {
		rawURL = strings.TrimSpace(os.Getenv("NEON_DATABASE_URL"))
	}
	if rawURL == "" {
		return nil, fmt.Errorf("POSTGRES_URL is required (NEON_DATABASE_URL supported as compatibility fallback)")
	}

	return NewPostgresPool(ctx, rawURL)
}

func NewPostgresPool(ctx context.Context, rawURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse POSTGRES_URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return pool, nil
}
