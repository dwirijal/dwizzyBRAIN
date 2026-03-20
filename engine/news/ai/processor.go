package ai

import (
	"context"
	"fmt"
	"time"
)

type store interface {
	ListPendingArticles(context.Context, int) ([]Article, error)
	LoadCoins(context.Context) ([]CoinEntity, error)
	LoadProtocols(context.Context) ([]ProtocolEntity, error)
	UpsertMetadata(context.Context, Metadata) error
	ReplaceEntities(context.Context, int64, []Entity) error
	MarkBatchProcessed(context.Context, []int64) error
}

type Service struct {
	store      store
	batchLimit int
	now        func() time.Time
}

func NewService(store store, batchLimit int) *Service {
	if batchLimit <= 0 {
		batchLimit = 25
	}
	return &Service{
		store:      store,
		batchLimit: batchLimit,
		now:        time.Now,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.store == nil {
		return Result{}, fmt.Errorf("news ai store is required")
	}

	articles, err := s.store.ListPendingArticles(ctx, s.batchLimit)
	if err != nil {
		return Result{}, err
	}
	if len(articles) == 0 {
		return Result{}, nil
	}

	coins, err := s.store.LoadCoins(ctx)
	if err != nil {
		return Result{}, err
	}
	protocols, err := s.store.LoadProtocols(ctx)
	if err != nil {
		return Result{}, err
	}

	processedIDs := make([]int64, 0, len(articles))
	result := Result{}

	for _, article := range articles {
		start := s.now()
		analyzed := analyzeArticle(article, coins, protocols)
		analyzed.metadata.ProcessingLatencyMS = int(time.Since(start).Milliseconds())

		if err := s.store.UpsertMetadata(ctx, analyzed.metadata); err != nil {
			result.Failures++
			result.FailedArticles = append(result.FailedArticles, article.ID)
			continue
		}
		if err := s.store.ReplaceEntities(ctx, article.ID, analyzed.entities); err != nil {
			result.Failures++
			result.FailedArticles = append(result.FailedArticles, article.ID)
			continue
		}

		processedIDs = append(processedIDs, article.ID)
		result.ArticlesProcessed++
		result.MetadataUpserted++
		result.EntitiesUpserted += len(analyzed.entities)
	}

	if err := s.store.MarkBatchProcessed(ctx, processedIDs); err != nil {
		return result, err
	}

	return result, nil
}
