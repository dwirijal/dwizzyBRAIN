package storage

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewTimescalePoolFromEnv(ctx context.Context) (*pgxpool.Pool, error) {
	rawURL := strings.TrimSpace(os.Getenv("TIMESCALE_URL"))
	if rawURL == "" {
		return nil, fmt.Errorf("TIMESCALE_URL is required")
	}

	return NewTimescalePool(ctx, rawURL)
}

func NewTimescalePool(ctx context.Context, rawURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("parse TIMESCALE_URL: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create timescale pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping timescale: %w", err)
	}

	return pool, nil
}

func EnsureTimescaleExtension(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("timescale pool is required")
	}

	var version string
	if err := pool.QueryRow(ctx, "select extversion from pg_extension where extname = 'timescaledb'").Scan(&version); err != nil {
		return fmt.Errorf("query timescaledb extension: %w", err)
	}

	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("timescaledb extension is not installed")
	}

	return nil
}
