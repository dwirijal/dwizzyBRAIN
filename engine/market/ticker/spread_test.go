package ticker

import (
	"context"
	"testing"
	"time"

	"dwizzyBRAIN/shared/schema"
)

func TestSpreadRecorderCollect(t *testing.T) {
	agg := NewAggregator()
	now := time.Date(2026, 3, 18, 22, 30, 0, 0, time.UTC)
	agg.now = func() time.Time { return now }

	_, _ = agg.Update(schema.ResolvedTicker{
		CoinID: "bitcoin", Symbol: "BTC/USDT", Exchange: "binance",
		BaseAsset: "BTC", QuoteAsset: "USDT",
		Price: 71290, Bid: 71289, Ask: 71291, Volume: 10,
		Timestamp: now,
	})
	_, _ = agg.Update(schema.ResolvedTicker{
		CoinID: "bitcoin", Symbol: "BTC/USDT", Exchange: "kraken",
		BaseAsset: "BTC", QuoteAsset: "USDT",
		Price: 71295, Bid: 71294, Ask: 71296, Volume: 5,
		Timestamp: now,
	})

	recorder := NewSpreadRecorder(agg, nil)
	recorder.now = func() time.Time { return now }
	records := recorder.Collect()

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].CoinID != "bitcoin" {
		t.Fatalf("expected bitcoin, got %s", records[0].CoinID)
	}
	if records[0].SpreadPct <= 0 {
		t.Fatalf("expected positive intra-exchange spread, got %.6f", records[0].SpreadPct)
	}
}

type stubSpreadSource struct {
	snapshots []Snapshot
}

func (s stubSpreadSource) Snapshots() []Snapshot { return s.snapshots }

type stubInsertStore struct {
	records []SpreadRecord
	err     error
}

func (s *stubInsertStore) Insert(ctx context.Context, records []SpreadRecord) error {
	if s.err != nil {
		return s.err
	}
	s.records = append(s.records, records...)
	return nil
}

func TestSpreadRecorderRecord(t *testing.T) {
	store := &stubInsertStore{}
	source := stubSpreadSource{
		snapshots: []Snapshot{
			{
				CoinID: "bitcoin",
				AvailableExchanges: []ExchangeTicker{
					{Exchange: "binance", Bid: 10, Ask: 11, Volume: 100},
					{Exchange: "kraken", Bid: 9, Ask: 12, Volume: 50, IsStale: true},
				},
			},
		},
	}
	recorder := NewSpreadRecorder(source, nil)
	recorder.now = func() time.Time { return time.Date(2026, 3, 18, 22, 40, 0, 0, time.UTC) }
	recorder.store = &SpreadStore{}

	records := recorder.Collect()
	if len(records) != 1 {
		t.Fatalf("expected 1 non-stale record, got %d", len(records))
	}
	if records[0].Exchange != "binance" {
		t.Fatalf("expected binance, got %s", records[0].Exchange)
	}

	manual := NewSpreadRecorder(source, nil)
	manual.now = recorder.now
	if err := store.Insert(context.Background(), manual.Collect()); err != nil {
		t.Fatalf("Insert() returned error: %v", err)
	}
	if len(store.records) != 1 {
		t.Fatalf("expected 1 inserted record, got %d", len(store.records))
	}
}
