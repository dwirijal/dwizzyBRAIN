package mapping

import (
	"context"
	"fmt"
	"strings"
)

type unknownStore interface {
	ListCoins(ctx context.Context) ([]CoinRecord, error)
	ListPendingUnknownSymbols(ctx context.Context, limit int) ([]UnknownSymbol, error)
	UpsertMapping(ctx context.Context, mapping Mapping, status string) error
	UpdateUnknownSymbol(ctx context.Context, exchange, rawSymbol, status, resolvedCoinID, notes string) error
}

type UnknownResolveResult struct {
	Resolved     int
	Unresolvable int
}

type UnknownSymbolResolver struct {
	store unknownStore
}

func NewUnknownSymbolResolver(store unknownStore) *UnknownSymbolResolver {
	return &UnknownSymbolResolver{store: store}
}

func (r *UnknownSymbolResolver) ResolvePending(ctx context.Context, limit int) (UnknownResolveResult, error) {
	if r.store == nil {
		return UnknownResolveResult{}, fmt.Errorf("unknown symbol store is required")
	}

	coins, err := r.store.ListCoins(ctx)
	if err != nil {
		return UnknownResolveResult{}, fmt.Errorf("list coins: %w", err)
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

	pending, err := r.store.ListPendingUnknownSymbols(ctx, limit)
	if err != nil {
		return UnknownResolveResult{}, fmt.Errorf("list pending unknown symbols: %w", err)
	}

	result := UnknownResolveResult{}
	for _, item := range pending {
		base := strings.ToUpper(strings.TrimSpace(item.BaseAsset))
		if base == "" {
			base = deriveBaseAsset(item.RawSymbol)
		}

		coin, ok := coinBySymbol[base]
		if !ok {
			if err := r.store.UpdateUnknownSymbol(ctx, item.Exchange, item.RawSymbol, "unresolvable", "", "no coin match by base asset"); err != nil {
				return UnknownResolveResult{}, fmt.Errorf("mark unknown symbol unresolvable %s:%s: %w", item.Exchange, item.RawSymbol, err)
			}
			result.Unresolvable++
			continue
		}

		mapping := Mapping{
			CoinID:         coin.CoinID,
			Exchange:       normalizeExchange(item.Exchange),
			ExchangeSymbol: normalizeSymbol(item.RawSymbol),
			BaseAsset:      base,
			QuoteAsset:     detectQuoteAsset(item.RawSymbol),
		}
		if mapping.QuoteAsset == "" {
			mapping.QuoteAsset = "USDT"
		}

		if err := r.store.UpsertMapping(ctx, mapping, "active"); err != nil {
			return UnknownResolveResult{}, fmt.Errorf("upsert resolved unknown mapping %s:%s: %w", item.Exchange, item.RawSymbol, err)
		}
		if err := r.store.UpdateUnknownSymbol(ctx, item.Exchange, item.RawSymbol, "resolved", coin.CoinID, "resolved by base asset symbol match"); err != nil {
			return UnknownResolveResult{}, fmt.Errorf("mark unknown symbol resolved %s:%s: %w", item.Exchange, item.RawSymbol, err)
		}
		result.Resolved++
	}

	return result, nil
}

func detectQuoteAsset(symbol string) string {
	symbol = normalizeSymbol(symbol)
	for _, quote := range []string{"USDT", "USDC", "BUSD", "BTC", "ETH"} {
		if strings.HasSuffix(symbol, quote) {
			return quote
		}
	}
	return ""
}
