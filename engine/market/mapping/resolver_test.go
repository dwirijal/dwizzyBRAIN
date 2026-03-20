package mapping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

type stubStore struct {
	primary      map[string]Mapping
	reverse      map[string]Mapping
	unknown      []string
	primaryErr   error
	reverseErr   error
	primaryCalls int
	reverseCalls int
}

func (s *stubStore) GetPrimaryMapping(ctx context.Context, coinID, exchange string) (Mapping, error) {
	s.primaryCalls++
	if s.primaryErr != nil {
		return Mapping{}, s.primaryErr
	}
	value, ok := s.primary[coinID+":"+exchange]
	if !ok {
		return Mapping{}, ErrMappingNotFound
	}
	return value, nil
}

func (s *stubStore) GetMappingBySymbol(ctx context.Context, exchange, symbol string) (Mapping, error) {
	s.reverseCalls++
	if s.reverseErr != nil {
		return Mapping{}, s.reverseErr
	}
	value, ok := s.reverse[exchange+":"+symbol]
	if !ok {
		return Mapping{}, ErrMappingNotFound
	}
	return value, nil
}

func (s *stubStore) RecordUnknownSymbol(ctx context.Context, exchange, rawSymbol, baseAsset string) error {
	s.unknown = append(s.unknown, exchange+":"+rawSymbol+":"+baseAsset)
	return nil
}

func TestResolveExchangeSymbolCachesResult(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	store := &stubStore{
		primary: map[string]Mapping{
			"bitcoin:binance": {
				CoinID:         "bitcoin",
				Exchange:       "binance",
				ExchangeSymbol: "BTCUSDT",
				BaseAsset:      "BTC",
				QuoteAsset:     "USDT",
				IsPrimary:      true,
				VerifiedAt:     time.Now().UTC(),
			},
		},
	}

	resolver := NewSymbolResolver(store, client)
	got, err := resolver.ResolveExchangeSymbol(context.Background(), "bitcoin", "binance")
	if err != nil {
		t.Fatalf("ResolveExchangeSymbol() returned error: %v", err)
	}
	if got.ExchangeSymbol != "BTCUSDT" {
		t.Fatalf("expected BTCUSDT, got %s", got.ExchangeSymbol)
	}

	_, err = resolver.ResolveExchangeSymbol(context.Background(), "bitcoin", "binance")
	if err != nil {
		t.Fatalf("ResolveExchangeSymbol() second call returned error: %v", err)
	}
	if store.primaryCalls != 1 {
		t.Fatalf("expected store to be hit once, got %d", store.primaryCalls)
	}
}

func TestResolveCoinIDCachesReverseLookup(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	store := &stubStore{
		reverse: map[string]Mapping{
			"binance:ETHUSDT": {
				CoinID:         "ethereum",
				Exchange:       "binance",
				ExchangeSymbol: "ETHUSDT",
				BaseAsset:      "ETH",
				QuoteAsset:     "USDT",
				IsPrimary:      true,
			},
		},
	}

	resolver := NewSymbolResolver(store, client)
	got, err := resolver.ResolveCoinID(context.Background(), "binance", "ethusdt")
	if err != nil {
		t.Fatalf("ResolveCoinID() returned error: %v", err)
	}
	if got.CoinID != "ethereum" {
		t.Fatalf("expected ethereum, got %s", got.CoinID)
	}

	_, err = resolver.ResolveCoinID(context.Background(), "binance", "ETHUSDT")
	if err != nil {
		t.Fatalf("ResolveCoinID() second call returned error: %v", err)
	}
	if store.reverseCalls != 1 {
		t.Fatalf("expected reverse store call once, got %d", store.reverseCalls)
	}
}

func TestResolveCoinIDRecordsUnknownSymbol(t *testing.T) {
	store := &stubStore{}
	resolver := NewSymbolResolver(store, nil)

	_, err := resolver.ResolveCoinID(context.Background(), "binance", "NEWTKUSDT")
	if !errors.Is(err, ErrMappingNotFound) {
		t.Fatalf("expected ErrMappingNotFound, got %v", err)
	}
	if len(store.unknown) != 1 {
		t.Fatalf("expected unknown symbol record, got %d", len(store.unknown))
	}
	if store.unknown[0] != "binance:NEWTKUSDT:NEWTK" {
		t.Fatalf("unexpected unknown symbol payload %q", store.unknown[0])
	}
}

func TestDeriveBaseAsset(t *testing.T) {
	tests := map[string]string{
		"BTCUSDT": "BTC",
		"ETHBTC":  "ETH",
		"BONK":    "",
	}

	for symbol, want := range tests {
		if got := deriveBaseAsset(symbol); got != want {
			t.Fatalf("deriveBaseAsset(%q) = %q, want %q", symbol, got, want)
		}
	}
}
