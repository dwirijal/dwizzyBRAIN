package impact

import (
	"context"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/engine/market/ohlcv"

	"github.com/jackc/pgx/v5"
)

type stubMappingLister struct {
	items []mapping.Mapping
	err   error
}

func (s stubMappingLister) ListMappingsByCoin(ctx context.Context, coinID string) ([]mapping.Mapping, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]mapping.Mapping(nil), s.items...), nil
}

type stubCandleReader struct {
	candles map[string]ohlcv.Candle
	err     error
}

func (s stubCandleReader) GetCandleAtOrBefore(ctx context.Context, coinID, exchange, timeframe string, at time.Time) (ohlcv.Candle, error) {
	if s.err != nil {
		return ohlcv.Candle{}, s.err
	}
	key := coinID + "|" + exchange + "|" + timeframe + "|" + at.UTC().Format(time.RFC3339)
	candle, ok := s.candles[key]
	if !ok {
		return ohlcv.Candle{}, pgx.ErrNoRows
	}
	return candle, nil
}

func TestPriceResolverFallsBackToNextExchange(t *testing.T) {
	base := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)
	resolver := NewPriceResolver(
		stubMappingLister{items: []mapping.Mapping{
			{CoinID: "bitcoin", Exchange: "binance", ExchangeSymbol: "BTC/USDT", IsPrimary: true},
			{CoinID: "bitcoin", Exchange: "bybit", ExchangeSymbol: "BTC/USDT"},
		}},
		stubCandleReader{candles: map[string]ohlcv.Candle{
			"bitcoin|bybit|1m|2026-03-19T00:00:00Z": {
				CoinID:    "bitcoin",
				Exchange:  "bybit",
				Symbol:    "BTC/USDT",
				Close:     71234.5,
				Timeframe: "1m",
				Timestamp: base,
			},
		}},
		"1m",
	)

	sample, err := resolver.Resolve(context.Background(), "bitcoin", base)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if sample.Exchange != "bybit" {
		t.Fatalf("Exchange = %q, want bybit", sample.Exchange)
	}
	if sample.Price != 71234.5 {
		t.Fatalf("Price = %v, want 71234.5", sample.Price)
	}
}
