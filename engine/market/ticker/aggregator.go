package ticker

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"dwizzyBRAIN/shared/schema"
)

const defaultMaxAge = 15 * time.Second

type ExchangeTicker struct {
	Exchange   string    `json:"exchange"`
	Symbol     string    `json:"symbol"`
	Price      float64   `json:"price"`
	Bid        float64   `json:"bid"`
	Ask        float64   `json:"ask"`
	Volume     float64   `json:"volume"`
	Timestamp  time.Time `json:"timestamp"`
	SpreadPct  float64   `json:"spread_pct"`
	IsStale    bool      `json:"is_stale"`
	BaseAsset  string    `json:"base_asset"`
	QuoteAsset string    `json:"quote_asset"`
}

type Snapshot struct {
	CoinID                 string           `json:"coin_id"`
	BaseAsset              string           `json:"base_asset"`
	QuoteAsset             string           `json:"quote_asset"`
	LastUpdatedAt          time.Time        `json:"last_updated_at"`
	BestBid                float64          `json:"best_bid"`
	BestBidExchange        string           `json:"best_bid_exchange"`
	BestAsk                float64          `json:"best_ask"`
	BestAskExchange        string           `json:"best_ask_exchange"`
	CrossExchangeSpreadPct float64          `json:"cross_exchange_spread_pct"`
	ExchangeCount          int              `json:"exchange_count"`
	AvailableExchanges     []ExchangeTicker `json:"available_exchanges"`
}

type Aggregator struct {
	mu     sync.RWMutex
	latest map[string]map[string]schema.ResolvedTicker
	maxAge time.Duration
	now    func() time.Time
}

func NewAggregator() *Aggregator {
	return &Aggregator{
		latest: make(map[string]map[string]schema.ResolvedTicker),
		maxAge: defaultMaxAge,
		now:    time.Now,
	}
}

func (a *Aggregator) Update(ticker schema.ResolvedTicker) (Snapshot, error) {
	if err := ticker.Validate(); err != nil {
		return Snapshot{}, fmt.Errorf("validate resolved ticker: %w", err)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	coinID := strings.TrimSpace(ticker.CoinID)
	exchange := strings.ToLower(strings.TrimSpace(ticker.Exchange))
	if a.latest[coinID] == nil {
		a.latest[coinID] = make(map[string]schema.ResolvedTicker)
	}

	current, ok := a.latest[coinID][exchange]
	if ok && current.Timestamp.After(ticker.Timestamp) {
		return a.snapshotLocked(coinID), nil
	}

	a.latest[coinID][exchange] = ticker
	return a.snapshotLocked(coinID), nil
}

func (a *Aggregator) Snapshot(coinID string) (Snapshot, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	coinID = strings.TrimSpace(coinID)
	if _, ok := a.latest[coinID]; !ok {
		return Snapshot{}, false
	}

	return a.snapshotLocked(coinID), true
}

func (a *Aggregator) Snapshots() []Snapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	coinIDs := make([]string, 0, len(a.latest))
	for coinID := range a.latest {
		coinIDs = append(coinIDs, coinID)
	}
	sort.Strings(coinIDs)

	snapshots := make([]Snapshot, 0, len(coinIDs))
	for _, coinID := range coinIDs {
		snapshots = append(snapshots, a.snapshotLocked(coinID))
	}

	return snapshots
}

func (a *Aggregator) snapshotLocked(coinID string) Snapshot {
	exchangeMap := a.latest[coinID]
	now := a.now().UTC()

	snapshot := Snapshot{
		CoinID:             coinID,
		AvailableExchanges: make([]ExchangeTicker, 0, len(exchangeMap)),
	}

	bestBid := 0.0
	bestAsk := 0.0
	for _, ticker := range exchangeMap {
		isStale := a.maxAge > 0 && ticker.Timestamp.Before(now.Add(-a.maxAge))
		if snapshot.BaseAsset == "" {
			snapshot.BaseAsset = ticker.BaseAsset
		}
		if snapshot.QuoteAsset == "" {
			snapshot.QuoteAsset = ticker.QuoteAsset
		}
		if ticker.Timestamp.After(snapshot.LastUpdatedAt) {
			snapshot.LastUpdatedAt = ticker.Timestamp
		}

		entry := ExchangeTicker{
			Exchange:   ticker.Exchange,
			Symbol:     ticker.Symbol,
			Price:      ticker.Price,
			Bid:        ticker.Bid,
			Ask:        ticker.Ask,
			Volume:     ticker.Volume,
			Timestamp:  ticker.Timestamp,
			SpreadPct:  intraExchangeSpreadPct(ticker.Bid, ticker.Ask),
			IsStale:    isStale,
			BaseAsset:  ticker.BaseAsset,
			QuoteAsset: ticker.QuoteAsset,
		}
		snapshot.AvailableExchanges = append(snapshot.AvailableExchanges, entry)

		if isStale {
			continue
		}
		if ticker.Bid > bestBid {
			bestBid = ticker.Bid
			snapshot.BestBid = ticker.Bid
			snapshot.BestBidExchange = ticker.Exchange
		}
		if ticker.Ask > 0 && (bestAsk == 0 || ticker.Ask < bestAsk) {
			bestAsk = ticker.Ask
			snapshot.BestAsk = ticker.Ask
			snapshot.BestAskExchange = ticker.Exchange
		}
	}

	sort.Slice(snapshot.AvailableExchanges, func(i, j int) bool {
		return snapshot.AvailableExchanges[i].Exchange < snapshot.AvailableExchanges[j].Exchange
	})

	snapshot.ExchangeCount = len(snapshot.AvailableExchanges)
	if snapshot.BestBid > 0 && snapshot.BestAsk > 0 && snapshot.BestBidExchange != "" && snapshot.BestAskExchange != "" {
		snapshot.CrossExchangeSpreadPct = ((snapshot.BestBid - snapshot.BestAsk) / snapshot.BestAsk) * 100
	}

	return snapshot
}

func intraExchangeSpreadPct(bid, ask float64) float64 {
	if bid <= 0 || ask <= 0 || ask < bid {
		return 0
	}
	return ((ask - bid) / bid) * 100
}
