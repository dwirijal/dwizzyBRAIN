package yields

import (
	"context"
	"encoding/json"
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

func (s *Store) LookupProtocolSlugByProject(ctx context.Context, project string) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("postgres pool is required")
	}
	project = strings.TrimSpace(project)
	if project == "" {
		return "", nil
	}

	const query = `
SELECT slug
FROM defi_protocols
WHERE LOWER(slug) = LOWER($1)
   OR LOWER(name) = LOWER($1)
   OR LOWER(REPLACE(name, ' ', '-')) = LOWER($1)
LIMIT 1`

	var slug string
	if err := s.db.QueryRow(ctx, query, project).Scan(&slug); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("lookup protocol slug for project %s: %w", project, err)
	}
	return slug, nil
}

func (s *Store) UpsertLatest(ctx context.Context, items []LatestRecord) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(items) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_yield_latest (
    pool, chain, project, symbol, protocol_slug,
    tvl_usd, apy, apy_base, apy_reward, apy_pct_1d, apy_pct_7d, apy_pct_30d,
    apy_mean_30d, volume_usd_1d, volume_usd_7d, stablecoin, il_risk, exposure,
    reward_tokens, underlying_tokens, predictions, pool_meta, outlier, count,
    updated_at, synced_at
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8, $9, $10, $11, $12,
    $13, $14, $15, $16, $17, $18,
    $19, $20, $21, $22, $23, $24,
    $25, $26
) ON CONFLICT (pool) DO UPDATE SET
    chain = EXCLUDED.chain,
    project = EXCLUDED.project,
    symbol = EXCLUDED.symbol,
    protocol_slug = EXCLUDED.protocol_slug,
    tvl_usd = EXCLUDED.tvl_usd,
    apy = EXCLUDED.apy,
    apy_base = EXCLUDED.apy_base,
    apy_reward = EXCLUDED.apy_reward,
    apy_pct_1d = EXCLUDED.apy_pct_1d,
    apy_pct_7d = EXCLUDED.apy_pct_7d,
    apy_pct_30d = EXCLUDED.apy_pct_30d,
    apy_mean_30d = EXCLUDED.apy_mean_30d,
    volume_usd_1d = EXCLUDED.volume_usd_1d,
    volume_usd_7d = EXCLUDED.volume_usd_7d,
    stablecoin = EXCLUDED.stablecoin,
    il_risk = EXCLUDED.il_risk,
    exposure = EXCLUDED.exposure,
    reward_tokens = EXCLUDED.reward_tokens,
    underlying_tokens = EXCLUDED.underlying_tokens,
    predictions = EXCLUDED.predictions,
    pool_meta = EXCLUDED.pool_meta,
    outlier = EXCLUDED.outlier,
    count = EXCLUDED.count,
    updated_at = EXCLUDED.updated_at,
    synced_at = EXCLUDED.synced_at`

	batch := &pgx.Batch{}
	for _, item := range items {
		rewardTokens := item.RewardTokens
		if rewardTokens == nil {
			rewardTokens = []string{}
		}
		underlyingTokens := item.UnderlyingTokens
		if underlyingTokens == nil {
			underlyingTokens = []string{}
		}
		predictions := item.Predictions
		if predictions == nil {
			predictions = map[string]any{}
		}
		poolMeta, err := json.Marshal(item.PoolMeta)
		if err != nil {
			return fmt.Errorf("marshal pool meta: %w", err)
		}
		predictionsJSON, err := json.Marshal(predictions)
		if err != nil {
			return fmt.Errorf("marshal predictions: %w", err)
		}
		batch.Queue(
			query,
			item.Pool,
			item.Chain,
			item.Project,
			nullableString(item.Symbol),
			nullableString(item.ProtocolSlug),
			item.TVLUSD,
			item.APY,
			item.APYBase,
			item.APYReward,
			item.APYPct1D,
			item.APYPct7D,
			item.APYPct30D,
			item.APYMean30D,
			item.VolumeUsd1D,
			item.VolumeUsd7D,
			item.Stablecoin,
			nullableString(item.ILRisk),
			nullableString(item.Exposure),
			rewardTokens,
			underlyingTokens,
			predictionsJSON,
			poolMeta,
			item.Outlier,
			item.Count,
			item.UpdatedAt.UTC(),
			item.SyncedAt.UTC(),
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range items {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert yield latest: %w", err)
		}
	}
	return nil
}

func (s *Store) InsertHistory(ctx context.Context, records []HistoryRecord) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(records) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_yield_history (
    time, pool, chain, project, symbol, tvl_usd, apy, apy_base, apy_reward, metadata
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT DO NOTHING`

	batch := &pgx.Batch{}
	for _, record := range records {
		metadata := record.Metadata
		if metadata == nil {
			metadata = map[string]any{}
		}
		payload, err := json.Marshal(metadata)
		if err != nil {
			return fmt.Errorf("marshal yield metadata: %w", err)
		}
		batch.Queue(query,
			record.Time.UTC(),
			record.Pool,
			record.Chain,
			record.Project,
			nullableString(record.Symbol),
			record.TVLUSD,
			record.APY,
			record.APYBase,
			record.APYReward,
			payload,
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range records {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("insert yield history: %w", err)
		}
	}
	return nil
}

func nullableString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
