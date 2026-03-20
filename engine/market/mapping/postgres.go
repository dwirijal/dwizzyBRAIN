package mapping

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type queryRowScanner interface {
	Scan(dest ...any) error
}

type dbQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

type PostgresStore struct {
	db dbQuerier
}

type CoinRecord struct {
	CoinID string
	Symbol string
}

type UnknownSymbol struct {
	Exchange  string
	RawSymbol string
	BaseAsset string
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{db: pool}
}

func (s *PostgresStore) GetPrimaryMapping(ctx context.Context, coinID, exchange string) (Mapping, error) {
	const query = `
SELECT coin_id, exchange, exchange_symbol, base_asset, quote_asset, is_primary, COALESCE(verified_at, TIMESTAMPTZ 'epoch')
FROM coin_exchange_mappings
WHERE coin_id = $1
  AND exchange = $2
  AND status = 'active'
ORDER BY is_primary DESC, verified_at DESC NULLS LAST, updated_at DESC
LIMIT 1`

	return scanMapping(s.db.QueryRow(ctx, query, strings.TrimSpace(coinID), normalizeExchange(exchange)))
}

func (s *PostgresStore) GetMappingBySymbol(ctx context.Context, exchange, symbol string) (Mapping, error) {
	const query = `
SELECT coin_id, exchange, exchange_symbol, base_asset, quote_asset, is_primary, COALESCE(verified_at, TIMESTAMPTZ 'epoch')
FROM coin_exchange_mappings
WHERE exchange = $1
  AND UPPER(exchange_symbol) = $2
  AND status = 'active'
LIMIT 1`

	return scanMapping(s.db.QueryRow(ctx, query, normalizeExchange(exchange), normalizeSymbol(symbol)))
}

func (s *PostgresStore) RecordUnknownSymbol(ctx context.Context, exchange, rawSymbol, baseAsset string) error {
	const query = `
INSERT INTO unknown_symbols (exchange, raw_symbol, base_asset)
VALUES ($1, $2, NULLIF($3, ''))
ON CONFLICT (exchange, raw_symbol)
DO UPDATE SET
    base_asset = COALESCE(NULLIF(EXCLUDED.base_asset, ''), unknown_symbols.base_asset),
    last_seen_at = NOW(),
    seen_count = unknown_symbols.seen_count + 1,
    updated_at = NOW()`

	if _, err := s.db.Exec(ctx, query, normalizeExchange(exchange), normalizeSymbol(rawSymbol), strings.TrimSpace(baseAsset)); err != nil {
		return fmt.Errorf("record unknown symbol %s:%s: %w", exchange, rawSymbol, err)
	}

	return nil
}

func (s *PostgresStore) ListCoins(ctx context.Context) ([]CoinRecord, error) {
	const query = `
SELECT COALESCE(NULLIF(coin_id, ''), id) AS coin_id, UPPER(symbol) AS symbol
FROM coins
WHERE COALESCE(symbol, '') <> ''`

	rows, err := s.query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list coins: %w", err)
	}
	defer rows.Close()

	coins := make([]CoinRecord, 0)
	for rows.Next() {
		var coin CoinRecord
		if err := rows.Scan(&coin.CoinID, &coin.Symbol); err != nil {
			return nil, fmt.Errorf("scan coin: %w", err)
		}
		coins = append(coins, coin)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate coins: %w", err)
	}

	return coins, nil
}

func (s *PostgresStore) UpsertMapping(ctx context.Context, mapping Mapping, status string) error {
	const query = `
INSERT INTO coin_exchange_mappings (
    coin_id, exchange, exchange_symbol, base_asset, quote_asset, status, is_primary, verified_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6::exchange_symbol_status, $7, $8, NOW())
ON CONFLICT (exchange, exchange_symbol)
DO UPDATE SET
    coin_id = EXCLUDED.coin_id,
    base_asset = EXCLUDED.base_asset,
    quote_asset = EXCLUDED.quote_asset,
    status = EXCLUDED.status,
    is_primary = EXCLUDED.is_primary,
    verified_at = EXCLUDED.verified_at,
    updated_at = NOW()`

	verifiedAt := mapping.VerifiedAt
	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}

	_, err := s.db.Exec(
		ctx,
		query,
		strings.TrimSpace(mapping.CoinID),
		normalizeExchange(mapping.Exchange),
		normalizeSymbol(mapping.ExchangeSymbol),
		strings.ToUpper(strings.TrimSpace(mapping.BaseAsset)),
		strings.ToUpper(strings.TrimSpace(mapping.QuoteAsset)),
		normalizeMappingStatus(status),
		mapping.IsPrimary,
		verifiedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert mapping %s:%s: %w", mapping.Exchange, mapping.ExchangeSymbol, err)
	}

	return nil
}

func (s *PostgresStore) ListMappingsByExchange(ctx context.Context, exchange string) ([]Mapping, error) {
	const query = `
SELECT coin_id, exchange, exchange_symbol, base_asset, quote_asset, is_primary, COALESCE(verified_at, TIMESTAMPTZ 'epoch')
FROM coin_exchange_mappings
WHERE exchange = $1`

	rows, err := s.query(ctx, query, normalizeExchange(exchange))
	if err != nil {
		return nil, fmt.Errorf("list mappings for %s: %w", exchange, err)
	}
	defer rows.Close()

	mappings := make([]Mapping, 0)
	for rows.Next() {
		mapping, err := scanMapping(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mappings for %s: %w", exchange, err)
	}

	return mappings, nil
}

func (s *PostgresStore) ListMappingsByCoin(ctx context.Context, coinID string) ([]Mapping, error) {
	const query = `
SELECT coin_id, exchange, exchange_symbol, base_asset, quote_asset, is_primary, COALESCE(verified_at, TIMESTAMPTZ 'epoch')
FROM coin_exchange_mappings
WHERE coin_id = $1
  AND status = 'active'
ORDER BY is_primary DESC, verified_at DESC NULLS LAST, exchange ASC, exchange_symbol ASC`

	rows, err := s.query(ctx, query, strings.TrimSpace(coinID))
	if err != nil {
		return nil, fmt.Errorf("list mappings for coin %s: %w", coinID, err)
	}
	defer rows.Close()

	mappings := make([]Mapping, 0)
	for rows.Next() {
		mapping, err := scanMapping(rows)
		if err != nil {
			return nil, fmt.Errorf("scan mapping: %w", err)
		}
		mappings = append(mappings, mapping)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mappings for coin %s: %w", coinID, err)
	}

	return mappings, nil
}

func (s *PostgresStore) SetMappingStatus(ctx context.Context, exchange, symbol, status string, verifiedAt time.Time) error {
	const query = `
UPDATE coin_exchange_mappings
SET status = $3::exchange_symbol_status,
    verified_at = $4,
    updated_at = NOW()
WHERE exchange = $1
  AND UPPER(exchange_symbol) = $2`

	if verifiedAt.IsZero() {
		verifiedAt = time.Now().UTC()
	}

	if _, err := s.db.Exec(ctx, query, normalizeExchange(exchange), normalizeSymbol(symbol), normalizeMappingStatus(status), verifiedAt); err != nil {
		return fmt.Errorf("set mapping status %s:%s: %w", exchange, symbol, err)
	}

	return nil
}

func (s *PostgresStore) ListPendingUnknownSymbols(ctx context.Context, limit int) ([]UnknownSymbol, error) {
	if limit <= 0 {
		limit = 100
	}

	const query = `
SELECT exchange, raw_symbol, COALESCE(base_asset, '')
FROM unknown_symbols
WHERE status = 'pending'
ORDER BY seen_count DESC, first_seen_at ASC
LIMIT $1`

	rows, err := s.query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending unknown symbols: %w", err)
	}
	defer rows.Close()

	symbols := make([]UnknownSymbol, 0)
	for rows.Next() {
		var item UnknownSymbol
		if err := rows.Scan(&item.Exchange, &item.RawSymbol, &item.BaseAsset); err != nil {
			return nil, fmt.Errorf("scan unknown symbol: %w", err)
		}
		symbols = append(symbols, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate unknown symbols: %w", err)
	}

	return symbols, nil
}

func (s *PostgresStore) UpdateUnknownSymbol(ctx context.Context, exchange, rawSymbol, status, resolvedCoinID, notes string) error {
	const query = `
UPDATE unknown_symbols
SET status = $3::unknown_symbol_status,
    resolved_coin_id = NULLIF($4, ''),
    resolve_notes = NULLIF($5, ''),
    updated_at = NOW()
WHERE exchange = $1
  AND raw_symbol = $2`

	if _, err := s.db.Exec(
		ctx,
		query,
		normalizeExchange(exchange),
		normalizeSymbol(rawSymbol),
		normalizeUnknownStatus(status),
		strings.TrimSpace(resolvedCoinID),
		strings.TrimSpace(notes),
	); err != nil {
		return fmt.Errorf("update unknown symbol %s:%s: %w", exchange, rawSymbol, err)
	}

	return nil
}

func (s *PostgresStore) query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	type queryer interface {
		Query(context.Context, string, ...any) (pgx.Rows, error)
	}

	q, ok := s.db.(queryer)
	if !ok {
		return nil, fmt.Errorf("db does not support query")
	}

	return q.Query(ctx, sql, args...)
}

func normalizeMappingStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if slices.Contains([]string{"active", "delisted", "not_listed", "dex_only", "unknown"}, status) {
		return status
	}
	return "active"
}

func normalizeUnknownStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if slices.Contains([]string{"pending", "resolved", "unresolvable", "ignored"}, status) {
		return status
	}
	return "pending"
}

func scanMapping(row queryRowScanner) (Mapping, error) {
	var mapping Mapping
	err := row.Scan(
		&mapping.CoinID,
		&mapping.Exchange,
		&mapping.ExchangeSymbol,
		&mapping.BaseAsset,
		&mapping.QuoteAsset,
		&mapping.IsPrimary,
		&mapping.VerifiedAt,
	)
	if err == nil {
		return mapping, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Mapping{}, ErrMappingNotFound
	}
	return Mapping{}, err
}
