package news

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubFetcher struct {
	articles []Article
	err      error
}

func (s stubFetcher) Fetch(ctx context.Context, source Source) ([]Article, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]Article(nil), s.articles...), nil
}

type stubStore struct {
	sources       []Source
	inserted      int
	failSource    bool
	failedSources []string
}

func (s *stubStore) ListActiveSources(ctx context.Context) ([]Source, error) {
	return append([]Source(nil), s.sources...), nil
}
func (s *stubStore) InsertArticles(ctx context.Context, articles []Article) (int, error) {
	return len(articles), nil
}
func (s *stubStore) MarkSourceSuccess(ctx context.Context, sourceName string, fetched int) error {
	return nil
}
func (s *stubStore) MarkSourceFailure(ctx context.Context, sourceName string) error {
	s.failedSources = append(s.failedSources, sourceName)
	return nil
}

func TestServiceRunOnce(t *testing.T) {
	svc := NewService(stubFetcher{
		articles: []Article{{ExternalID: "1", Source: "coindesk", SourceURL: "https://example.com/1", Title: "Bitcoin Rallies", PublishedAt: time.Now(), FetchedAt: time.Now()}},
	}, &stubStore{
		sources: []Source{
			{SourceName: "coindesk", CredibilityScore: 0.9, FetchType: "rss", IsActive: true},
			{SourceName: "decrypt", CredibilityScore: 0.8, FetchType: "rss", IsActive: true},
		},
	}, []string{"coindesk"})

	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.SourcesProcessed != 1 {
		t.Fatalf("SourcesProcessed = %d, want 1", result.SourcesProcessed)
	}
	if result.ArticlesInserted != 1 {
		t.Fatalf("ArticlesInserted = %d, want 1", result.ArticlesInserted)
	}
}

func TestServiceRunOnceFetchErrorContinues(t *testing.T) {
	store := &stubStore{
		sources: []Source{
			{SourceName: "coindesk", CredibilityScore: 0.9, FetchType: "rss", IsActive: true},
		},
	}
	svc := NewService(stubFetcher{err: errors.New("boom")}, store, nil)
	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.Failures != 1 {
		t.Fatalf("Failures = %d, want 1", result.Failures)
	}
	if len(store.failedSources) != 1 || store.failedSources[0] != "coindesk" {
		t.Fatalf("failed sources = %#v", store.failedSources)
	}
}
