package mapping

import (
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

type mappingLister interface {
	ListMappingsByExchange(ctx context.Context, exchange string) ([]Mapping, error)
	SetMappingStatus(ctx context.Context, exchange, symbol, status string, verifiedAt time.Time) error
}

type ValidationResult struct {
	Active    int
	Delisted  int
	Validated int
}

type MappingValidator struct {
	store mappingLister
	now   func() time.Time
}

func NewMappingValidator(store mappingLister) *MappingValidator {
	return &MappingValidator{
		store: store,
		now:   time.Now,
	}
}

func (v *MappingValidator) ValidateExchangeMarkets(ctx context.Context, exchange string, markets map[string]ccxt.MarketInterface) (ValidationResult, error) {
	if v.store == nil {
		return ValidationResult{}, fmt.Errorf("validator store is required")
	}

	existing, err := v.store.ListMappingsByExchange(ctx, exchange)
	if err != nil {
		return ValidationResult{}, fmt.Errorf("list mappings: %w", err)
	}

	available := make(map[string]bool, len(markets))
	for _, market := range markets {
		if !isSpotMarket(market) {
			continue
		}
		symbol := normalizeSymbol(stringPtr(market.Symbol))
		if symbol == "" {
			continue
		}
		available[symbol] = isMarketActive(market)
	}

	result := ValidationResult{}
	now := v.now().UTC()
	for _, mapping := range existing {
		result.Validated++
		symbol := normalizeSymbol(mapping.ExchangeSymbol)
		active, ok := available[symbol]
		status := "delisted"
		if ok && active {
			status = "active"
			result.Active++
		} else {
			result.Delisted++
		}

		if err := v.store.SetMappingStatus(ctx, exchange, symbol, status, now); err != nil {
			return ValidationResult{}, fmt.Errorf("set status for %s:%s: %w", exchange, symbol, err)
		}
	}

	return result, nil
}

func isMarketActive(market ccxt.MarketInterface) bool {
	if market.Active != nil {
		return *market.Active
	}
	if market.Type != nil && strings.TrimSpace(strings.ToLower(*market.Type)) != "spot" {
		return false
	}
	return true
}
