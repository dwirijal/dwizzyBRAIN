package coverage

import (
	"context"
	"testing"
	"time"
)

type stubStore struct {
	coins     []Coin
	exchanges map[string]map[string]time.Time
	upserts   []Coverage
}

func (s *stubStore) ListCoins(ctx context.Context) ([]Coin, error) {
	return s.coins, nil
}

func (s *stubStore) ListExchangeCoverage(ctx context.Context) (map[string]map[string]time.Time, error) {
	return s.exchanges, nil
}

func (s *stubStore) UpsertCoverage(ctx context.Context, coverage Coverage) error {
	s.upserts = append(s.upserts, coverage)
	return nil
}

func TestGapDetectorDetectAll(t *testing.T) {
	now := time.Date(2026, 3, 18, 22, 50, 0, 0, time.UTC)
	store := &stubStore{
		coins: []Coin{
			{CoinID: "bitcoin", Rank: 1},
			{CoinID: "mid", Rank: 250},
			{CoinID: "tail", Rank: 800},
			{CoinID: "unknown", Rank: 1500},
		},
		exchanges: map[string]map[string]time.Time{
			"bitcoin": {"binance": now, "bybit": now},
			"mid":     {"kraken": now},
		},
	}

	detector := NewGapDetector(store)
	detector.now = func() time.Time { return now }

	result, err := detector.DetectAll(context.Background())
	if err != nil {
		t.Fatalf("DetectAll() returned error: %v", err)
	}
	if result.Processed != 4 {
		t.Fatalf("expected 4 processed coins, got %d", result.Processed)
	}

	got := map[string]string{}
	for _, item := range store.upserts {
		got[item.CoinID] = item.Tier
	}
	if got["bitcoin"] != "A" {
		t.Fatalf("expected bitcoin tier A, got %s", got["bitcoin"])
	}
	if got["mid"] != "B" {
		t.Fatalf("expected mid tier B, got %s", got["mid"])
	}
	if got["tail"] != "C" {
		t.Fatalf("expected tail tier C, got %s", got["tail"])
	}
	if got["unknown"] != "D" {
		t.Fatalf("expected unknown tier D, got %s", got["unknown"])
	}
}
