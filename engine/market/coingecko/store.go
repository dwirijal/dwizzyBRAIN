package coingecko

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) UpsertCoins(ctx context.Context, coins []MarketCoin) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("postgres pool is required")
	}
	if len(coins) == 0 {
		return 0, nil
	}

	const query = `
INSERT INTO coins (
    id, symbol, name, image_url, rank, market_cap_rank, is_active, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, TRUE, NOW())
ON CONFLICT (id)
DO UPDATE SET
    symbol = EXCLUDED.symbol,
    name = EXCLUDED.name,
    image_url = EXCLUDED.image_url,
    rank = EXCLUDED.rank,
    market_cap_rank = EXCLUDED.market_cap_rank,
    is_active = TRUE,
    updated_at = NOW()`

	batch := &pgx.Batch{}
	for _, coin := range coins {
		rank := firstRank(coin.MarketCapRank, coin.MarketCapRankWithRehypothecated)
		batch.Queue(
			query,
			strings.TrimSpace(coin.ID),
			strings.ToLower(strings.TrimSpace(coin.Symbol)),
			strings.TrimSpace(coin.Name),
			strings.TrimSpace(coin.Image),
			rank,
			rank,
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range coins {
		if _, err := results.Exec(); err != nil {
			return 0, fmt.Errorf("upsert coin: %w", err)
		}
	}

	return len(coins), nil
}

func (s *Store) UpsertColdCoinData(ctx context.Context, coins []MarketCoin) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("postgres pool is required")
	}
	if len(coins) == 0 {
		return 0, nil
	}

	const query = `
INSERT INTO cold_coin_data (
    coin_id,
    ath,
    atl,
    ath_date,
    atl_date,
    market_cap_rank,
    current_price_usd,
    market_cap_usd,
    total_volume_24h,
    price_change_24h,
    market_cap_change_24h,
    updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW())
ON CONFLICT (coin_id)
DO UPDATE SET
    ath = EXCLUDED.ath,
    atl = EXCLUDED.atl,
    ath_date = EXCLUDED.ath_date,
    atl_date = EXCLUDED.atl_date,
    market_cap_rank = EXCLUDED.market_cap_rank,
    current_price_usd = EXCLUDED.current_price_usd,
    market_cap_usd = EXCLUDED.market_cap_usd,
    total_volume_24h = EXCLUDED.total_volume_24h,
    price_change_24h = EXCLUDED.price_change_24h,
    market_cap_change_24h = EXCLUDED.market_cap_change_24h,
    updated_at = NOW()`

	batch := &pgx.Batch{}
	for _, coin := range coins {
		rank := firstRank(coin.MarketCapRank, coin.MarketCapRankWithRehypothecated)
		batch.Queue(
			query,
			strings.TrimSpace(coin.ID),
			coin.ATH,
			coin.ATL,
			coin.ATHDate,
			coin.ATLDate,
			rank,
			coin.CurrentPrice,
			coin.MarketCap,
			coin.TotalVolume,
			coin.PriceChange24h,
			coin.MarketCapChange24h,
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range coins {
		if _, err := results.Exec(); err != nil {
			return 0, fmt.Errorf("upsert cold coin data: %w", err)
		}
	}

	return len(coins), nil
}

func firstRank(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func (s *Store) CountCoins(ctx context.Context) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("postgres pool is required")
	}

	var count int
	if err := s.db.QueryRow(ctx, `SELECT count(*) FROM coins`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count coins: %w", err)
	}
	return count, nil
}

func (s *Store) LatestColdCoinIDs(ctx context.Context, limit int) ([]string, error) {
	if s.db == nil {
		return nil, fmt.Errorf("postgres pool is required")
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.Query(ctx, `SELECT coin_id FROM cold_coin_data ORDER BY updated_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("query cold coin ids: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan cold coin id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cold coin ids: %w", err)
	}

	return ids, nil
}
