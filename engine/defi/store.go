package defi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

func (s *Store) LookupCoinIDBySymbol(ctx context.Context, symbol string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("postgres pool is required")
	}
	symbol = strings.TrimSpace(symbol)
	if symbol == "" {
		return "", nil
	}

	const query = `
SELECT coin_id
FROM coins
WHERE LOWER(COALESCE(NULLIF(symbol, ''), coin_id)) = LOWER($1)
   OR LOWER(COALESCE(NULLIF(coin_id, ''), symbol)) = LOWER($1)
LIMIT 1`

	var coinID string
	if err := s.db.QueryRow(ctx, query, symbol).Scan(&coinID); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup coin id for symbol %s: %w", symbol, err)
	}
	return coinID, nil
}

func (s *Store) UpsertProtocols(ctx context.Context, items []ProtocolUpsert) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(items) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_protocols (
    slug, name, description, logo, category, url, twitter, coin_id, audit_status, oracles, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    logo = EXCLUDED.logo,
    category = EXCLUDED.category,
    url = EXCLUDED.url,
    twitter = EXCLUDED.twitter,
    coin_id = COALESCE(EXCLUDED.coin_id, defi_protocols.coin_id),
    audit_status = EXCLUDED.audit_status,
    oracles = EXCLUDED.oracles,
    updated_at = EXCLUDED.updated_at`

	batch := &pgx.Batch{}
	for _, item := range items {
		oracles := item.Oracles
		if oracles == nil {
			oracles = []string{}
		}
		batch.Queue(
			query,
			item.Slug,
			item.Name,
			item.Description,
			item.Logo,
			item.Category,
			item.URL,
			item.Twitter,
			nullString(item.CoinID),
			item.AuditStatus,
			oracles,
			item.UpdatedAt.UTC(),
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range items {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert defi protocol: %w", err)
		}
	}
	return nil
}

func (s *Store) UpsertProtocolCoverage(ctx context.Context, items []ProtocolCoverage) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(items) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_protocol_coverage (slug, tier, updated_at)
VALUES ($1, $2, $3)
ON CONFLICT (slug) DO UPDATE SET
    tier = EXCLUDED.tier,
    updated_at = EXCLUDED.updated_at`

	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(query, item.Slug, item.Tier, item.UpdatedAt.UTC())
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range items {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert defi protocol coverage: %w", err)
		}
	}
	return nil
}

func (s *Store) UpsertChains(ctx context.Context, items []ChainUpsert) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(items) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_chain_tvl_latest (chain, tvl, change_1d, change_7d, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (chain) DO UPDATE SET
    tvl = EXCLUDED.tvl,
    change_1d = EXCLUDED.change_1d,
    change_7d = EXCLUDED.change_7d,
    updated_at = EXCLUDED.updated_at`

	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(query, item.Chain, item.TVLUSD, nullFloat(item.Change1D), nullFloat(item.Change7D), item.UpdatedAt.UTC())
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range items {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert defi chain: %w", err)
		}
	}
	return nil
}

func (s *Store) UpsertProtocolLatest(ctx context.Context, items []ProtocolLatest) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(items) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_protocol_tvl_latest (slug, tvl, change_1d, change_7d, updated_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (slug) DO UPDATE SET
    tvl = EXCLUDED.tvl,
    change_1d = EXCLUDED.change_1d,
    change_7d = EXCLUDED.change_7d,
    updated_at = EXCLUDED.updated_at`

	batch := &pgx.Batch{}
	for _, item := range items {
		batch.Queue(query, item.Slug, item.TVLUSD, nullFloat(item.Change1D), nullFloat(item.Change7D), item.UpdatedAt.UTC())
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range items {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert defi protocol tvl: %w", err)
		}
	}
	return nil
}

func (s *Store) InsertProtocolHistory(ctx context.Context, records []ProtocolHistoryRecord) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(records) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_protocol_tvl_history (time, protocol_slug, tvl_usd, chain, metadata)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT DO NOTHING`

	batch := &pgx.Batch{}
	for _, record := range records {
		metadata := record.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}
		payload, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal defi history metadata: %w", err)
		}
		batch.Queue(query, record.Time.UTC(), record.ProtocolSlug, record.TVLUSD, record.Chain, payload)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range records {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("insert defi protocol history: %w", err)
		}
	}
	return nil
}

func (s *Store) InsertChainHistory(ctx context.Context, records []ChainHistoryRecord) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(records) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_chain_tvl_history (time, chain, tvl_usd)
VALUES ($1, $2, $3)
ON CONFLICT DO NOTHING`

	batch := &pgx.Batch{}
	for _, record := range records {
		batch.Queue(query, record.Time.UTC(), record.Chain, record.TVLUSD)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range records {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("insert defi chain history: %w", err)
		}
	}
	return nil
}

type ProtocolUpsert struct {
	Slug        string
	Name        string
	Description string
	Logo        string
	Category    string
	URL         string
	Twitter     string
	CoinID      string
	AuditStatus string
	Oracles     []string
	UpdatedAt   time.Time
}

type ProtocolCoverage struct {
	Slug      string
	Tier      string
	UpdatedAt time.Time
}

type ProtocolLatest struct {
	Slug      string
	TVLUSD    float64
	Change1D  *float64
	Change7D  *float64
	UpdatedAt time.Time
}

type ChainUpsert struct {
	Chain     string
	TVLUSD    float64
	Change1D  *float64
	Change7D  *float64
	UpdatedAt time.Time
}

func nullString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func nullFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
