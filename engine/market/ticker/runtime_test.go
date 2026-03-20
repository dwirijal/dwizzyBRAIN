package ticker

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/shared/schema"
)

type stubTickerPoller struct {
	raw schema.RawTicker
	err error
}

func (s stubTickerPoller) PollTicker(_ context.Context, exchangeID, symbol string) (schema.RawTicker, error) {
	if s.err != nil {
		return schema.RawTicker{}, s.err
	}
	out := s.raw
	out.Exchange = exchangeID
	out.Symbol = symbol
	return out, nil
}

type stubSymbolLookup struct {
	mapping mapping.Mapping
	err     error
}

func (s stubSymbolLookup) ResolveExchangeSymbol(_ context.Context, coinID, exchange string) (mapping.Mapping, error) {
	if s.err != nil {
		return mapping.Mapping{}, s.err
	}
	out := s.mapping
	out.CoinID = coinID
	out.Exchange = exchange
	return out, nil
}

type stubPollingIngestion struct {
	resolved schema.ResolvedTicker
	err      error
	raws     []schema.RawTicker
}

func (s *stubPollingIngestion) ProcessTicker(_ context.Context, raw schema.RawTicker) (schema.ResolvedTicker, error) {
	s.raws = append(s.raws, raw)
	if s.err != nil {
		return schema.ResolvedTicker{}, s.err
	}
	out := s.resolved
	out.Symbol = raw.Symbol
	out.Exchange = raw.Exchange
	return out, nil
}

func TestCCXTPollJobRunOnceUpdatesAggregator(t *testing.T) {
	agg := NewAggregator()
	now := time.Now().UTC()
	ingestion := &stubPollingIngestion{
		resolved: schema.ResolvedTicker{
			CoinID:         "bitcoin",
			Symbol:         "BTC/USDT",
			Exchange:       "binance",
			BaseAsset:      "BTC",
			QuoteAsset:     "USDT",
			ResolvedSymbol: "BTC/USDT",
			Price:          70000,
			Bid:            69990,
			Ask:            70010,
			Volume:         1234,
			Timestamp:      now,
		},
	}
	job := NewCCXTPollJob(
		stubTickerPoller{
			raw: schema.RawTicker{
				Price:     70000,
				Bid:       69990,
				Ask:       70010,
				Volume:    1234,
				Timestamp: now,
			},
		},
		stubSymbolLookup{
			mapping: mapping.Mapping{
				CoinID:         "bitcoin",
				Exchange:       "binance",
				ExchangeSymbol: "BTC/USDT",
				BaseAsset:      "BTC",
				QuoteAsset:     "USDT",
			},
		},
		ingestion,
		agg,
		[]PollTarget{{CoinID: "bitcoin", Exchange: "binance"}},
		time.Second,
		log.New(io.Discard, "", 0),
	)

	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}

	if len(ingestion.raws) != 1 {
		t.Fatalf("expected 1 raw ticker, got %d", len(ingestion.raws))
	}
	if ingestion.raws[0].Symbol != "BTC/USDT" {
		t.Fatalf("expected polled symbol BTC/USDT, got %s", ingestion.raws[0].Symbol)
	}

	snapshot, ok := agg.Snapshot("bitcoin")
	if !ok {
		t.Fatal("expected bitcoin snapshot to exist")
	}
	if snapshot.BestBid != 69990 || snapshot.BestAsk != 70010 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.BestBidExchange != "binance" || snapshot.BestAskExchange != "binance" {
		t.Fatalf("unexpected best exchanges: %+v", snapshot)
	}
}

func TestCCXTPollJobRunOnceResolveError(t *testing.T) {
	job := NewCCXTPollJob(
		stubTickerPoller{},
		stubSymbolLookup{err: mapping.ErrMappingNotFound},
		&stubPollingIngestion{},
		NewAggregator(),
		[]PollTarget{{CoinID: "bitcoin", Exchange: "binance"}},
		time.Second,
		log.New(io.Discard, "", 0),
	)

	err := job.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, mapping.ErrMappingNotFound) {
		t.Fatalf("expected ErrMappingNotFound, got %v", err)
	}
}

func TestSpreadJobRunOnceRequiresRecorder(t *testing.T) {
	job := NewSpreadJob(nil, time.Second, log.New(io.Discard, "", 0))
	if err := job.RunOnce(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
