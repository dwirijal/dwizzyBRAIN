package ai

import (
	"context"
	"testing"
	"time"
)

type stubStore struct {
	articles  []Article
	coins     []CoinEntity
	protocols []ProtocolEntity
	upserts   int
	entities  int
	processed []int64
}

func (s *stubStore) ListPendingArticles(ctx context.Context, limit int) ([]Article, error) {
	return append([]Article(nil), s.articles...), nil
}
func (s *stubStore) LoadCoins(ctx context.Context) ([]CoinEntity, error) {
	return append([]CoinEntity(nil), s.coins...), nil
}
func (s *stubStore) LoadProtocols(ctx context.Context) ([]ProtocolEntity, error) {
	return append([]ProtocolEntity(nil), s.protocols...), nil
}
func (s *stubStore) UpsertMetadata(ctx context.Context, metadata Metadata) error {
	s.upserts++
	return nil
}
func (s *stubStore) ReplaceEntities(ctx context.Context, articleID int64, entities []Entity) error {
	s.entities += len(entities)
	s.processed = append(s.processed, articleID)
	return nil
}
func (s *stubStore) MarkBatchProcessed(ctx context.Context, articleIDs []int64) error {
	s.processed = append(s.processed, articleIDs...)
	return nil
}

func TestAnalyzeArticle(t *testing.T) {
	analyzed := analyzeArticle(Article{
		ID:                1,
		Source:            "coindesk",
		SourceCredibility: 0.9,
		Title:             "Bitcoin rallies after ETF approval",
		BodyPreview:       "Bitcoin rose sharply after approval from regulators. Ethereum also gained.",
		CPVotesPositive:   12,
		CPVotesImportant:  4,
	}, []CoinEntity{
		{CoinID: "bitcoin", Symbol: "BTC", Name: "Bitcoin"},
		{CoinID: "ethereum", Symbol: "ETH", Name: "Ethereum"},
	}, []ProtocolEntity{
		{Slug: "lido", Name: "Lido"},
	})

	if analyzed.metadata.Sentiment != "bullish" {
		t.Fatalf("sentiment = %q, want bullish", analyzed.metadata.Sentiment)
	}
	if analyzed.metadata.Category != "regulation" {
		t.Fatalf("category = %q, want regulation", analyzed.metadata.Category)
	}
	if !analyzed.metadata.IsBreaking {
		t.Fatalf("expected breaking article")
	}
	if len(analyzed.entities) < 2 {
		t.Fatalf("expected at least 2 entities, got %d", len(analyzed.entities))
	}
}

func TestServiceRunOnce(t *testing.T) {
	store := &stubStore{
		articles: []Article{
			{
				ID:                1,
				Source:            "coindesk",
				SourceCredibility: 0.9,
				Title:             "Bitcoin rallies after ETF approval",
				BodyPreview:       "Bitcoin rose sharply after approval from regulators.",
				PublishedAt:       time.Now(),
				FetchedAt:         time.Now(),
			},
		},
		coins: []CoinEntity{{CoinID: "bitcoin", Symbol: "BTC", Name: "Bitcoin"}},
	}
	svc := NewService(store, 10)
	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.ArticlesProcessed != 1 {
		t.Fatalf("ArticlesProcessed = %d, want 1", result.ArticlesProcessed)
	}
	if store.upserts != 1 {
		t.Fatalf("metadata upserts = %d, want 1", store.upserts)
	}
}
