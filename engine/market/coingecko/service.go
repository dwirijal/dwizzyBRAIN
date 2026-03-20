package coingecko

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const defaultColdLoadInterval = 24 * time.Hour

type loader interface {
	LoadTopMarkets(ctx context.Context, pages, perPage int) ([]MarketCoin, error)
}

type coldStore interface {
	UpsertCoins(ctx context.Context, coins []MarketCoin) (int, error)
	UpsertColdCoinData(ctx context.Context, coins []MarketCoin) (int, error)
	LatestColdCoinIDs(ctx context.Context, limit int) ([]string, error)
}

type Result struct {
	CoinsInserted    int `json:"coins_inserted"`
	ColdRowsUpserted int `json:"cold_rows_upserted"`
}

type Service struct {
	loader  loader
	store   coldStore
	pages   int
	perPage int
	now     func() time.Time
}

func NewService(loader loader, store coldStore, pages, perPage int) *Service {
	if pages <= 0 {
		pages = defaultPageCount
	}
	if perPage <= 0 || perPage > 250 {
		perPage = defaultPageSize
	}
	return &Service{
		loader:  loader,
		store:   store,
		pages:   pages,
		perPage: perPage,
		now:     time.Now,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.loader == nil {
		return Result{}, fmt.Errorf("coingecko loader is required")
	}
	if s.store == nil {
		return Result{}, fmt.Errorf("coingecko store is required")
	}

	coins, err := s.loader.LoadTopMarkets(ctx, s.pages, s.perPage)
	if err != nil {
		return Result{}, err
	}
	if len(coins) == 0 {
		return Result{}, nil
	}

	inserted, err := s.store.UpsertCoins(ctx, coins)
	if err != nil {
		return Result{}, err
	}
	rows, err := s.store.UpsertColdCoinData(ctx, coins)
	if err != nil {
		return Result{}, err
	}

	return Result{CoinsInserted: inserted, ColdRowsUpserted: rows}, nil
}

func (s *Service) LoadTopCoinIDs(ctx context.Context, limit int) ([]string, error) {
	if s.store == nil {
		return nil, fmt.Errorf("coingecko store is required")
	}
	return s.store.LatestColdCoinIDs(ctx, limit)
}

func normalizeCoinIDs(raw []string) []string {
	ids := make([]string, 0, len(raw))
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		ids = append(ids, item)
	}
	return ids
}

type clocked interface {
	Now() time.Time
}
