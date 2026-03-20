package impact

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/engine/market/ohlcv"

	"github.com/jackc/pgx/v5"
)

type mappingLister interface {
	ListMappingsByCoin(ctx context.Context, coinID string) ([]mapping.Mapping, error)
}

type candleReader interface {
	GetCandleAtOrBefore(ctx context.Context, coinID, exchange, timeframe string, at time.Time) (ohlcv.Candle, error)
}

type PriceSample struct {
	CoinID    string
	Exchange  string
	Symbol    string
	Price     float64
	Timestamp time.Time
}

type PriceResolver struct {
	mappings  mappingLister
	candles   candleReader
	timeframe string
}

func NewPriceResolver(mappings mappingLister, candles candleReader, timeframe string) *PriceResolver {
	if strings.TrimSpace(timeframe) == "" {
		timeframe = "1m"
	}
	return &PriceResolver{
		mappings:  mappings,
		candles:   candles,
		timeframe: strings.ToLower(strings.TrimSpace(timeframe)),
	}
}

func (r *PriceResolver) Resolve(ctx context.Context, coinID string, at time.Time) (PriceSample, error) {
	if r.mappings == nil {
		return PriceSample{}, fmt.Errorf("mapping lister is required")
	}
	if r.candles == nil {
		return PriceSample{}, fmt.Errorf("candle reader is required")
	}

	coinID = strings.TrimSpace(coinID)
	if coinID == "" {
		return PriceSample{}, fmt.Errorf("coin_id is required")
	}

	items, err := r.mappings.ListMappingsByCoin(ctx, coinID)
	if err != nil {
		return PriceSample{}, fmt.Errorf("list coin mappings: %w", err)
	}
	items = dedupeMappingsByExchange(items)
	if len(items) == 0 {
		return PriceSample{}, ErrPriceNotFound
	}

	at = at.UTC()
	for _, item := range items {
		candle, err := r.candles.GetCandleAtOrBefore(ctx, coinID, item.Exchange, r.timeframe, at)
		if err != nil {
			if err == pgx.ErrNoRows {
				continue
			}
			return PriceSample{}, fmt.Errorf("resolve price %s on %s: %w", coinID, item.Exchange, err)
		}
		if candle.Close <= 0 {
			continue
		}

		return PriceSample{
			CoinID:    coinID,
			Exchange:  strings.ToLower(strings.TrimSpace(item.Exchange)),
			Symbol:    candle.Symbol,
			Price:     candle.Close,
			Timestamp: candle.Timestamp.UTC(),
		}, nil
	}

	return PriceSample{}, ErrPriceNotFound
}

var ErrPriceNotFound = fmt.Errorf("price not found")

func dedupeMappingsByExchange(items []mapping.Mapping) []mapping.Mapping {
	if len(items) == 0 {
		return nil
	}

	unique := make(map[string]mapping.Mapping, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.Exchange))
		current, ok := unique[key]
		if !ok {
			unique[key] = item
			order = append(order, key)
			continue
		}
		if !current.IsPrimary && item.IsPrimary {
			unique[key] = item
		}
	}

	out := make([]mapping.Mapping, 0, len(unique))
	for _, key := range order {
		out = append(out, unique[key])
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].IsPrimary != out[j].IsPrimary {
			return out[i].IsPrimary
		}
		if exchangePriority(out[i].Exchange) != exchangePriority(out[j].Exchange) {
			return exchangePriority(out[i].Exchange) < exchangePriority(out[j].Exchange)
		}
		return strings.ToUpper(strings.TrimSpace(out[i].ExchangeSymbol)) < strings.ToUpper(strings.TrimSpace(out[j].ExchangeSymbol))
	})

	return out
}

func exchangePriority(exchange string) int {
	switch strings.ToLower(strings.TrimSpace(exchange)) {
	case "binance":
		return 0
	case "bybit":
		return 1
	case "okx":
		return 2
	case "kucoin":
		return 3
	case "gateio":
		return 4
	case "kraken":
		return 5
	case "mexc":
		return 6
	case "htx":
		return 7
	default:
		return 100
	}
}
