package market

import (
	"context"
	"errors"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/shared/schema"
)

type stubRawTickerBatchReader struct {
	tickers []schema.RawTicker
	err     error
	calls   int
}

func (s *stubRawTickerBatchReader) ReadMessage() ([]schema.RawTicker, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.tickers, nil
}

type stubRawTickerPoller struct {
	ticker     schema.RawTicker
	err        error
	calls      int
	exchangeID string
	symbol     string
}

func (s *stubRawTickerPoller) PollTicker(ctx context.Context, exchangeID, symbol string) (schema.RawTicker, error) {
	s.calls++
	s.exchangeID = exchangeID
	s.symbol = symbol
	if s.err != nil {
		return schema.RawTicker{}, s.err
	}
	return s.ticker, nil
}

func TestIngestionServiceProcessBatch(t *testing.T) {
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
	publisher := &stubResolvedTickerPublisher{}
	service := NewIngestionService(resolver, publisher)

	raws := []schema.RawTicker{
		{
			Symbol:    "BTCUSDT",
			Exchange:  "binance",
			Price:     65000,
			Bid:       64999,
			Ask:       65001,
			Volume:    100,
			Timestamp: time.Unix(1710000000, 0).UTC(),
		},
		{
			Symbol:    "BTCUSDT",
			Exchange:  "binance",
			Price:     65010,
			Bid:       65009,
			Ask:       65011,
			Volume:    101,
			Timestamp: time.Unix(1710000060, 0).UTC(),
		},
	}

	got, err := service.ProcessBatch(context.Background(), raws)
	if err != nil {
		t.Fatalf("ProcessBatch() returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 resolved tickers, got %d", len(got))
	}
	if resolver.calls != 2 {
		t.Fatalf("expected resolver to be called twice, got %d", resolver.calls)
	}
	if publisher.calls != 2 {
		t.Fatalf("expected publisher to be called twice, got %d", publisher.calls)
	}
}

func TestWSIngestionRunnerReadAndProcess(t *testing.T) {
	reader := &stubRawTickerBatchReader{
		tickers: []schema.RawTicker{
			{
				Symbol:    "ETHUSDT",
				Exchange:  "binance",
				Price:     3200,
				Bid:       3199,
				Ask:       3201,
				Volume:    50,
				Timestamp: time.Unix(1710000100, 0).UTC(),
			},
		},
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
	runner := NewWSIngestionRunner(reader, NewIngestionService(resolver, publisher))

	got, err := runner.ReadAndProcess(context.Background())
	if err != nil {
		t.Fatalf("ReadAndProcess() returned error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 resolved ticker, got %d", len(got))
	}
	if got[0].CoinID != "ethereum" {
		t.Fatalf("expected ethereum, got %s", got[0].CoinID)
	}
}

func TestRESTIngestionRunnerPollAndProcess(t *testing.T) {
	poller := &stubRawTickerPoller{
		ticker: schema.RawTicker{
			Symbol:    "BTC/USDT",
			Exchange:  "kraken",
			Price:     64000,
			Bid:       63999,
			Ask:       64001,
			Volume:    10,
			Timestamp: time.Unix(1710000200, 0).UTC(),
		},
	}
	resolver := &stubTickerResolver{
		mapping: mapping.Mapping{
			CoinID:         "bitcoin",
			Exchange:       "kraken",
			ExchangeSymbol: "BTC/USDT",
			BaseAsset:      "BTC",
			QuoteAsset:     "USDT",
			IsPrimary:      true,
		},
	}
	publisher := &stubResolvedTickerPublisher{}
	runner := NewRESTIngestionRunner(poller, NewIngestionService(resolver, publisher))

	got, err := runner.PollAndProcess(context.Background(), "kraken", "BTC/USDT")
	if err != nil {
		t.Fatalf("PollAndProcess() returned error: %v", err)
	}
	if got.CoinID != "bitcoin" {
		t.Fatalf("expected bitcoin, got %s", got.CoinID)
	}
	if poller.calls != 1 {
		t.Fatalf("expected poller to be called once, got %d", poller.calls)
	}
	if poller.exchangeID != "kraken" || poller.symbol != "BTC/USDT" {
		t.Fatalf("unexpected poller inputs: %s %s", poller.exchangeID, poller.symbol)
	}
}

func TestWSIngestionRunnerReturnsReaderError(t *testing.T) {
	runner := NewWSIngestionRunner(
		&stubRawTickerBatchReader{err: errors.New("read failed")},
		NewIngestionService(&stubTickerResolver{}, nil),
	)

	if _, err := runner.ReadAndProcess(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestRESTIngestionRunnerReturnsPollerError(t *testing.T) {
	runner := NewRESTIngestionRunner(
		&stubRawTickerPoller{err: errors.New("poll failed")},
		NewIngestionService(&stubTickerResolver{}, nil),
	)

	if _, err := runner.PollAndProcess(context.Background(), "kraken", "BTC/USDT"); err == nil {
		t.Fatal("expected error")
	}
}
