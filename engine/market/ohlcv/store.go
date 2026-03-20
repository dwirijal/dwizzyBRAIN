package ohlcv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TimescaleStore struct {
	db *pgxpool.Pool
}

func NewTimescaleStore(db *pgxpool.Pool) *TimescaleStore {
	return &TimescaleStore{db: db}
}

func (s *TimescaleStore) UpsertCandles(ctx context.Context, candles []Candle) error {
	if s.db == nil {
		return fmt.Errorf("timescale pool is required")
	}
	if len(candles) == 0 {
		return nil
	}

	const query = `
INSERT INTO ohlcv (
    time, symbol_id, interval, open, high, low, close, volume, trades, coin_id, exchange, symbol, timeframe, quote_volume, is_closed
 ) VALUES ($1, $2, $3::varchar, $4, $5, $6, $7, $8, NULLIF($9, 0), $10, $11, $12, $3::varchar, NULLIF($13, 0), $14)
ON CONFLICT (time, coin_id, exchange, timeframe)
DO UPDATE SET
    interval = EXCLUDED.interval,
    symbol = EXCLUDED.symbol,
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    quote_volume = EXCLUDED.quote_volume,
    trades = EXCLUDED.trades,
    is_closed = EXCLUDED.is_closed`

	batch := &pgx.Batch{}
	for _, candle := range candles {
		symbolID, err := s.ensureSymbolID(ctx, candle)
		if err != nil {
			return err
		}

		batch.Queue(
			query,
			candle.Timestamp.UTC(),
			symbolID,
			normalizeTimeframe(candle.Timeframe),
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
			candle.Trades,
			strings.TrimSpace(candle.CoinID),
			normalizeExchange(candle.Exchange),
			strings.TrimSpace(candle.Symbol),
			candle.QuoteVolume,
			candle.IsClosed,
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range candles {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert ohlcv candle: %w", err)
		}
	}

	return nil
}

func (s *TimescaleStore) GetCandles(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]Candle, error) {
	if s.db == nil {
		return nil, fmt.Errorf("timescale pool is required")
	}
	if limit <= 0 {
		limit = 200
	}

	const query = `
SELECT time, coin_id, exchange, symbol, timeframe, open, high, low, close, volume, COALESCE(quote_volume, 0), COALESCE(trades, 0), is_closed
FROM ohlcv
WHERE coin_id = $1
  AND exchange = $2
  AND timeframe = $3
ORDER BY time DESC
LIMIT $4`

	rows, err := s.db.Query(ctx, query, strings.TrimSpace(coinID), normalizeExchange(exchange), normalizeTimeframe(timeframe), limit)
	if err != nil {
		return nil, fmt.Errorf("query ohlcv: %w", err)
	}
	defer rows.Close()

	candles := make([]Candle, 0)
	for rows.Next() {
		var candle Candle
		if err := rows.Scan(
			&candle.Timestamp,
			&candle.CoinID,
			&candle.Exchange,
			&candle.Symbol,
			&candle.Timeframe,
			&candle.Open,
			&candle.High,
			&candle.Low,
			&candle.Close,
			&candle.Volume,
			&candle.QuoteVolume,
			&candle.Trades,
			&candle.IsClosed,
		); err != nil {
			return nil, fmt.Errorf("scan ohlcv candle: %w", err)
		}
		candles = append(candles, candle)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate ohlcv candles: %w", err)
	}

	return candles, nil
}

func (s *TimescaleStore) LatestTimestamp(ctx context.Context, coinID, exchange, timeframe string) (time.Time, error) {
	if s.db == nil {
		return time.Time{}, fmt.Errorf("timescale pool is required")
	}

	const query = `
SELECT time
FROM ohlcv
WHERE coin_id = $1
  AND exchange = $2
  AND timeframe = $3
ORDER BY time DESC
LIMIT 1`

	var ts time.Time
	if err := s.db.QueryRow(ctx, query, strings.TrimSpace(coinID), normalizeExchange(exchange), normalizeTimeframe(timeframe)).Scan(&ts); err != nil {
		if err == pgx.ErrNoRows {
			return time.Time{}, nil
		}
		return time.Time{}, fmt.Errorf("latest ohlcv time: %w", err)
	}

	return ts.UTC(), nil
}

func (s *TimescaleStore) GetCandleAtOrBefore(ctx context.Context, coinID, exchange, timeframe string, at time.Time) (Candle, error) {
	if s.db == nil {
		return Candle{}, fmt.Errorf("timescale pool is required")
	}

	const query = `
SELECT time, coin_id, exchange, symbol, timeframe, open, high, low, close, volume, COALESCE(quote_volume, 0), COALESCE(trades, 0), is_closed
FROM ohlcv
WHERE coin_id = $1
  AND exchange = $2
  AND timeframe = $3
  AND time <= $4
ORDER BY time DESC
LIMIT 1`

	var candle Candle
	if err := s.db.QueryRow(ctx, query, strings.TrimSpace(coinID), normalizeExchange(exchange), normalizeTimeframe(timeframe), at.UTC()).Scan(
		&candle.Timestamp,
		&candle.CoinID,
		&candle.Exchange,
		&candle.Symbol,
		&candle.Timeframe,
		&candle.Open,
		&candle.High,
		&candle.Low,
		&candle.Close,
		&candle.Volume,
		&candle.QuoteVolume,
		&candle.Trades,
		&candle.IsClosed,
	); err != nil {
		if err == pgx.ErrNoRows {
			return Candle{}, err
		}
		return Candle{}, fmt.Errorf("query ohlcv candle at or before: %w", err)
	}

	return candle, nil
}

func (s *TimescaleStore) ensureSymbolID(ctx context.Context, candle Candle) (string, error) {
	const query = `
INSERT INTO symbols (exchange, symbol, base_currency, quote_currency, active, metadata, updated_at)
VALUES ($1, $2, $3, $4, TRUE, '{}'::jsonb, NOW())
ON CONFLICT (exchange, symbol)
DO UPDATE SET
    base_currency = EXCLUDED.base_currency,
    quote_currency = EXCLUDED.quote_currency,
    active = TRUE,
    updated_at = NOW()
RETURNING id`

	base, quote := splitSymbol(candle.Symbol)
	var symbolID string
	if err := s.db.QueryRow(
		ctx,
		query,
		normalizeExchange(candle.Exchange),
		strings.TrimSpace(candle.Symbol),
		base,
		quote,
	).Scan(&symbolID); err != nil {
		return "", fmt.Errorf("ensure symbol id for %s:%s: %w", candle.Exchange, candle.Symbol, err)
	}

	return symbolID, nil
}

func splitSymbol(symbol string) (string, string) {
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return "", ""
	}
	if strings.Contains(symbol, "/") {
		parts := strings.SplitN(symbol, "/", 2)
		return strings.ToUpper(strings.TrimSpace(parts[0])), strings.ToUpper(strings.TrimSpace(parts[1]))
	}
	for _, quote := range []string{"USDT", "USDC", "BUSD", "BTC", "ETH"} {
		if strings.HasSuffix(strings.ToUpper(symbol), quote) && len(symbol) > len(quote) {
			base := symbol[:len(symbol)-len(quote)]
			return strings.ToUpper(strings.TrimSpace(base)), quote
		}
	}
	return strings.ToUpper(symbol), ""
}
