package news

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListActiveSources(ctx context.Context) ([]Source, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
    source_name, display_name, base_url, rss_url, logo_url,
    credibility_score, poll_interval_seconds, is_active, fetch_type,
    articles_fetched_total, last_fetched_at, last_success_at, consecutive_failures
FROM news_sources
WHERE is_active = TRUE AND fetch_type IN ('rss', 'telegram')
ORDER BY credibility_score DESC, source_name`)
	if err != nil {
		return nil, fmt.Errorf("list active news sources: %w", err)
	}
	defer rows.Close()

	sources := make([]Source, 0)
	for rows.Next() {
		var source Source
		if err := rows.Scan(
			&source.SourceName,
			&source.DisplayName,
			&source.BaseURL,
			&source.RSSURL,
			&source.LogoURL,
			&source.CredibilityScore,
			&source.PollIntervalSeconds,
			&source.IsActive,
			&source.FetchType,
			&source.ArticlesFetchedTotal,
			&source.LastFetchedAt,
			&source.LastSuccessAt,
			&source.ConsecutiveFailures,
		); err != nil {
			return nil, fmt.Errorf("scan news source: %w", err)
		}
		sources = append(sources, source)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate news sources: %w", err)
	}
	return sources, nil
}

func (s *Store) InsertArticles(ctx context.Context, articles []Article) (int, error) {
	if len(articles) == 0 {
		return 0, nil
	}

	inserted := 0
	for _, article := range articles {
		tag, err := s.pool.Exec(ctx, `
INSERT INTO news_articles (
    external_id, source, source_url, title, body_preview, full_url, image_url, author,
    published_at, fetched_at, cp_kind, cp_votes_positive, cp_votes_negative, cp_votes_important,
    is_processed, is_active
) VALUES (
    $1, $2::news_source_name, $3, $4, $5, $6, $7, $8,
    $9, $10, $11, $12, $13, $14,
    FALSE, TRUE
) ON CONFLICT (external_id, source) DO UPDATE SET
    source_url = EXCLUDED.source_url,
    title = EXCLUDED.title,
    body_preview = EXCLUDED.body_preview,
    full_url = EXCLUDED.full_url,
    image_url = EXCLUDED.image_url,
    author = EXCLUDED.author,
    published_at = EXCLUDED.published_at,
    fetched_at = EXCLUDED.fetched_at,
    cp_kind = EXCLUDED.cp_kind,
    cp_votes_positive = EXCLUDED.cp_votes_positive,
    cp_votes_negative = EXCLUDED.cp_votes_negative,
    cp_votes_important = EXCLUDED.cp_votes_important,
    is_active = TRUE`,
			article.ExternalID,
			strings.ToLower(strings.TrimSpace(article.Source)),
			article.SourceURL,
			article.Title,
			nullIfEmpty(article.BodyPreview),
			article.FullURL,
			nullIfEmpty(article.ImageURL),
			nullIfEmpty(article.Author),
			article.PublishedAt,
			article.FetchedAt,
			nullIfEmpty(article.CPKind),
			article.CPVotesPositive,
			article.CPVotesNegative,
			article.CPVotesImportant,
		)
		if err != nil {
			return inserted, fmt.Errorf("upsert news article %s: %w", article.ExternalID, err)
		}
		if tag.RowsAffected() > 0 {
			inserted++
		}
	}
	return inserted, nil
}

func (s *Store) MarkSourceSuccess(ctx context.Context, sourceName string, fetched int) error {
	_, err := s.pool.Exec(ctx, `
UPDATE news_sources
SET
    articles_fetched_total = articles_fetched_total + $2,
    last_fetched_at = NOW(),
    last_success_at = NOW(),
    consecutive_failures = 0,
    updated_at = NOW()
WHERE source_name = $1::news_source_name`,
		strings.ToLower(strings.TrimSpace(sourceName)),
		fetched,
	)
	if err != nil {
		return fmt.Errorf("mark source success %s: %w", sourceName, err)
	}
	return nil
}

func (s *Store) MarkSourceFailure(ctx context.Context, sourceName string) error {
	_, err := s.pool.Exec(ctx, `
UPDATE news_sources
SET
    last_fetched_at = NOW(),
    consecutive_failures = consecutive_failures + 1,
    updated_at = NOW()
WHERE source_name = $1::news_source_name`,
		strings.ToLower(strings.TrimSpace(sourceName)),
	)
	if err != nil {
		return fmt.Errorf("mark source failure %s: %w", sourceName, err)
	}
	return nil
}

func nullIfEmpty(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

var _ interface {
	ListActiveSources(context.Context) ([]Source, error)
	InsertArticles(context.Context, []Article) (int, error)
	MarkSourceSuccess(context.Context, string, int) error
	MarkSourceFailure(context.Context, string) error
} = (*Store)(nil)
