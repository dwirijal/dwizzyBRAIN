package mapping

import (
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

type coinLister interface {
	ListCoins(ctx context.Context) ([]CoinRecord, error)
}

type mappingWriter interface {
	UpsertMapping(ctx context.Context, mapping Mapping, status string) error
}

type BuilderStore interface {
	coinLister
	mappingWriter
}

type BuildResult struct {
	Matched   int
	Skipped   int
	Unmatched int
}

type MappingBuilder struct {
	store BuilderStore
	now   func() time.Time
}

func NewMappingBuilder(store BuilderStore) *MappingBuilder {
	return &MappingBuilder{
		store: store,
		now:   time.Now,
	}
}

func (b *MappingBuilder) BuildFromMarkets(ctx context.Context, exchange string, markets map[string]ccxt.MarketInterface) (BuildResult, error) {
	if b.store == nil {
		return BuildResult{}, fmt.Errorf("builder store is required")
	}

	coins, err := b.store.ListCoins(ctx)
	if err != nil {
		return BuildResult{}, fmt.Errorf("list coins: %w", err)
	}

	coinBySymbol := make(map[string]CoinRecord, len(coins))
	for _, coin := range coins {
		symbol := strings.ToUpper(strings.TrimSpace(coin.Symbol))
		if symbol == "" {
			continue
		}
		if _, exists := coinBySymbol[symbol]; !exists {
			coinBySymbol[symbol] = coin
		}
	}

	candidates := make([]Mapping, 0, len(markets))
	result := BuildResult{}
	for _, market := range markets {
		if !isSpotMarket(market) {
			result.Skipped++
			continue
		}

		base := upperPtr(market.BaseCurrency)
		quote := upperPtr(market.QuoteCurrency)
		symbol := strings.TrimSpace(stringPtr(market.Symbol))
		if base == "" || quote == "" || symbol == "" {
			result.Skipped++
			continue
		}

		coin, ok := coinBySymbol[base]
		if !ok {
			result.Unmatched++
			continue
		}

		candidates = append(candidates, Mapping{
			CoinID:         coin.CoinID,
			Exchange:       normalizeExchange(exchange),
			ExchangeSymbol: symbol,
			BaseAsset:      base,
			QuoteAsset:     quote,
			VerifiedAt:     b.now().UTC(),
		})
	}

	primaries := choosePrimaryMappings(candidates)
	for i := range candidates {
		key := primaryGroupKey(candidates[i])
		candidates[i].IsPrimary = primaries[key] == normalizeSymbol(candidates[i].ExchangeSymbol)
		if err := b.store.UpsertMapping(ctx, candidates[i], "active"); err != nil {
			return BuildResult{}, fmt.Errorf("upsert mapping %s:%s: %w", candidates[i].Exchange, candidates[i].ExchangeSymbol, err)
		}
		result.Matched++
	}

	return result, nil
}

func choosePrimaryMappings(mappings []Mapping) map[string]string {
	primaries := make(map[string]string, len(mappings))
	ranks := make(map[string]int, len(mappings))

	for _, mapping := range mappings {
		key := primaryGroupKey(mapping)
		rank := quotePriority(mapping.QuoteAsset)
		currentRank, exists := ranks[key]
		if !exists || rank < currentRank {
			ranks[key] = rank
			primaries[key] = normalizeSymbol(mapping.ExchangeSymbol)
		}
	}

	return primaries
}

func primaryGroupKey(mapping Mapping) string {
	return strings.ToLower(strings.TrimSpace(mapping.CoinID)) + ":" + normalizeExchange(mapping.Exchange)
}

func quotePriority(quote string) int {
	switch strings.ToUpper(strings.TrimSpace(quote)) {
	case "USDT":
		return 0
	case "USDC":
		return 1
	case "BUSD":
		return 2
	case "BTC":
		return 3
	case "ETH":
		return 4
	default:
		return 100
	}
}

func isSpotMarket(market ccxt.MarketInterface) bool {
	if market.Spot != nil {
		return *market.Spot
	}
	if market.Contract != nil && *market.Contract {
		return false
	}
	if market.Active != nil && !*market.Active {
		return false
	}
	if market.Type != nil && strings.TrimSpace(strings.ToLower(*market.Type)) == "spot" {
		return true
	}
	return true
}

func stringPtr(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func upperPtr(value *string) string {
	return strings.ToUpper(strings.TrimSpace(stringPtr(value)))
}
