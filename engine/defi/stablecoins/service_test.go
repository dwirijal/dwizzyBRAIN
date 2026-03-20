package stablecoins

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	lookups []string
	latest  []LatestRecord
	history []HistoryRecord
}

func (f *fakeStore) LookupCoinID(ctx context.Context, asset Asset) (string, error) {
	f.lookups = append(f.lookups, asset.Symbol)
	if asset.Symbol == "USDT" {
		return "tether", nil
	}
	return "", nil
}

func (f *fakeStore) UpsertLatest(ctx context.Context, items []LatestRecord) error {
	f.latest = append(f.latest, items...)
	return nil
}

func (f *fakeStore) InsertHistory(ctx context.Context, records []HistoryRecord) error {
	f.history = append(f.history, records...)
	return nil
}

type fakeClient struct {
	assets []Asset
}

func (f *fakeClient) Assets(ctx context.Context) ([]Asset, error) { return f.assets, nil }

func TestServiceRunOnce(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	service := NewService(
		&fakeClient{
			assets: []Asset{
				{
					ID:                   "1",
					Name:                 "Tether",
					Symbol:               "USDT",
					GeckoID:              "tether",
					PegType:              "peggedUSD",
					PegMechanism:         "fiat-backed",
					Circulating:          PeggedAmount{PeggedUSD: 100},
					CirculatingPrevDay:   PeggedAmount{PeggedUSD: 99},
					CirculatingPrevWeek:  PeggedAmount{PeggedUSD: 98},
					CirculatingPrevMonth: PeggedAmount{PeggedUSD: 97},
					ChainCirculating: map[string]ChainBalance{
						"Ethereum": {Current: PeggedAmount{PeggedUSD: 60}},
					},
					Price: ptr(1.01),
				},
				{
					ID:          "2",
					Name:        "Unknown",
					Symbol:      "ZZZ",
					GeckoID:     "zzz",
					Circulating: PeggedAmount{PeggedUSD: 10},
				},
			},
		},
		&fakeStore{},
		10,
		10,
		10,
		0.01,
	)
	service.now = func() time.Time { return now }

	result, err := service.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if result.AssetsFetched != 2 || result.AssetsUpserted != 1 || result.SkippedUnmapped != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.DepegsDetected != 1 {
		t.Fatalf("unexpected depeg count: %#v", result)
	}
}

func ptr(v float64) *float64 { return &v }
