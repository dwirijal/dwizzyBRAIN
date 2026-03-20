package ticker

import (
	"testing"
	"time"

	"dwizzyBRAIN/shared/schema"
)

func TestAggregatorUpdateBuildsSnapshot(t *testing.T) {
	agg := NewAggregator()
	now := time.Date(2026, 3, 18, 22, 0, 0, 0, time.UTC)
	agg.now = func() time.Time { return now }

	first, err := agg.Update(schema.ResolvedTicker{
		CoinID: "bitcoin", Symbol: "BTC/USDT", Exchange: "binance",
		BaseAsset: "BTC", QuoteAsset: "USDT",
		Price: 71290, Bid: 71289, Ask: 71291, Volume: 10,
		Timestamp: now.Add(-2 * time.Second),
	})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}
	if first.ExchangeCount != 1 {
		t.Fatalf("expected 1 exchange, got %d", first.ExchangeCount)
	}

	second, err := agg.Update(schema.ResolvedTicker{
		CoinID: "bitcoin", Symbol: "BTC/USDT", Exchange: "kraken",
		BaseAsset: "BTC", QuoteAsset: "USDT",
		Price: 71295, Bid: 71294, Ask: 71296, Volume: 5,
		Timestamp: now.Add(-1 * time.Second),
	})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	if second.BestBid != 71294 || second.BestBidExchange != "kraken" {
		t.Fatalf("unexpected best bid snapshot: %+v", second)
	}
	if second.BestAsk != 71291 || second.BestAskExchange != "binance" {
		t.Fatalf("unexpected best ask snapshot: %+v", second)
	}
	if second.ExchangeCount != 2 {
		t.Fatalf("expected 2 exchanges, got %d", second.ExchangeCount)
	}
	if second.CrossExchangeSpreadPct <= 0 {
		t.Fatalf("expected positive cross-exchange spread, got %.6f", second.CrossExchangeSpreadPct)
	}
}

func TestAggregatorIgnoresOutOfOrderTicker(t *testing.T) {
	agg := NewAggregator()
	now := time.Date(2026, 3, 18, 22, 0, 0, 0, time.UTC)
	agg.now = func() time.Time { return now }

	if _, err := agg.Update(schema.ResolvedTicker{
		CoinID: "bitcoin", Symbol: "BTC/USDT", Exchange: "binance",
		BaseAsset: "BTC", QuoteAsset: "USDT",
		Price: 71300, Bid: 71299, Ask: 71301,
		Timestamp: now,
	}); err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	snapshot, err := agg.Update(schema.ResolvedTicker{
		CoinID: "bitcoin", Symbol: "BTC/USDT", Exchange: "binance",
		BaseAsset: "BTC", QuoteAsset: "USDT",
		Price: 70000, Bid: 69999, Ask: 70001,
		Timestamp: now.Add(-time.Minute),
	})
	if err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}
	if snapshot.AvailableExchanges[0].Price != 71300 {
		t.Fatalf("expected newer price to win, got %.2f", snapshot.AvailableExchanges[0].Price)
	}
}

func TestAggregatorMarksStaleExchange(t *testing.T) {
	agg := NewAggregator()
	now := time.Date(2026, 3, 18, 22, 0, 30, 0, time.UTC)
	agg.now = func() time.Time { return now }
	agg.maxAge = 10 * time.Second

	if _, err := agg.Update(schema.ResolvedTicker{
		CoinID: "bitcoin", Symbol: "BTC/USDT", Exchange: "binance",
		BaseAsset: "BTC", QuoteAsset: "USDT",
		Price: 71300, Bid: 71299, Ask: 71301,
		Timestamp: now.Add(-20 * time.Second),
	}); err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}

	snapshot, ok := agg.Snapshot("bitcoin")
	if !ok {
		t.Fatal("expected snapshot to exist")
	}
	if !snapshot.AvailableExchanges[0].IsStale {
		t.Fatal("expected exchange to be marked stale")
	}
	if snapshot.BestBid != 0 || snapshot.BestAsk != 0 {
		t.Fatalf("expected stale ticker to be excluded from best bid/ask, got %+v", snapshot)
	}
}
