package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/shared/schema"
)

type stubTickerResolver struct {
	mapping mapping.Mapping
	err     error
	calls   int
}

func (s *stubTickerResolver) ResolveCoinID(ctx context.Context, exchange, rawSymbol string) (mapping.Mapping, error) {
	s.calls++
	if s.err != nil {
		return mapping.Mapping{}, s.err
	}
	return s.mapping, nil
}

type stubResolvedTickerPublisher struct {
	ticker schema.ResolvedTicker
	err    error
	calls  int
}

func (s *stubResolvedTickerPublisher) PublishResolvedTicker(ctx context.Context, ticker schema.ResolvedTicker) error {
	s.calls++
	s.ticker = ticker
	return s.err
}

func TestIngestionServiceResolveTicker(t *testing.T) {
	raw := schema.RawTicker{
		Symbol:    "BTCUSDT",
		Exchange:  "binance",
		Price:     65000,
		Bid:       64999,
		Ask:       65001,
		Volume:    100,
		Timestamp: time.Unix(1710000000, 0).UTC(),
	}

	resolver := &stubTickerResolver{
		mapping: mapping.Mapping{
			CoinID:         "bitcoin",
			Exchange:       "binance",
			ExchangeSymbol: "BTCUSDT",
			BaseAsset:      "BTC",
			QuoteAsset:     "USDT",
			IsPrimary:      true,
		},
	}

	service := NewIngestionService(resolver, nil)
	got, err := service.ResolveTicker(context.Background(), raw)
	if err != nil {
		t.Fatalf("ResolveTicker() returned error: %v", err)
	}
	if got.CoinID != "bitcoin" {
		t.Fatalf("expected bitcoin, got %s", got.CoinID)
	}
	if got.ResolvedSymbol != "BTCUSDT" {
		t.Fatalf("expected resolved symbol BTCUSDT, got %s", got.ResolvedSymbol)
	}
}

func TestIngestionServiceProcessTickerPublishesResolvedTicker(t *testing.T) {
	raw := schema.RawTicker{
		Symbol:    "ETHUSDT",
		Exchange:  "binance",
		Price:     3200,
		Bid:       3199,
		Ask:       3201,
		Volume:    50,
		Timestamp: time.Unix(1710000100, 0).UTC(),
	}

	resolver := &stubTickerResolver{
		mapping: mapping.Mapping{
			CoinID:         "ethereum",
			Exchange:       "binance",
			ExchangeSymbol: "ETHUSDT",
			BaseAsset:      "ETH",
			QuoteAsset:     "USDT",
			IsPrimary:      true,
		},
	}
	publisher := &stubResolvedTickerPublisher{}

	service := NewIngestionService(resolver, publisher)
	got, err := service.ProcessTicker(context.Background(), raw)
	if err != nil {
		t.Fatalf("ProcessTicker() returned error: %v", err)
	}
	if publisher.calls != 1 {
		t.Fatalf("expected publisher to be called once, got %d", publisher.calls)
	}
	if publisher.ticker.CoinID != "ethereum" || got.CoinID != "ethereum" {
		t.Fatalf("expected ethereum to be published, got %s", publisher.ticker.CoinID)
	}
}

func TestIngestionServiceReturnsResolverError(t *testing.T) {
	service := NewIngestionService(&stubTickerResolver{err: errors.New("not found")}, nil)
	_, err := service.ResolveTicker(context.Background(), schema.RawTicker{
		Symbol:    "BADUSDT",
		Exchange:  "binance",
		Timestamp: time.Unix(1710000100, 0).UTC(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
