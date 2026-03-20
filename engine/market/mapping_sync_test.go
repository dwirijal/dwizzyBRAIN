package market

import (
	"context"
	"errors"
	"testing"

	"dwizzyBRAIN/engine/market/mapping"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

type stubMarketLoader struct {
	markets    map[string]ccxt.MarketInterface
	err        error
	calls      int
	lastCalled string
}

func (s *stubMarketLoader) LoadMarkets(ctx context.Context, exchangeID string) (map[string]ccxt.MarketInterface, error) {
	s.calls++
	s.lastCalled = exchangeID
	if s.err != nil {
		return nil, s.err
	}
	return s.markets, nil
}

type stubMappingBuilder struct {
	result       mapping.BuildResult
	err          error
	calls        int
	lastExchange string
	lastMarkets  map[string]ccxt.MarketInterface
}

func (s *stubMappingBuilder) BuildFromMarkets(ctx context.Context, exchange string, markets map[string]ccxt.MarketInterface) (mapping.BuildResult, error) {
	s.calls++
	s.lastExchange = exchange
	s.lastMarkets = markets
	if s.err != nil {
		return mapping.BuildResult{}, s.err
	}
	return s.result, nil
}

type stubMappingValidator struct {
	result       mapping.ValidationResult
	err          error
	calls        int
	lastExchange string
	lastMarkets  map[string]ccxt.MarketInterface
}

func (s *stubMappingValidator) ValidateExchangeMarkets(ctx context.Context, exchange string, markets map[string]ccxt.MarketInterface) (mapping.ValidationResult, error) {
	s.calls++
	s.lastExchange = exchange
	s.lastMarkets = markets
	if s.err != nil {
		return mapping.ValidationResult{}, s.err
	}
	return s.result, nil
}

func TestMappingSyncServiceSyncExchange(t *testing.T) {
	spot := true
	markets := map[string]ccxt.MarketInterface{
		"BTC/USDT": {Symbol: strptrMarket("BTC/USDT"), Spot: &spot},
	}
	loader := &stubMarketLoader{markets: markets}
	builder := &stubMappingBuilder{result: mapping.BuildResult{Matched: 2, Skipped: 1}}
	validator := &stubMappingValidator{result: mapping.ValidationResult{Validated: 2, Active: 1, Delisted: 1}}

	service := NewMappingSyncService(loader, builder, validator)
	result, err := service.SyncExchange(context.Background(), "kraken")
	if err != nil {
		t.Fatalf("SyncExchange() returned error: %v", err)
	}

	if result.Exchange != "kraken" {
		t.Fatalf("expected kraken, got %s", result.Exchange)
	}
	if result.Build.Matched != 2 || result.Validation.Validated != 2 {
		t.Fatalf("unexpected sync result: %+v", result)
	}
	if loader.calls != 1 || builder.calls != 1 || validator.calls != 1 {
		t.Fatalf("expected single call to loader/builder/validator, got %d/%d/%d", loader.calls, builder.calls, validator.calls)
	}
	if builder.lastExchange != "kraken" || validator.lastExchange != "kraken" {
		t.Fatalf("expected builder and validator exchange kraken, got %s and %s", builder.lastExchange, validator.lastExchange)
	}
}

func TestMappingSyncServiceSyncAll(t *testing.T) {
	loader := &stubMarketLoader{markets: map[string]ccxt.MarketInterface{}}
	builder := &stubMappingBuilder{result: mapping.BuildResult{Matched: 1}}
	validator := &stubMappingValidator{result: mapping.ValidationResult{Validated: 1, Active: 1}}

	service := NewMappingSyncService(loader, builder, validator)
	results, err := service.SyncAll(context.Background(), []string{"kraken", "mexc"})
	if err != nil {
		t.Fatalf("SyncAll() returned error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if loader.calls != 2 {
		t.Fatalf("expected loader to be called twice, got %d", loader.calls)
	}
}

func TestMappingSyncServiceReturnsLoaderError(t *testing.T) {
	service := NewMappingSyncService(
		&stubMarketLoader{err: errors.New("boom")},
		&stubMappingBuilder{},
		&stubMappingValidator{},
	)

	if _, err := service.SyncExchange(context.Background(), "kraken"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMappingSyncServiceReturnsBuilderError(t *testing.T) {
	service := NewMappingSyncService(
		&stubMarketLoader{markets: map[string]ccxt.MarketInterface{}},
		&stubMappingBuilder{err: errors.New("build failed")},
		&stubMappingValidator{},
	)

	if _, err := service.SyncExchange(context.Background(), "kraken"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMappingSyncServiceReturnsValidatorError(t *testing.T) {
	service := NewMappingSyncService(
		&stubMarketLoader{markets: map[string]ccxt.MarketInterface{}},
		&stubMappingBuilder{},
		&stubMappingValidator{err: errors.New("validate failed")},
	)

	if _, err := service.SyncExchange(context.Background(), "kraken"); err == nil {
		t.Fatal("expected error")
	}
}

func strptrMarket(v string) *string { return &v }
