package stablecoins

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

func (s *Store) LookupCoinID(ctx context.Context, asset Asset) (string, error) {
	if s.db == nil {
		return "", fmt.Errorf("postgres pool is required")
	}
	identifiers := []string{
		strings.TrimSpace(asset.GeckoID),
		strings.TrimSpace(asset.Symbol),
		strings.TrimSpace(asset.Name),
	}
	for _, identifier := range identifiers {
		if identifier == "" {
			continue
		}
		const query = `
SELECT coin_id
FROM coins
WHERE LOWER(coin_id) = LOWER($1)
   OR LOWER(symbol) = LOWER($1)
   OR LOWER(name) = LOWER($1)
LIMIT 1`
		var coinID string
		if err := s.db.QueryRow(ctx, query, identifier).Scan(&coinID); err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return "", fmt.Errorf("lookup coin id for %s: %w", identifier, err)
		}
		return coinID, nil
	}
	return "", nil
}

func (s *Store) UpsertLatest(ctx context.Context, items []LatestRecord) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if len(items) == 0 {
		return nil
	}

	const query = `
INSERT INTO defi_stablecoin_backing (
    coin_id, snapshot_date, peg_type, peg_mechanism, price_usd, mcap_usd,
    circulating, backing_composition, attestation_url, attested_at, synced_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
ON CONFLICT (coin_id, snapshot_date) DO UPDATE SET
    peg_type = EXCLUDED.peg_type,
    peg_mechanism = EXCLUDED.peg_mechanism,
    price_usd = EXCLUDED.price_usd,
    mcap_usd = EXCLUDED.mcap_usd,
    circulating = EXCLUDED.circulating,
    backing_composition = EXCLUDED.backing_composition,
    attestation_url = EXCLUDED.attestation_url,
    attested_at = EXCLUDED.attested_at,
    synced_at = EXCLUDED.synced_at`

	batch := &pgx.Batch{}
	for _, item := range items {
		backing, err := json.Marshal(item.BackingComposition)
		if err != nil {
			return fmt.Errorf("marshal backing composition: %w", err)
		}
		batch.Queue(
			query,
			item.CoinID,
			item.SnapshotDate.UTC().Format("2006-01-02"),
			item.PegType,
			item.PegMechanism,
			item.PriceUSD,
			item.MCAPUSD,
			item.Circulating,
			backing,
			item.AttestationURL,
			item.AttestedAt,
			item.SyncedAt.UTC(),
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range items {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("upsert stablecoin latest: %w", err)
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
INSERT INTO defi_stable_mcap_history (time, coin_id, mcap_usd, circulating, price_usd)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT DO NOTHING`

	batch := &pgx.Batch{}
	for _, record := range records {
		batch.Queue(query, record.Time.UTC(), record.CoinID, record.MCAPUSD, record.Circulating, record.PriceUSD)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range records {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("insert stablecoin history: %w", err)
		}
	}
	return nil
}
