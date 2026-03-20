package mapping

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"
)

const defaultCacheTTL = time.Hour

var ErrMappingNotFound = errors.New("mapping not found")

type Mapping struct {
	CoinID         string    `json:"coin_id"`
	Exchange       string    `json:"exchange"`
	ExchangeSymbol string    `json:"exchange_symbol"`
	BaseAsset      string    `json:"base_asset"`
	QuoteAsset     string    `json:"quote_asset"`
	IsPrimary      bool      `json:"is_primary"`
	VerifiedAt     time.Time `json:"verified_at,omitempty"`
}

type Store interface {
	GetPrimaryMapping(ctx context.Context, coinID, exchange string) (Mapping, error)
	GetMappingBySymbol(ctx context.Context, exchange, symbol string) (Mapping, error)
	RecordUnknownSymbol(ctx context.Context, exchange, rawSymbol, baseAsset string) error
}

type SymbolResolver struct {
	store    Store
	cache    redis.Cmdable
	cacheTTL time.Duration
}

func NewSymbolResolver(store Store, cache redis.Cmdable) *SymbolResolver {
	return &SymbolResolver{
		store:    store,
		cache:    cache,
		cacheTTL: defaultCacheTTL,
	}
}

func (r *SymbolResolver) ResolveExchangeSymbol(ctx context.Context, coinID, exchange string) (Mapping, error) {
	coinID = strings.TrimSpace(coinID)
	exchange = normalizeExchange(exchange)
	if coinID == "" || exchange == "" {
		return Mapping{}, fmt.Errorf("coinID and exchange are required")
	}

	cacheKey := primaryCacheKey(coinID, exchange)
	if mapping, err := r.loadCache(ctx, cacheKey); err == nil {
		return mapping, nil
	}

	mapping, err := r.store.GetPrimaryMapping(ctx, coinID, exchange)
	if err != nil {
		return Mapping{}, err
	}

	r.writeCache(ctx, cacheKey, mapping)
	r.writeCache(ctx, reverseCacheKey(exchange, mapping.ExchangeSymbol), mapping)

	return mapping, nil
}

func (r *SymbolResolver) ResolveCoinID(ctx context.Context, exchange, rawSymbol string) (Mapping, error) {
	exchange = normalizeExchange(exchange)
	rawSymbol = normalizeSymbol(rawSymbol)
	if exchange == "" || rawSymbol == "" {
		return Mapping{}, fmt.Errorf("exchange and rawSymbol are required")
	}

	cacheKey := reverseCacheKey(exchange, rawSymbol)
	if mapping, err := r.loadCache(ctx, cacheKey); err == nil {
		return mapping, nil
	}

	mapping, err := r.store.GetMappingBySymbol(ctx, exchange, rawSymbol)
	if err == nil {
		r.writeCache(ctx, cacheKey, mapping)
		r.writeCache(ctx, primaryCacheKey(mapping.CoinID, exchange), mapping)
		return mapping, nil
	}

	if errors.Is(err, ErrMappingNotFound) {
		baseAsset := deriveBaseAsset(rawSymbol)
		_ = r.store.RecordUnknownSymbol(ctx, exchange, rawSymbol, baseAsset)
	}

	return Mapping{}, err
}

func (r *SymbolResolver) loadCache(ctx context.Context, key string) (Mapping, error) {
	if r.cache == nil {
		return Mapping{}, ErrMappingNotFound
	}

	payload, err := r.cache.Get(ctx, key).Result()
	if err != nil {
		return Mapping{}, err
	}

	var mapping Mapping
	if err := json.Unmarshal([]byte(payload), &mapping); err != nil {
		return Mapping{}, err
	}

	return mapping, nil
}

func (r *SymbolResolver) writeCache(ctx context.Context, key string, mapping Mapping) {
	if r.cache == nil {
		return
	}

	payload, err := json.Marshal(mapping)
	if err != nil {
		return
	}

	_ = r.cache.Set(ctx, key, payload, r.cacheTTL).Err()
}

func primaryCacheKey(coinID, exchange string) string {
	return "mapping:coin:" + strings.ToLower(strings.TrimSpace(coinID)) + ":" + normalizeExchange(exchange)
}

func reverseCacheKey(exchange, symbol string) string {
	return "mapping:symbol:" + normalizeExchange(exchange) + ":" + normalizeSymbol(symbol)
}

func normalizeExchange(exchange string) string {
	return strings.ToLower(strings.TrimSpace(exchange))
}

func normalizeSymbol(symbol string) string {
	return strings.ToUpper(strings.TrimSpace(symbol))
}

func deriveBaseAsset(symbol string) string {
	symbol = normalizeSymbol(symbol)
	for _, quote := range []string{"USDT", "USDC", "BUSD", "BTC", "ETH"} {
		if strings.HasSuffix(symbol, quote) && len(symbol) > len(quote) {
			return strings.TrimSuffix(symbol, quote)
		}
	}

	return ""
}
