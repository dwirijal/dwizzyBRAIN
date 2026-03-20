package coverage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStore struct {
	db *pgxpool.Pool
}

func NewPostgresStore(db *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: db}
}

func (s *PostgresStore) ListCoins(ctx context.Context) ([]Coin, error) {
	if s.db == nil {
		return nil, fmt.Errorf("postgres pool is required")
	}

	const query = `
SELECT COALESCE(NULLIF(coin_id, ''), id) AS coin_id,
       COALESCE(market_cap_rank, rank, 0) AS rank
FROM coins
WHERE COALESCE(is_active, TRUE) = TRUE`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query coins: %w", err)
	}
	defer rows.Close()

	coins := make([]Coin, 0)
	for rows.Next() {
		var coin Coin
		if err := rows.Scan(&coin.CoinID, &coin.Rank); err != nil {
			return nil, fmt.Errorf("scan coin: %w", err)
		}
		coins = append(coins, coin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate coins: %w", err)
	}

	return coins, nil
}

func (s *PostgresStore) ListExchangeCoverage(ctx context.Context) (map[string]map[string]time.Time, error) {
	if s.db == nil {
		return nil, fmt.Errorf("postgres pool is required")
	}

	const query = `
SELECT coin_id, exchange, COALESCE(verified_at, NOW())
FROM coin_exchange_mappings
WHERE status = 'active'`

	rows, err := s.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query active mappings: %w", err)
	}
	defer rows.Close()

	result := make(map[string]map[string]time.Time)
	for rows.Next() {
		var coinID, exchange string
		var verifiedAt time.Time
		if err := rows.Scan(&coinID, &exchange, &verifiedAt); err != nil {
			return nil, fmt.Errorf("scan active mapping: %w", err)
		}
		coinID = strings.TrimSpace(coinID)
		exchange = strings.ToLower(strings.TrimSpace(exchange))
		if result[coinID] == nil {
			result[coinID] = make(map[string]time.Time)
		}
		result[coinID][exchange] = verifiedAt.UTC()
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active mappings: %w", err)
	}

	return result, nil
}

func (s *PostgresStore) UpsertCoverage(ctx context.Context, coverage Coverage) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}

	const query = `
INSERT INTO coin_coverage (
    coin_id, tier, on_binance, on_bybit, on_okx, on_kucoin, on_gate, on_kraken, on_mexc, on_htx,
    on_coinpaprika, is_dex_only, binance_verified_at, bybit_verified_at, assigned_at, updated_at
) VALUES (
    $1, $2::coverage_tier, $3, $4, $5, $6, $7, $8, $9, $10,
    $11, $12, $13, $14, $15, $16
)
ON CONFLICT (coin_id)
DO UPDATE SET
    tier = EXCLUDED.tier,
    on_binance = EXCLUDED.on_binance,
    on_bybit = EXCLUDED.on_bybit,
    on_okx = EXCLUDED.on_okx,
    on_kucoin = EXCLUDED.on_kucoin,
    on_gate = EXCLUDED.on_gate,
    on_kraken = EXCLUDED.on_kraken,
    on_mexc = EXCLUDED.on_mexc,
    on_htx = EXCLUDED.on_htx,
    on_coinpaprika = EXCLUDED.on_coinpaprika,
    is_dex_only = EXCLUDED.is_dex_only,
    binance_verified_at = EXCLUDED.binance_verified_at,
    bybit_verified_at = EXCLUDED.bybit_verified_at,
    updated_at = EXCLUDED.updated_at`

	_, err := s.db.Exec(
		ctx,
		query,
		strings.TrimSpace(coverage.CoinID),
		coverage.Tier,
		coverage.OnBinance,
		coverage.OnBybit,
		coverage.OnOKX,
		coverage.OnKucoin,
		coverage.OnGate,
		coverage.OnKraken,
		coverage.OnMexc,
		coverage.OnHtx,
		coverage.OnCoinpaprika,
		coverage.IsDexOnly,
		coverage.BinanceVerifiedAt,
		coverage.BybitVerifiedAt,
		coverage.AssignedAt.UTC(),
		coverage.UpdatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert coin coverage: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetCoverage(ctx context.Context, coinID string) (Coverage, error) {
	if s.db == nil {
		return Coverage{}, fmt.Errorf("postgres pool is required")
	}

	const query = `
SELECT coin_id, tier, on_binance, on_bybit, on_okx, on_kucoin, on_gate, on_kraken, on_mexc, on_htx,
       on_coinpaprika, is_dex_only, binance_verified_at, bybit_verified_at, assigned_at, updated_at
FROM coin_coverage
WHERE coin_id = $1`

	var coverage Coverage
	err := s.db.QueryRow(ctx, query, strings.TrimSpace(coinID)).Scan(
		&coverage.CoinID,
		&coverage.Tier,
		&coverage.OnBinance,
		&coverage.OnBybit,
		&coverage.OnOKX,
		&coverage.OnKucoin,
		&coverage.OnGate,
		&coverage.OnKraken,
		&coverage.OnMexc,
		&coverage.OnHtx,
		&coverage.OnCoinpaprika,
		&coverage.IsDexOnly,
		&coverage.BinanceVerifiedAt,
		&coverage.BybitVerifiedAt,
		&coverage.AssignedAt,
		&coverage.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return Coverage{}, fmt.Errorf("coverage not found")
		}
		return Coverage{}, fmt.Errorf("query coverage: %w", err)
	}

	return coverage, nil
}
