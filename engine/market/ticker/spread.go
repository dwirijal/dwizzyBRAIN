package ticker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SpreadRecord struct {
	Timestamp time.Time `json:"timestamp"`
	CoinID    string    `json:"coin_id"`
	Exchange  string    `json:"exchange"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	SpreadPct float64   `json:"spread_pct"`
	Volume24h float64   `json:"volume_24h"`
}

type spreadSnapshotSource interface {
	Snapshots() []Snapshot
}

type SpreadRecorder struct {
	source spreadSnapshotSource
	store  *SpreadStore
	now    func() time.Time
}

func NewSpreadRecorder(source spreadSnapshotSource, store *SpreadStore) *SpreadRecorder {
	return &SpreadRecorder{
		source: source,
		store:  store,
		now:    time.Now,
	}
}

func (r *SpreadRecorder) Collect() []SpreadRecord {
	if r.source == nil {
		return nil
	}

	now := r.now().UTC()
	snapshots := r.source.Snapshots()
	records := make([]SpreadRecord, 0)
	for _, snapshot := range snapshots {
		for _, exchange := range snapshot.AvailableExchanges {
			if exchange.IsStale || exchange.Bid <= 0 || exchange.Ask <= 0 {
				continue
			}
			records = append(records, SpreadRecord{
				Timestamp: now,
				CoinID:    snapshot.CoinID,
				Exchange:  exchange.Exchange,
				Bid:       exchange.Bid,
				Ask:       exchange.Ask,
				SpreadPct: exchange.SpreadPct,
				Volume24h: exchange.Volume,
			})
		}
	}

	return records
}

func (r *SpreadRecorder) Record(ctx context.Context) ([]SpreadRecord, error) {
	if r.store == nil {
		return nil, fmt.Errorf("spread store is required")
	}

	records := r.Collect()
	if len(records) == 0 {
		return nil, nil
	}

	if err := r.store.Insert(ctx, records); err != nil {
		return nil, err
	}

	return records, nil
}

type SpreadStore struct {
	db *pgxpool.Pool
}

func NewSpreadStore(db *pgxpool.Pool) *SpreadStore {
	return &SpreadStore{db: db}
}

func (s *SpreadStore) Insert(ctx context.Context, records []SpreadRecord) error {
	if s.db == nil {
		return fmt.Errorf("timescale pool is required")
	}
	if len(records) == 0 {
		return nil
	}

	const query = `
INSERT INTO exchange_spread_history (time, coin_id, exchange, bid, ask, spread_pct, volume_24h)
VALUES ($1, $2, $3, $4, $5, $6, $7)`

	batch := &pgx.Batch{}
	for _, record := range records {
		batch.Queue(
			query,
			record.Timestamp.UTC(),
			strings.TrimSpace(record.CoinID),
			strings.ToLower(strings.TrimSpace(record.Exchange)),
			record.Bid,
			record.Ask,
			record.SpreadPct,
			record.Volume24h,
		)
	}

	results := s.db.SendBatch(ctx, batch)
	defer results.Close()
	for range records {
		if _, err := results.Exec(); err != nil {
			return fmt.Errorf("insert spread record: %w", err)
		}
	}

	return nil
}

func (s *SpreadStore) Latest(ctx context.Context, coinID string, limit int) ([]SpreadRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("timescale pool is required")
	}
	if limit <= 0 {
		limit = 100
	}

	const query = `
SELECT time, coin_id, exchange, bid, ask, spread_pct, COALESCE(volume_24h, 0)
FROM exchange_spread_history
WHERE coin_id = $1
ORDER BY time DESC
LIMIT $2`

	rows, err := s.db.Query(ctx, query, strings.TrimSpace(coinID), limit)
	if err != nil {
		return nil, fmt.Errorf("query spread history: %w", err)
	}
	defer rows.Close()

	records := make([]SpreadRecord, 0)
	for rows.Next() {
		var record SpreadRecord
		if err := rows.Scan(
			&record.Timestamp,
			&record.CoinID,
			&record.Exchange,
			&record.Bid,
			&record.Ask,
			&record.SpreadPct,
			&record.Volume24h,
		); err != nil {
			return nil, fmt.Errorf("scan spread record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate spread history: %w", err)
	}

	return records, nil
}

func (s *SpreadStore) LatestByExchange(ctx context.Context, coinID string) ([]SpreadRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("timescale pool is required")
	}

	const query = `
SELECT DISTINCT ON (exchange) time, coin_id, exchange, bid, ask, spread_pct, COALESCE(volume_24h, 0)
FROM exchange_spread_history
WHERE coin_id = $1
ORDER BY exchange, time DESC`

	rows, err := s.db.Query(ctx, query, strings.TrimSpace(coinID))
	if err != nil {
		return nil, fmt.Errorf("query latest spread history by exchange: %w", err)
	}
	defer rows.Close()

	records := make([]SpreadRecord, 0)
	for rows.Next() {
		var record SpreadRecord
		if err := rows.Scan(
			&record.Timestamp,
			&record.CoinID,
			&record.Exchange,
			&record.Bid,
			&record.Ask,
			&record.SpreadPct,
			&record.Volume24h,
		); err != nil {
			return nil, fmt.Errorf("scan latest spread record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest spread history: %w", err)
	}

	return records, nil
}
