package coverage

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Coin struct {
	CoinID string
	Rank   int
}

type Coverage struct {
	CoinID            string
	Tier              string
	OnBinance         bool
	OnBybit           bool
	OnOKX             bool
	OnKucoin          bool
	OnGate            bool
	OnKraken          bool
	OnMexc            bool
	OnHtx             bool
	OnCoinpaprika     bool
	IsDexOnly         bool
	BinanceVerifiedAt *time.Time
	BybitVerifiedAt   *time.Time
	AssignedAt        time.Time
	UpdatedAt         time.Time
}

type Store interface {
	ListCoins(ctx context.Context) ([]Coin, error)
	ListExchangeCoverage(ctx context.Context) (map[string]map[string]time.Time, error)
	UpsertCoverage(ctx context.Context, coverage Coverage) error
}

type Result struct {
	Processed int
}

type GapDetector struct {
	store Store
	now   func() time.Time
}

func NewGapDetector(store Store) *GapDetector {
	return &GapDetector{
		store: store,
		now:   time.Now,
	}
}

func (d *GapDetector) DetectAll(ctx context.Context) (Result, error) {
	if d.store == nil {
		return Result{}, fmt.Errorf("coverage store is required")
	}

	coins, err := d.store.ListCoins(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("list coins: %w", err)
	}
	exchangeCoverage, err := d.store.ListExchangeCoverage(ctx)
	if err != nil {
		return Result{}, fmt.Errorf("list exchange coverage: %w", err)
	}

	now := d.now().UTC()
	result := Result{}
	for _, coin := range coins {
		coverage := buildCoverage(now, coin, exchangeCoverage[coin.CoinID])
		if err := d.store.UpsertCoverage(ctx, coverage); err != nil {
			return result, fmt.Errorf("upsert coverage for %s: %w", coin.CoinID, err)
		}
		result.Processed++
	}

	return result, nil
}

func buildCoverage(now time.Time, coin Coin, exchanges map[string]time.Time) Coverage {
	coverage := Coverage{
		CoinID:     strings.TrimSpace(coin.CoinID),
		AssignedAt: now,
		UpdatedAt:  now,
	}

	var cexCount int
	if ts, ok := exchanges["binance"]; ok {
		coverage.OnBinance = true
		coverage.BinanceVerifiedAt = timePtr(ts)
		cexCount++
	}
	if ts, ok := exchanges["bybit"]; ok {
		coverage.OnBybit = true
		coverage.BybitVerifiedAt = timePtr(ts)
		cexCount++
	}
	if _, ok := exchanges["okx"]; ok {
		coverage.OnOKX = true
		cexCount++
	}
	if _, ok := exchanges["kucoin"]; ok {
		coverage.OnKucoin = true
		cexCount++
	}
	if _, ok := exchanges["gateio"]; ok {
		coverage.OnGate = true
		cexCount++
	}
	if _, ok := exchanges["kraken"]; ok {
		coverage.OnKraken = true
		cexCount++
	}
	if _, ok := exchanges["mexc"]; ok {
		coverage.OnMexc = true
		cexCount++
	}
	if _, ok := exchanges["htx"]; ok {
		coverage.OnHtx = true
		cexCount++
	}
	if _, ok := exchanges["coinpaprika"]; ok {
		coverage.OnCoinpaprika = true
	}

	coverage.Tier = assignTier(coin.Rank, cexCount, coverage.OnCoinpaprika)
	return coverage
}

func assignTier(rank, cexCount int, hasFallback bool) string {
	switch {
	case rank > 0 && rank <= 100 && cexCount >= 2:
		return "A"
	case rank > 0 && rank <= 500 && (cexCount >= 1 || hasFallback):
		return "B"
	case rank > 0 && rank <= 1000:
		return "C"
	default:
		return "D"
	}
}

func timePtr(v time.Time) *time.Time {
	value := v.UTC()
	return &value
}
