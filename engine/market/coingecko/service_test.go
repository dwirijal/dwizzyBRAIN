package coingecko

import (
	"context"
	"testing"
	"time"
)

type stubLoader struct {
	coins []MarketCoin
	err   error
}

func (s stubLoader) LoadTopMarkets(ctx context.Context, pages, perPage int) ([]MarketCoin, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.coins, nil
}

type stubStore struct {
	coins []MarketCoin
}

func (s *stubStore) UpsertCoins(ctx context.Context, coins []MarketCoin) (int, error) {
	s.coins = append(s.coins, coins...)
	return len(coins), nil
}

func (s *stubStore) UpsertColdCoinData(ctx context.Context, coins []MarketCoin) (int, error) {
	return len(coins), nil
}

func (s *stubStore) LatestColdCoinIDs(ctx context.Context, limit int) ([]string, error) {
	return nil, nil
}

func TestServiceRunOnce(t *testing.T) {
	service := NewService(stubLoader{
		coins: []MarketCoin{
			{ID: "bitcoin", Symbol: "btc", Name: "Bitcoin", MarketCapRank: intPtr(1)},
			{ID: "ethereum", Symbol: "eth", Name: "Ethereum", MarketCapRank: intPtr(2)},
		},
	}, &stubStore{}, 4, 250)
	service.now = func() time.Time { return time.Date(2026, 3, 18, 0, 0, 0, 0, time.UTC) }

	result, err := service.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() returned error: %v", err)
	}
	if result.CoinsInserted != 2 || result.ColdRowsUpserted != 2 {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func intPtr(v int) *int { return &v }
