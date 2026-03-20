package arbitrage

import (
	"context"
	"errors"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/ticker"

	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

type stubSnapshotSource struct {
	snapshots []ticker.Snapshot
}

func (s stubSnapshotSource) Snapshots() []ticker.Snapshot { return s.snapshots }

type stubConfigSource struct {
	config Config
	err    error
}

func (s stubConfigSource) Get(ctx context.Context, coinID string) (Config, error) {
	if s.err != nil {
		return Config{}, s.err
	}
	return s.config, nil
}

type stubSignalStore struct {
	items []Opportunity
	err   error
}

func (s *stubSignalStore) Insert(ctx context.Context, opportunity Opportunity) error {
	if s.err != nil {
		return s.err
	}
	s.items = append(s.items, opportunity)
	return nil
}

func TestEngineScan(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	source := stubSnapshotSource{
		snapshots: []ticker.Snapshot{
			{
				CoinID:                 "bitcoin",
				BestAsk:                100,
				BestAskExchange:        "binance",
				BestBid:                102,
				BestBidExchange:        "kraken",
				CrossExchangeSpreadPct: 2,
				AvailableExchanges: []ticker.ExchangeTicker{
					{Exchange: "binance", Symbol: "BTC/USDT", Ask: 100, Volume: 200},
					{Exchange: "kraken", Symbol: "BTC/USDT", Bid: 102, Volume: 150},
				},
			},
		},
	}
	configs := stubConfigSource{config: Config{
		CoinID:          "bitcoin",
		MinSpreadPct:    0.5,
		MinDepthUSD:     10000,
		CooldownSeconds: 300,
		IsEnabled:       true,
	}}
	store := &stubSignalStore{}

	engine := NewEngine(source, configs, store, client)
	engine.now = func() time.Time { return time.Date(2026, 3, 18, 23, 0, 0, 0, time.UTC) }

	items, err := engine.Scan(context.Background())
	if err != nil {
		t.Fatalf("Scan() returned error: %v", err)
	}
	if len(items) != 1 || len(store.items) != 1 {
		t.Fatalf("expected one opportunity, got %d and %d", len(items), len(store.items))
	}
	if items[0].BuyExchange != "binance" || items[0].SellExchange != "kraken" {
		t.Fatalf("unexpected opportunity: %+v", items[0])
	}
}

func TestEngineRespectsCooldown(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	source := stubSnapshotSource{
		snapshots: []ticker.Snapshot{
			{
				CoinID:                 "bitcoin",
				BestAsk:                100,
				BestAskExchange:        "binance",
				BestBid:                102,
				BestBidExchange:        "kraken",
				CrossExchangeSpreadPct: 2,
				AvailableExchanges: []ticker.ExchangeTicker{
					{Exchange: "binance", Symbol: "BTC/USDT", Ask: 100, Volume: 200},
					{Exchange: "kraken", Symbol: "BTC/USDT", Bid: 102, Volume: 150},
				},
			},
		},
	}
	configs := stubConfigSource{config: Config{CoinID: "bitcoin", MinSpreadPct: 0.5, MinDepthUSD: 10000, CooldownSeconds: 300, IsEnabled: true}}
	store := &stubSignalStore{}
	engine := NewEngine(source, configs, store, client)

	if _, err := engine.Scan(context.Background()); err != nil {
		t.Fatalf("first Scan() returned error: %v", err)
	}
	items, err := engine.Scan(context.Background())
	if err != nil {
		t.Fatalf("second Scan() returned error: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected cooldown to suppress signals, got %d", len(items))
	}
}

func TestEngineReturnsConfigError(t *testing.T) {
	engine := NewEngine(stubSnapshotSource{
		snapshots: []ticker.Snapshot{{CoinID: "bitcoin"}},
	}, stubConfigSource{err: errors.New("boom")}, &stubSignalStore{}, nil)
	if _, err := engine.Scan(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
