package arbitrage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dwizzyBRAIN/engine/market/ticker"

	redis "github.com/redis/go-redis/v9"
)

const defaultScanInterval = 5 * time.Second

type Opportunity struct {
	CoinID         string
	Symbol         string
	BuyExchange    string
	SellExchange   string
	BuyPrice       float64
	SellPrice      float64
	GrossSpreadPct float64
	BuyDepthUSD    float64
	SellDepthUSD   float64
	DetectedAt     time.Time
	IsProfitable   bool
}

type snapshotSource interface {
	Snapshots() []ticker.Snapshot
}

type configSource interface {
	Get(ctx context.Context, coinID string) (Config, error)
}

type signalStore interface {
	Insert(ctx context.Context, opportunity Opportunity) error
}

type Engine struct {
	source   snapshotSource
	configs  configSource
	store    signalStore
	cache    redis.Cmdable
	now      func() time.Time
	interval time.Duration
}

func NewEngine(source snapshotSource, configs configSource, store signalStore, cache redis.Cmdable) *Engine {
	return &Engine{
		source:   source,
		configs:  configs,
		store:    store,
		cache:    cache,
		now:      time.Now,
		interval: defaultScanInterval,
	}
}

func (e *Engine) Scan(ctx context.Context) ([]Opportunity, error) {
	if e.source == nil {
		return nil, fmt.Errorf("snapshot source is required")
	}
	if e.configs == nil {
		return nil, fmt.Errorf("config source is required")
	}
	if e.store == nil {
		return nil, fmt.Errorf("signal store is required")
	}

	var out []Opportunity
	for _, snapshot := range e.source.Snapshots() {
		cfg, err := e.configs.Get(ctx, snapshot.CoinID)
		if err != nil {
			return nil, fmt.Errorf("load config for %s: %w", snapshot.CoinID, err)
		}
		if !cfg.IsEnabled {
			continue
		}

		opp, ok := detectOpportunity(snapshot, cfg, e.now().UTC())
		if !ok {
			continue
		}
		if e.onCooldown(ctx, opp, cfg.CooldownSeconds) {
			continue
		}
		if err := e.store.Insert(ctx, opp); err != nil {
			return nil, fmt.Errorf("insert arbitrage opportunity for %s: %w", snapshot.CoinID, err)
		}
		e.markCooldown(ctx, opp, cfg.CooldownSeconds)
		out = append(out, opp)
	}

	return out, nil
}

func detectOpportunity(snapshot ticker.Snapshot, cfg Config, now time.Time) (Opportunity, bool) {
	if snapshot.BestAsk <= 0 || snapshot.BestBid <= 0 {
		return Opportunity{}, false
	}
	if snapshot.BestBidExchange == "" || snapshot.BestAskExchange == "" || snapshot.BestBidExchange == snapshot.BestAskExchange {
		return Opportunity{}, false
	}
	if snapshot.CrossExchangeSpreadPct < cfg.MinSpreadPct {
		return Opportunity{}, false
	}

	buy, sell, ok := findSides(snapshot)
	if !ok {
		return Opportunity{}, false
	}
	buyDepth := buy.Ask * buy.Volume
	sellDepth := sell.Bid * sell.Volume
	if buyDepth < cfg.MinDepthUSD || sellDepth < cfg.MinDepthUSD {
		return Opportunity{}, false
	}

	return Opportunity{
		CoinID:         snapshot.CoinID,
		Symbol:         buy.Symbol,
		BuyExchange:    buy.Exchange,
		SellExchange:   sell.Exchange,
		BuyPrice:       buy.Ask,
		SellPrice:      sell.Bid,
		GrossSpreadPct: snapshot.CrossExchangeSpreadPct,
		BuyDepthUSD:    buyDepth,
		SellDepthUSD:   sellDepth,
		DetectedAt:     now,
		IsProfitable:   snapshot.CrossExchangeSpreadPct >= cfg.MinSpreadPct,
	}, true
}

func findSides(snapshot ticker.Snapshot) (ticker.ExchangeTicker, ticker.ExchangeTicker, bool) {
	var buy ticker.ExchangeTicker
	var sell ticker.ExchangeTicker
	var buyOK, sellOK bool
	for _, exchange := range snapshot.AvailableExchanges {
		if exchange.IsStale {
			continue
		}
		if exchange.Exchange == snapshot.BestAskExchange {
			buy = exchange
			buyOK = true
		}
		if exchange.Exchange == snapshot.BestBidExchange {
			sell = exchange
			sellOK = true
		}
	}
	return buy, sell, buyOK && sellOK
}

func (e *Engine) onCooldown(ctx context.Context, opportunity Opportunity, seconds int) bool {
	if e.cache == nil || seconds <= 0 {
		return false
	}
	key := cooldownKey(opportunity)
	exists, err := e.cache.Exists(ctx, key).Result()
	return err == nil && exists > 0
}

func (e *Engine) markCooldown(ctx context.Context, opportunity Opportunity, seconds int) {
	if e.cache == nil || seconds <= 0 {
		return
	}
	_ = e.cache.Set(ctx, cooldownKey(opportunity), "1", time.Duration(seconds)*time.Second).Err()
}

func cooldownKey(opportunity Opportunity) string {
	return "arb:cooldown:" +
		strings.TrimSpace(opportunity.CoinID) + ":" +
		strings.ToLower(strings.TrimSpace(opportunity.BuyExchange)) + ":" +
		strings.ToLower(strings.TrimSpace(opportunity.SellExchange))
}
