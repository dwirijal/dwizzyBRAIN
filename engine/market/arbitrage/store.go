package arbitrage

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SignalStore struct {
	db *pgxpool.Pool
}

func NewSignalStore(db *pgxpool.Pool) *SignalStore {
	return &SignalStore{db: db}
}

func (s *SignalStore) Insert(ctx context.Context, opportunity Opportunity) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}

	const query = `
INSERT INTO arbitrage_signals (
    detected_at, coin_id, symbol, buy_exchange, sell_exchange, buy_price, sell_price,
    gross_spread_pct, buy_depth_usd, sell_depth_usd, alerted
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, FALSE
)`

	_, err := s.db.Exec(
		ctx,
		query,
		opportunity.DetectedAt.UTC(),
		strings.TrimSpace(opportunity.CoinID),
		strings.TrimSpace(opportunity.Symbol),
		strings.ToLower(strings.TrimSpace(opportunity.BuyExchange)),
		strings.ToLower(strings.TrimSpace(opportunity.SellExchange)),
		opportunity.BuyPrice,
		opportunity.SellPrice,
		opportunity.GrossSpreadPct,
		opportunity.BuyDepthUSD,
		opportunity.SellDepthUSD,
	)
	if err != nil {
		return fmt.Errorf("insert arbitrage signal: %w", err)
	}

	return nil
}
