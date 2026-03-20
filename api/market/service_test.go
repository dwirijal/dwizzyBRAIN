package marketapi

import (
	"context"
	"strings"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
)

func TestServiceGuardClauses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc := NewService(nil, nil, nil, nil)

	assertErrContains := func(name string, err error, want string) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("%s: error=%q want contains %q", name, err.Error(), want)
		}
	}

	_, _, err := svc.List(ctx, 10, 0)
	assertErrContains("List", err, "postgres pool is required")

	_, err = svc.Detail(ctx, "bitcoin")
	assertErrContains("Detail", err, "postgres pool is required")

	_, err = svc.Arbitrage(ctx, "bitcoin", 5)
	assertErrContains("Arbitrage", err, "postgres pool is required")

	_, err = svc.OHLCV(ctx, "bitcoin", "binance", "1m", 5)
	assertErrContains("OHLCV", err, "ohlcv reader is required")

	_, err = svc.Tickers(ctx, "")
	assertErrContains("Tickers", err, "coin_id is required")

	_, err = svc.OrderBook(ctx, "")
	assertErrContains("OrderBook", err, "coin_id is required")
}

func TestClampAndNormalizeHelpers(t *testing.T) {
	t.Parallel()

	if got := clampLimit(-1); got != defaultListLimit {
		t.Fatalf("clampLimit(-1)=%d want=%d", got, defaultListLimit)
	}
	if got := clampLimit(99999); got != maxListLimit {
		t.Fatalf("clampLimit(99999)=%d want=%d", got, maxListLimit)
	}
	if got := clampLimit(17); got != 17 {
		t.Fatalf("clampLimit(17)=%d want=17", got)
	}

	if got := normalizeExchange("  BINANCE "); got != "binance" {
		t.Fatalf("normalizeExchange mismatch: %q", got)
	}
	if got := normalizeTimeframe(" 1H "); got != "1h" {
		t.Fatalf("normalizeTimeframe mismatch: %q", got)
	}
	if got := priceKey("", "binance"); got != "" {
		t.Fatalf("priceKey empty symbol=%q want empty", got)
	}
	if got := priceKey("BTCUSDT", " BINANCE "); got != "price:BTCUSDT:binance" {
		t.Fatalf("priceKey mismatch: %q", got)
	}
	if got := exchangePriority("binance"); got >= exchangePriority("bybit") {
		t.Fatalf("exchangePriority expected binance < bybit, got %d >= %d", got, exchangePriority("bybit"))
	}
	if got := exchangePriority("unknown"); got != 100 {
		t.Fatalf("exchangePriority unknown=%d want=100", got)
	}
}

func TestPrimaryExchangeFromMappings(t *testing.T) {
	t.Parallel()

	none := primaryExchangeFromMappings(nil)
	if none != "" {
		t.Fatalf("primaryExchangeFromMappings(nil)=%q want empty", none)
	}

	items := []mapping.Mapping{
		{Exchange: "okx", IsPrimary: false},
		{Exchange: "binance", IsPrimary: true},
	}
	if got := primaryExchangeFromMappings(items); got != "binance" {
		t.Fatalf("primaryExchangeFromMappings primary=%q want=binance", got)
	}

	fallback := []mapping.Mapping{
		{Exchange: "  BYBIT ", IsPrimary: false},
	}
	if got := primaryExchangeFromMappings(fallback); got != "bybit" {
		t.Fatalf("primaryExchangeFromMappings fallback=%q want=bybit", got)
	}
}

func TestDedupeMappingsByExchangePrefersPrimary(t *testing.T) {
	t.Parallel()

	items := []mapping.Mapping{
		{CoinID: "bitcoin", Exchange: "binance", ExchangeSymbol: "BTCUSDT", IsPrimary: false},
		{CoinID: "bitcoin", Exchange: "BINANCE", ExchangeSymbol: "XBTUSDT", IsPrimary: true},
		{CoinID: "bitcoin", Exchange: "bybit", ExchangeSymbol: "BTCUSDT", IsPrimary: false},
	}

	out := dedupeMappingsByExchange(items)
	if len(out) != 2 {
		t.Fatalf("dedupeMappingsByExchange len=%d want=2", len(out))
	}
	if !out[0].IsPrimary || strings.ToLower(out[0].Exchange) != "binance" {
		t.Fatalf("first mapping mismatch: %+v", out[0])
	}
}

func TestPickBestPriceAndBuildExchangePrices(t *testing.T) {
	t.Parallel()

	items := []mapping.Mapping{
		{CoinID: "bitcoin", Exchange: "binance", ExchangeSymbol: "BTCUSDT", IsPrimary: true},
		{CoinID: "bitcoin", Exchange: "bybit", ExchangeSymbol: "BTCUSDT", IsPrimary: false},
	}

	binancePrice := 101.5
	bybitPrice := 102.1
	priceMap := map[string]*float64{
		priceKey("bitcoin", "binance"): &binancePrice,
		priceKey("BTCUSDT", "bybit"):   &bybitPrice,
	}

	bestPrice, source, ok := pickBestPrice(priceMap, items)
	if !ok || bestPrice == nil || source == nil {
		t.Fatalf("pickBestPrice expected a result, got ok=%v price=%v source=%v", ok, bestPrice, source)
	}
	if *bestPrice != binancePrice || source.Exchange != "binance" {
		t.Fatalf("pickBestPrice mismatch: price=%v source=%+v", *bestPrice, source)
	}

	exchanges := buildExchangePrices(items, priceMap)
	if len(exchanges) != 2 {
		t.Fatalf("buildExchangePrices len=%d want=2", len(exchanges))
	}
	if exchanges[0].Price == nil || *exchanges[0].Price != binancePrice {
		t.Fatalf("buildExchangePrices[0] unexpected: %+v", exchanges[0])
	}
	if exchanges[1].Price == nil || *exchanges[1].Price != bybitPrice {
		t.Fatalf("buildExchangePrices[1] unexpected: %+v", exchanges[1])
	}
}

func TestBuildTickerSnapshotAndConvertMappings(t *testing.T) {
	t.Parallel()

	bidA, askA := 101.0, 99.0
	bidB, askB := 100.5, 98.0
	snapshot := buildTickerSnapshot("bitcoin", []TickerExchange{
		{Exchange: "binance", Bid: &bidA, Ask: &askA},
		{Exchange: "bybit", Bid: &bidB, Ask: &askB},
	})

	if snapshot.ExchangeCount != 2 {
		t.Fatalf("ExchangeCount=%d want=2", snapshot.ExchangeCount)
	}
	if snapshot.BestBid == nil || *snapshot.BestBid != bidA || snapshot.BestBidExchange != "binance" {
		t.Fatalf("BestBid mismatch: %+v", snapshot)
	}
	if snapshot.BestAsk == nil || *snapshot.BestAsk != askB || snapshot.BestAskExchange != "bybit" {
		t.Fatalf("BestAsk mismatch: %+v", snapshot)
	}
	if snapshot.CrossExchangeSpreadPct == nil {
		t.Fatalf("CrossExchangeSpreadPct is nil")
	}

	local := time.Date(2026, 3, 19, 2, 30, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	mapped := convertMappings([]mapping.Mapping{
		{
			CoinID:         "bitcoin",
			Exchange:       "binance",
			ExchangeSymbol: "BTCUSDT",
			BaseAsset:      "BTC",
			QuoteAsset:     "USDT",
			IsPrimary:      true,
			VerifiedAt:     local,
		},
	})
	if len(mapped) != 1 {
		t.Fatalf("convertMappings len=%d want=1", len(mapped))
	}
	if mapped[0].VerifiedAt.Location() != time.UTC {
		t.Fatalf("convertMappings verified_at zone=%v want UTC", mapped[0].VerifiedAt.Location())
	}
	if !mapped[0].VerifiedAt.Equal(local.UTC()) {
		t.Fatalf("convertMappings verified_at=%v want=%v", mapped[0].VerifiedAt, local.UTC())
	}
}
