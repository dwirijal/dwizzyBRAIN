package ohlcv

import (
	"context"
	"errors"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market"
	"dwizzyBRAIN/engine/market/mapping"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

type stubResolver struct {
	mapping mapping.Mapping
	err     error
	calls   int
}

func (s *stubResolver) ResolveExchangeSymbol(ctx context.Context, coinID, exchange string) (mapping.Mapping, error) {
	s.calls++
	if s.err != nil {
		return mapping.Mapping{}, s.err
	}
	return s.mapping, nil
}

type stubFetcher struct {
	candles   []ccxt.OHLCV
	err       error
	calls     int
	exchange  string
	symbol    string
	timeframe string
	since     time.Time
	limit     int
}

func (s *stubFetcher) PollOHLCV(ctx context.Context, exchangeID, symbol, timeframe string, since time.Time, limit int) ([]ccxt.OHLCV, error) {
	s.calls++
	s.exchange = exchangeID
	s.symbol = symbol
	s.timeframe = timeframe
	s.since = since
	s.limit = limit
	if s.err != nil {
		return nil, s.err
	}
	return s.candles, nil
}

type stubStore struct {
	candles []Candle
	latest  time.Time
	err     error
	upserts []Candle
	gets    int
}

func (s *stubStore) UpsertCandles(ctx context.Context, candles []Candle) error {
	if s.err != nil {
		return s.err
	}
	s.upserts = append(s.upserts, candles...)
	return nil
}

func (s *stubStore) GetCandles(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]Candle, error) {
	s.gets++
	if s.err != nil {
		return nil, s.err
	}
	return s.candles, nil
}

func (s *stubStore) LatestTimestamp(ctx context.Context, coinID, exchange, timeframe string) (time.Time, error) {
	if s.err != nil {
		return time.Time{}, s.err
	}
	return s.latest, nil
}

type stubPublisher struct {
	messages []market.OHLCVMessage
	err      error
}

func (s *stubPublisher) PublishOHLCV(ctx context.Context, candle market.OHLCVMessage) error {
	if s.err != nil {
		return s.err
	}
	s.messages = append(s.messages, candle)
	return nil
}

func TestServiceBackfillOHLCV(t *testing.T) {
	resolver := &stubResolver{
		mapping: mapping.Mapping{
			CoinID:         "bitcoin",
			Exchange:       "binance",
			ExchangeSymbol: "BTC/USDT",
			BaseAsset:      "BTC",
			QuoteAsset:     "USDT",
		},
	}
	fetcher := &stubFetcher{
		candles: []ccxt.OHLCV{
			{Timestamp: time.Date(2026, 3, 18, 20, 0, 0, 0, time.UTC).UnixMilli(), Open: 64000, High: 64100, Low: 63900, Close: 64050, Volume: 123},
			{Timestamp: time.Date(2026, 3, 18, 20, 1, 0, 0, time.UTC).UnixMilli(), Open: 64050, High: 64200, Low: 64000, Close: 64150, Volume: 124},
		},
	}
	store := &stubStore{}
	publisher := &stubPublisher{}

	service := NewService(resolver, fetcher, store, publisher)
	service.now = func() time.Time { return time.Date(2026, 3, 18, 20, 2, 0, 0, time.UTC) }

	candles, err := service.BackfillOHLCV(context.Background(), SyncRequest{
		CoinID:    "bitcoin",
		Exchange:  "binance",
		Timeframe: "1m",
		Since:     time.Date(2026, 3, 18, 19, 0, 0, 0, time.UTC),
		Limit:     2,
	})
	if err != nil {
		t.Fatalf("BackfillOHLCV() returned error: %v", err)
	}
	if len(candles) != 2 {
		t.Fatalf("expected 2 candles, got %d", len(candles))
	}
	if fetcher.symbol != "BTC/USDT" || fetcher.timeframe != "1m" {
		t.Fatalf("unexpected fetch args: symbol=%s timeframe=%s", fetcher.symbol, fetcher.timeframe)
	}
	if len(store.upserts) != 2 || len(publisher.messages) != 2 {
		t.Fatalf("expected 2 upserts and 2 publishes, got %d and %d", len(store.upserts), len(publisher.messages))
	}
	if !candles[0].IsClosed {
		t.Fatal("expected first candle to be closed")
	}
}

func TestServiceIncrementalSyncUsesLatestTimestamp(t *testing.T) {
	resolver := &stubResolver{mapping: mapping.Mapping{CoinID: "bitcoin", Exchange: "kraken", ExchangeSymbol: "BTC/USDT"}}
	fetcher := &stubFetcher{}
	store := &stubStore{latest: time.Date(2026, 3, 18, 20, 0, 0, 0, time.UTC)}
	service := NewService(resolver, fetcher, store, nil)
	service.now = time.Now

	if _, err := service.IncrementalSync(context.Background(), SyncRequest{
		CoinID:    "bitcoin",
		Exchange:  "kraken",
		Timeframe: "5m",
		Limit:     100,
	}); err != nil {
		t.Fatalf("IncrementalSync() returned error: %v", err)
	}
	if !fetcher.since.Equal(store.latest) {
		t.Fatalf("expected since=%s, got %s", store.latest, fetcher.since)
	}
}

func TestServiceGetOHLCV(t *testing.T) {
	store := &stubStore{candles: []Candle{{CoinID: "bitcoin", Exchange: "binance", Timeframe: "1m"}}}
	service := NewService(nil, nil, store, nil)

	got, err := service.GetOHLCV(context.Background(), "bitcoin", "binance", "1m", 50)
	if err != nil {
		t.Fatalf("GetOHLCV() returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 candle, got %d", len(got))
	}
}

func TestServiceBackfillReturnsResolverError(t *testing.T) {
	service := NewService(&stubResolver{err: errors.New("missing mapping")}, &stubFetcher{}, &stubStore{}, nil)
	if _, err := service.BackfillOHLCV(context.Background(), SyncRequest{
		CoinID: "bitcoin", Exchange: "binance", Timeframe: "1m",
	}); err == nil {
		t.Fatal("expected error")
	}
}
