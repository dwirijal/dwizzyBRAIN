package stablecoins

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	defaultAssetLimit    = 100
	defaultHistoryLimit  = 30
	defaultHistoryPoints = 365
	defaultDepegPct      = 0.01
)

type client interface {
	Assets(ctx context.Context) ([]Asset, error)
}

type store interface {
	LookupCoinID(ctx context.Context, asset Asset) (string, error)
	UpsertLatest(ctx context.Context, items []LatestRecord) error
	InsertHistory(ctx context.Context, records []HistoryRecord) error
}

type Service struct {
	client         client
	store          store
	assetLimit     int
	historyLimit   int
	historyPoints  int
	depegThreshold float64
	now            func() time.Time
}

func NewService(client client, store store, assetLimit, historyLimit, historyPoints int, depegThreshold float64) *Service {
	if assetLimit <= 0 {
		assetLimit = defaultAssetLimit
	}
	if historyLimit <= 0 {
		historyLimit = defaultHistoryLimit
	}
	if historyPoints <= 0 {
		historyPoints = defaultHistoryPoints
	}
	if depegThreshold <= 0 {
		depegThreshold = defaultDepegPct
	}
	return &Service{
		client:         client,
		store:          store,
		assetLimit:     assetLimit,
		historyLimit:   historyLimit,
		historyPoints:  historyPoints,
		depegThreshold: depegThreshold,
		now:            time.Now,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.client == nil {
		return Result{}, fmt.Errorf("stablecoin client is required")
	}
	if s.store == nil {
		return Result{}, fmt.Errorf("stablecoin store is required")
	}

	assets, err := s.client.Assets(ctx)
	if err != nil {
		return Result{}, err
	}
	sort.Slice(assets, func(i, j int) bool {
		if assets[i].Circulating.PeggedUSD == assets[j].Circulating.PeggedUSD {
			return assets[i].Symbol < assets[j].Symbol
		}
		return assets[i].Circulating.PeggedUSD > assets[j].Circulating.PeggedUSD
	})
	if len(assets) > s.assetLimit {
		assets = assets[:s.assetLimit]
	}

	now := s.now().UTC()
	latest := make([]LatestRecord, 0, len(assets))
	history := make([]HistoryRecord, 0, len(assets)*4)
	depegs := 0
	skipped := 0

	for _, asset := range assets {
		coinID, err := s.store.LookupCoinID(ctx, asset)
		if err != nil {
			return Result{}, err
		}
		if strings.TrimSpace(coinID) == "" {
			skipped++
			continue
		}

		balance := asset.Circulating.PeggedUSD
		latest = append(latest, LatestRecord{
			CoinID:             coinID,
			SnapshotDate:       now,
			PegType:            normalize(asset.PegType),
			PegMechanism:       normalize(asset.PegMechanism),
			PriceUSD:           asset.Price,
			MCAPUSD:            balance,
			Circulating:        balance,
			BackingComposition: chainComposition(asset.ChainCirculating, balance),
			SyncedAt:           now,
		})

		history = append(history, buildHistory(asset, coinID, now, s.historyPoints)...)

		if asset.Price != nil {
			depeg := absFloat(*asset.Price - 1.0)
			if depeg >= s.depegThreshold {
				depegs++
			}
		}
	}

	if err := s.store.UpsertLatest(ctx, latest); err != nil {
		return Result{}, err
	}
	if err := s.store.InsertHistory(ctx, history); err != nil {
		return Result{}, err
	}

	return Result{
		AssetsFetched:   len(assets),
		AssetsUpserted:  len(latest),
		HistoryRows:     len(history),
		DepegsDetected:  depegs,
		SkippedUnmapped: skipped,
	}, nil
}

func buildHistory(asset Asset, coinID string, now time.Time, maxPoints int) []HistoryRecord {
	points := []struct {
		offset time.Duration
		value  PeggedAmount
	}{
		{0, asset.Circulating},
		{-24 * time.Hour, asset.CirculatingPrevDay},
		{-7 * 24 * time.Hour, asset.CirculatingPrevWeek},
		{-30 * 24 * time.Hour, asset.CirculatingPrevMonth},
	}
	records := make([]HistoryRecord, 0, len(points))
	for _, point := range points {
		if point.value.PeggedUSD <= 0 {
			continue
		}
		records = append(records, HistoryRecord{
			Time:        now.Add(point.offset),
			CoinID:      coinID,
			MCAPUSD:     point.value.PeggedUSD,
			Circulating: floatPtr(point.value.PeggedUSD),
			PriceUSD:    asset.Price,
		})
	}
	return trimHistory(records, maxPoints)
}

func trimHistory(records []HistoryRecord, maxPoints int) []HistoryRecord {
	if maxPoints <= 0 || len(records) <= maxPoints {
		return records
	}
	start := len(records) - maxPoints
	if start < 0 {
		start = 0
	}
	return records[start:]
}

func chainComposition(chains map[string]ChainBalance, total float64) map[string]float64 {
	if len(chains) == 0 || total <= 0 {
		return map[string]float64{}
	}
	composition := make(map[string]float64, len(chains))
	for name, chain := range chains {
		if chain.Current.PeggedUSD <= 0 {
			continue
		}
		composition[name] = (chain.Current.PeggedUSD / total) * 100
	}
	return composition
}

func normalize(value string) string {
	return strings.TrimSpace(value)
}

func absFloat(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func floatPtr(value float64) *float64 {
	v := value
	return &v
}
