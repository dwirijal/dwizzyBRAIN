package mapping

import (
	"context"
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

type lifecycleStore struct {
	coins          []CoinRecord
	mappings       []Mapping
	pendingUnknown []UnknownSymbol
	upserts        []Mapping
	statuses       map[string]string
	unknownStatus  map[string]string
	resolvedCoinID map[string]string
}

func (s *lifecycleStore) ListCoins(ctx context.Context) ([]CoinRecord, error) {
	return s.coins, nil
}

func (s *lifecycleStore) UpsertMapping(ctx context.Context, mapping Mapping, status string) error {
	s.upserts = append(s.upserts, mapping)
	if s.statuses == nil {
		s.statuses = map[string]string{}
	}
	s.statuses[normalizeExchange(mapping.Exchange)+":"+normalizeSymbol(mapping.ExchangeSymbol)] = status
	return nil
}

func (s *lifecycleStore) ListMappingsByExchange(ctx context.Context, exchange string) ([]Mapping, error) {
	return s.mappings, nil
}

func (s *lifecycleStore) SetMappingStatus(ctx context.Context, exchange, symbol, status string, verifiedAt time.Time) error {
	if s.statuses == nil {
		s.statuses = map[string]string{}
	}
	s.statuses[normalizeExchange(exchange)+":"+normalizeSymbol(symbol)] = status
	return nil
}

func (s *lifecycleStore) ListPendingUnknownSymbols(ctx context.Context, limit int) ([]UnknownSymbol, error) {
	return s.pendingUnknown, nil
}

func (s *lifecycleStore) UpdateUnknownSymbol(ctx context.Context, exchange, rawSymbol, status, resolvedCoinID, notes string) error {
	if s.unknownStatus == nil {
		s.unknownStatus = map[string]string{}
	}
	if s.resolvedCoinID == nil {
		s.resolvedCoinID = map[string]string{}
	}
	key := normalizeExchange(exchange) + ":" + normalizeSymbol(rawSymbol)
	s.unknownStatus[key] = status
	s.resolvedCoinID[key] = resolvedCoinID
	return nil
}

func TestMappingBuilderBuildFromMarkets(t *testing.T) {
	store := &lifecycleStore{
		coins: []CoinRecord{
			{CoinID: "bitcoin", Symbol: "BTC"},
			{CoinID: "ethereum", Symbol: "ETH"},
		},
	}

	builder := NewMappingBuilder(store)
	now := time.Date(2026, 3, 18, 22, 0, 0, 0, time.UTC)
	builder.now = func() time.Time { return now }

	spot := true
	inactive := false
	btcUSDT := "BTC/USDT"
	btcUSDC := "BTC/USDC"
	ethUSDT := "ETH/USDT"
	fut := "BTC/USDT:USDT"

	result, err := builder.BuildFromMarkets(context.Background(), "binance", map[string]ccxt.MarketInterface{
		"BTC/USDT": {Symbol: &btcUSDT, BaseCurrency: strptr("BTC"), QuoteCurrency: strptr("USDT"), Spot: &spot},
		"BTC/USDC": {Symbol: &btcUSDC, BaseCurrency: strptr("BTC"), QuoteCurrency: strptr("USDC"), Spot: &spot},
		"ETH/USDT": {Symbol: &ethUSDT, BaseCurrency: strptr("ETH"), QuoteCurrency: strptr("USDT"), Spot: &spot, Active: &inactive},
		"BTC-PERP": {Symbol: &fut, BaseCurrency: strptr("BTC"), QuoteCurrency: strptr("USDT"), Spot: boolptr(false)},
	})
	if err != nil {
		t.Fatalf("BuildFromMarkets() returned error: %v", err)
	}

	if result.Matched != 3 {
		t.Fatalf("expected 3 matched mappings, got %d", result.Matched)
	}
	if result.Skipped != 1 {
		t.Fatalf("expected 1 skipped market, got %d", result.Skipped)
	}

	var btcUSDTPrimary, btcUSDCHasPrimary bool
	for _, item := range store.upserts {
		if item.ExchangeSymbol == "BTC/USDT" && item.IsPrimary {
			btcUSDTPrimary = true
		}
		if item.ExchangeSymbol == "BTC/USDC" && item.IsPrimary {
			btcUSDCHasPrimary = true
		}
	}
	if !btcUSDTPrimary {
		t.Fatal("expected BTC/USDT to be primary mapping")
	}
	if btcUSDCHasPrimary {
		t.Fatal("expected BTC/USDC not to be primary while BTC/USDT exists")
	}
}

func TestMappingValidatorValidateExchangeMarkets(t *testing.T) {
	store := &lifecycleStore{
		mappings: []Mapping{
			{CoinID: "bitcoin", Exchange: "kraken", ExchangeSymbol: "BTC/USDT"},
			{CoinID: "ethereum", Exchange: "kraken", ExchangeSymbol: "ETH/USDT"},
		},
	}
	validator := NewMappingValidator(store)

	spot := true
	active := true
	result, err := validator.ValidateExchangeMarkets(context.Background(), "kraken", map[string]ccxt.MarketInterface{
		"BTC/USDT": {Symbol: strptr("BTC/USDT"), Spot: &spot, Active: &active},
	})
	if err != nil {
		t.Fatalf("ValidateExchangeMarkets() returned error: %v", err)
	}

	if result.Validated != 2 || result.Active != 1 || result.Delisted != 1 {
		t.Fatalf("unexpected validation result: %+v", result)
	}
	if store.statuses["kraken:BTC/USDT"] != "active" {
		t.Fatalf("expected BTC/USDT active, got %s", store.statuses["kraken:BTC/USDT"])
	}
	if store.statuses["kraken:ETH/USDT"] != "delisted" {
		t.Fatalf("expected ETH/USDT delisted, got %s", store.statuses["kraken:ETH/USDT"])
	}
}

func TestUnknownSymbolResolverResolvePending(t *testing.T) {
	store := &lifecycleStore{
		coins: []CoinRecord{
			{CoinID: "bitcoin", Symbol: "BTC"},
			{CoinID: "ethereum", Symbol: "ETH"},
		},
		pendingUnknown: []UnknownSymbol{
			{Exchange: "binance", RawSymbol: "BTCUSDT", BaseAsset: "BTC"},
			{Exchange: "binance", RawSymbol: "ABCUSDT", BaseAsset: "ABC"},
		},
	}

	resolver := NewUnknownSymbolResolver(store)
	result, err := resolver.ResolvePending(context.Background(), 10)
	if err != nil {
		t.Fatalf("ResolvePending() returned error: %v", err)
	}

	if result.Resolved != 1 || result.Unresolvable != 1 {
		t.Fatalf("unexpected unknown resolve result: %+v", result)
	}
	if store.unknownStatus["binance:BTCUSDT"] != "resolved" {
		t.Fatalf("expected BTCUSDT resolved, got %s", store.unknownStatus["binance:BTCUSDT"])
	}
	if store.resolvedCoinID["binance:BTCUSDT"] != "bitcoin" {
		t.Fatalf("expected BTCUSDT -> bitcoin, got %s", store.resolvedCoinID["binance:BTCUSDT"])
	}
	if store.unknownStatus["binance:ABCUSDT"] != "unresolvable" {
		t.Fatalf("expected ABCUSDT unresolvable, got %s", store.unknownStatus["binance:ABCUSDT"])
	}
}

func strptr(v string) *string { return &v }
func boolptr(v bool) *bool    { return &v }
