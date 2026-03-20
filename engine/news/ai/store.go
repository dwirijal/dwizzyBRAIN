package ai

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) ListPendingArticles(ctx context.Context, limit int) ([]Article, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
SELECT
    a.id, a.external_id, a.source, COALESCE(s.credibility_score, 0.5),
    a.source_url, a.title, COALESCE(a.body_preview, ''), COALESCE(a.full_url, ''),
    COALESCE(a.image_url, ''), COALESCE(a.author, ''), a.published_at, a.fetched_at,
    COALESCE(a.cp_kind, ''), a.cp_votes_positive, a.cp_votes_negative, a.cp_votes_important
FROM news_articles a
LEFT JOIN news_sources s ON s.source_name = a.source
WHERE a.is_processed = FALSE AND a.is_active = TRUE
ORDER BY a.published_at ASC, a.id ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending news articles: %w", err)
	}
	defer rows.Close()

	items := make([]Article, 0)
	for rows.Next() {
		var item Article
		if err := rows.Scan(
			&item.ID,
			&item.ExternalID,
			&item.Source,
			&item.SourceCredibility,
			&item.SourceURL,
			&item.Title,
			&item.BodyPreview,
			&item.FullURL,
			&item.ImageURL,
			&item.Author,
			&item.PublishedAt,
			&item.FetchedAt,
			&item.CPKind,
			&item.CPVotesPositive,
			&item.CPVotesNegative,
			&item.CPVotesImportant,
		); err != nil {
			return nil, fmt.Errorf("scan pending news article: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending news articles: %w", err)
	}

	return items, nil
}

func (s *Store) LoadCoins(ctx context.Context) ([]CoinEntity, error) {
	rows, err := s.pool.Query(ctx, `
SELECT
    COALESCE(NULLIF(coin_id, ''), id) AS coin_id,
    COALESCE(NULLIF(symbol, ''), '') AS symbol,
    COALESCE(NULLIF(name, ''), '') AS name
FROM coins
WHERE COALESCE(is_active, TRUE) = TRUE
ORDER BY COALESCE(rank, market_cap_rank, 999999), COALESCE(NULLIF(coin_id, ''), id)`)
	if err != nil {
		return nil, fmt.Errorf("load coins for news ai: %w", err)
	}
	defer rows.Close()

	items := make([]CoinEntity, 0)
	for rows.Next() {
		var item CoinEntity
		if err := rows.Scan(&item.CoinID, &item.Symbol, &item.Name); err != nil {
			return nil, fmt.Errorf("scan coin entity: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate coin entities: %w", err)
	}
	return items, nil
}

func (s *Store) LoadProtocols(ctx context.Context) ([]ProtocolEntity, error) {
	rows, err := s.pool.Query(ctx, `
SELECT COALESCE(NULLIF(slug, ''), '') AS slug, COALESCE(NULLIF(name, ''), '') AS name
FROM defi_protocols
WHERE COALESCE(NULLIF(slug, ''), '') <> ''
ORDER BY slug`)
	if err != nil {
		return nil, fmt.Errorf("load defi protocols for news ai: %w", err)
	}
	defer rows.Close()

	items := make([]ProtocolEntity, 0)
	for rows.Next() {
		var item ProtocolEntity
		if err := rows.Scan(&item.Slug, &item.Name); err != nil {
			return nil, fmt.Errorf("scan protocol entity: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate protocol entities: %w", err)
	}
	return items, nil
}

func (s *Store) UpsertMetadata(ctx context.Context, metadata Metadata) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO news_ai_metadata (
    article_id, summary_short, summary_long, key_points, sentiment, sentiment_score,
    category, subcategory, importance_score, is_breaking, breaking_type, model_used,
    processing_latency_ms, processed_at
) VALUES (
    $1, $2, $3, $4, $5::news_sentiment, $6, $7::news_category, $8, $9, $10, $11, $12, $13, NOW()
)
ON CONFLICT (article_id) DO UPDATE SET
    summary_short = EXCLUDED.summary_short,
    summary_long = EXCLUDED.summary_long,
    key_points = EXCLUDED.key_points,
    sentiment = EXCLUDED.sentiment,
    sentiment_score = EXCLUDED.sentiment_score,
    category = EXCLUDED.category,
    subcategory = EXCLUDED.subcategory,
    importance_score = EXCLUDED.importance_score,
    is_breaking = EXCLUDED.is_breaking,
    breaking_type = EXCLUDED.breaking_type,
    model_used = EXCLUDED.model_used,
    processing_latency_ms = EXCLUDED.processing_latency_ms,
    processed_at = NOW()`,
		metadata.ArticleID,
		metadata.SummaryShort,
		metadata.SummaryLong,
		metadata.KeyPoints,
		metadata.Sentiment,
		metadata.SentimentScore,
		metadata.Category,
		metadata.Subcategory,
		metadata.ImportanceScore,
		metadata.IsBreaking,
		nullIfEmpty(metadata.BreakingType),
		metadata.ModelUsed,
		metadata.ProcessingLatencyMS,
	)
	if err != nil {
		return fmt.Errorf("upsert news ai metadata article=%d: %w", metadata.ArticleID, err)
	}
	return nil
}

func (s *Store) ReplaceEntities(ctx context.Context, articleID int64, entities []Entity) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin news ai entity tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM news_entities WHERE article_id = $1`, articleID); err != nil {
		return fmt.Errorf("delete news entities article=%d: %w", articleID, err)
	}
	for _, entity := range entities {
		_, err := tx.Exec(ctx, `
INSERT INTO news_entities (
    article_id, coin_id, llama_slug, entity_type, entity_name, is_primary, mention_count, confidence
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			articleID,
			nullIfEmpty(entity.CoinID),
			nullIfEmpty(entity.LlamaSlug),
			entity.EntityType,
			nullIfEmpty(entity.EntityName),
			entity.IsPrimary,
			entity.MentionCount,
			entity.Confidence,
		)
		if err != nil {
			return fmt.Errorf("insert news entity article=%d: %w", articleID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit news ai entity tx: %w", err)
	}
	return nil
}

func (s *Store) MarkProcessed(ctx context.Context, articleID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE news_articles SET is_processed = TRUE WHERE id = $1`, articleID)
	if err != nil {
		return fmt.Errorf("mark article processed %d: %w", articleID, err)
	}
	return nil
}

func (s *Store) MarkBatchProcessed(ctx context.Context, articleIDs []int64) error {
	if len(articleIDs) == 0 {
		return nil
	}
	for _, id := range articleIDs {
		if err := s.MarkProcessed(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

var _ interface {
	ListPendingArticles(context.Context, int) ([]Article, error)
	LoadCoins(context.Context) ([]CoinEntity, error)
	LoadProtocols(context.Context) ([]ProtocolEntity, error)
	UpsertMetadata(context.Context, Metadata) error
	ReplaceEntities(context.Context, int64, []Entity) error
	MarkBatchProcessed(context.Context, []int64) error
} = (*Store)(nil)

func nullIfEmpty(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}

var _ = sql.NullString{}
var _ = pgx.ErrNoRows
