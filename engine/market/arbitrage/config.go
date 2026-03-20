package arbitrage

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	CoinID          string
	MinSpreadPct    float64
	MinDepthUSD     float64
	CooldownSeconds int
	IsEnabled       bool
}

type ConfigStore struct {
	db *pgxpool.Pool
}

func NewConfigStore(db *pgxpool.Pool) *ConfigStore {
	return &ConfigStore{db: db}
}

func (s *ConfigStore) Get(ctx context.Context, coinID string) (Config, error) {
	if s.db == nil {
		return Config{}, fmt.Errorf("postgres pool is required")
	}

	const query = `
SELECT coin_id, min_spread_pct, min_depth_usd, alert_cooldown_sec, is_enabled
FROM arbitrage_config
WHERE coin_id = $1`

	var cfg Config
	err := s.db.QueryRow(ctx, query, strings.TrimSpace(coinID)).Scan(
		&cfg.CoinID,
		&cfg.MinSpreadPct,
		&cfg.MinDepthUSD,
		&cfg.CooldownSeconds,
		&cfg.IsEnabled,
	)
	if err == nil {
		return cfg, nil
	}
	if err != pgx.ErrNoRows {
		return Config{}, fmt.Errorf("query arbitrage config: %w", err)
	}

	return Config{
		CoinID:          strings.TrimSpace(coinID),
		MinSpreadPct:    0.30,
		MinDepthUSD:     10000,
		CooldownSeconds: 300,
		IsEnabled:       true,
	}, nil
}

func (s *ConfigStore) Upsert(ctx context.Context, cfg Config) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}

	const query = `
INSERT INTO arbitrage_config (coin_id, min_spread_pct, min_depth_usd, alert_cooldown_sec, is_enabled, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW())
ON CONFLICT (coin_id)
DO UPDATE SET
    min_spread_pct = EXCLUDED.min_spread_pct,
    min_depth_usd = EXCLUDED.min_depth_usd,
    alert_cooldown_sec = EXCLUDED.alert_cooldown_sec,
    is_enabled = EXCLUDED.is_enabled,
    updated_at = NOW()`

	_, err := s.db.Exec(ctx, query,
		strings.TrimSpace(cfg.CoinID),
		cfg.MinSpreadPct,
		cfg.MinDepthUSD,
		cfg.CooldownSeconds,
		cfg.IsEnabled,
	)
	if err != nil {
		return fmt.Errorf("upsert arbitrage config: %w", err)
	}

	return nil
}
