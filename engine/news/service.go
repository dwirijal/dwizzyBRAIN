package news

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type fetcher interface {
	Fetch(ctx context.Context, source Source) ([]Article, error)
}

type store interface {
	ListActiveSources(ctx context.Context) ([]Source, error)
	InsertArticles(ctx context.Context, articles []Article) (int, error)
	MarkSourceSuccess(ctx context.Context, sourceName string, fetched int) error
	MarkSourceFailure(ctx context.Context, sourceName string) error
}

type Service struct {
	fetcher      fetcher
	store        store
	sourceFilter map[string]struct{}
	now          func() time.Time
}

func NewService(fetcher fetcher, store store, sourceFilter []string) *Service {
	filter := make(map[string]struct{}, len(sourceFilter))
	for _, source := range sourceFilter {
		source = strings.ToLower(strings.TrimSpace(source))
		if source == "" {
			continue
		}
		filter[source] = struct{}{}
	}
	return &Service{
		fetcher:      fetcher,
		store:        store,
		sourceFilter: filter,
		now:          time.Now,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.fetcher == nil {
		return Result{}, fmt.Errorf("news fetcher is required")
	}
	if s.store == nil {
		return Result{}, fmt.Errorf("news store is required")
	}

	sources, err := s.store.ListActiveSources(ctx)
	if err != nil {
		return Result{}, err
	}
	sources = s.filterSources(sources)
	sort.SliceStable(sources, func(i, j int) bool {
		if sources[i].CredibilityScore == sources[j].CredibilityScore {
			return sources[i].SourceName < sources[j].SourceName
		}
		return sources[i].CredibilityScore > sources[j].CredibilityScore
	})

	result := Result{}
	for _, source := range sources {
		articles, err := s.fetcher.Fetch(ctx, source)
		if err != nil {
			result.Failures++
			result.FailedSources = append(result.FailedSources, source.SourceName)
			_ = s.store.MarkSourceFailure(ctx, source.SourceName)
			continue
		}

		inserted, err := s.store.InsertArticles(ctx, articles)
		if err != nil {
			result.Failures++
			result.FailedSources = append(result.FailedSources, source.SourceName)
			_ = s.store.MarkSourceFailure(ctx, source.SourceName)
			continue
		}

		if err := s.store.MarkSourceSuccess(ctx, source.SourceName, len(articles)); err != nil {
			return result, err
		}

		result.SourcesProcessed++
		result.ArticlesFetched += len(articles)
		result.ArticlesInserted += inserted
	}

	return result, nil
}

func (s *Service) filterSources(sources []Source) []Source {
	if len(s.sourceFilter) == 0 {
		return sources
	}
	filtered := make([]Source, 0, len(sources))
	for _, source := range sources {
		if _, ok := s.sourceFilter[strings.ToLower(strings.TrimSpace(source.SourceName))]; !ok {
			continue
		}
		filtered = append(filtered, source)
	}
	return filtered
}
