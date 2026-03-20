package market

import (
	"context"
	"fmt"
	"strings"

	"dwizzyBRAIN/engine/market/mapping"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

type marketLoader interface {
	LoadMarkets(ctx context.Context, exchangeID string) (map[string]ccxt.MarketInterface, error)
}

type mappingBuilder interface {
	BuildFromMarkets(ctx context.Context, exchange string, markets map[string]ccxt.MarketInterface) (mapping.BuildResult, error)
}

type mappingValidator interface {
	ValidateExchangeMarkets(ctx context.Context, exchange string, markets map[string]ccxt.MarketInterface) (mapping.ValidationResult, error)
}

type MappingSyncResult struct {
	Exchange   string                   `json:"exchange"`
	Build      mapping.BuildResult      `json:"build"`
	Validation mapping.ValidationResult `json:"validation"`
}

type MappingSyncService struct {
	loader    marketLoader
	builder   mappingBuilder
	validator mappingValidator
}

func NewMappingSyncService(loader marketLoader, builder mappingBuilder, validator mappingValidator) *MappingSyncService {
	return &MappingSyncService{
		loader:    loader,
		builder:   builder,
		validator: validator,
	}
}

func (s *MappingSyncService) SyncExchange(ctx context.Context, exchange string) (MappingSyncResult, error) {
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	if exchange == "" {
		return MappingSyncResult{}, fmt.Errorf("exchange is required")
	}
	if s.loader == nil {
		return MappingSyncResult{}, fmt.Errorf("market loader is required")
	}
	if s.builder == nil {
		return MappingSyncResult{}, fmt.Errorf("mapping builder is required")
	}
	if s.validator == nil {
		return MappingSyncResult{}, fmt.Errorf("mapping validator is required")
	}

	markets, err := s.loader.LoadMarkets(ctx, exchange)
	if err != nil {
		return MappingSyncResult{}, fmt.Errorf("load markets for %s: %w", exchange, err)
	}

	buildResult, err := s.builder.BuildFromMarkets(ctx, exchange, markets)
	if err != nil {
		return MappingSyncResult{}, fmt.Errorf("build mappings for %s: %w", exchange, err)
	}

	validationResult, err := s.validator.ValidateExchangeMarkets(ctx, exchange, markets)
	if err != nil {
		return MappingSyncResult{}, fmt.Errorf("validate mappings for %s: %w", exchange, err)
	}

	return MappingSyncResult{
		Exchange:   exchange,
		Build:      buildResult,
		Validation: validationResult,
	}, nil
}

func (s *MappingSyncService) SyncAll(ctx context.Context, exchanges []string) ([]MappingSyncResult, error) {
	results := make([]MappingSyncResult, 0, len(exchanges))
	for _, exchange := range exchanges {
		result, err := s.SyncExchange(ctx, exchange)
		if err != nil {
			return results, err
		}
		results = append(results, result)
	}

	return results, nil
}
