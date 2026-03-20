package impact

import (
	"context"
	"database/sql"
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

func (s *Store) ListCandidates(ctx context.Context, limit int) ([]Candidate, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
SELECT DISTINCT ON (a.id, e.coin_id)
    a.id,
    COALESCE(NULLIF(e.coin_id, ''), '') AS coin_id,
    a.published_at,
    COALESCE(m.sentiment::text, 'neutral'),
    m.importance_score,
    COALESCE(m.category::text, 'other'),
    COALESCE(m.is_breaking, FALSE)
FROM news_articles a
JOIN news_entities e ON e.article_id = a.id
LEFT JOIN news_ai_metadata m ON m.article_id = a.id
LEFT JOIN news_price_impact i ON i.article_id = a.id AND i.coin_id = COALESCE(NULLIF(e.coin_id, ''), '')
WHERE a.is_active = TRUE
  AND a.is_processed = TRUE
  AND COALESCE(NULLIF(e.coin_id, ''), '') <> ''
  AND (i.article_id IS NULL OR i.price_at_publish IS NULL)
ORDER BY a.id, e.coin_id, e.is_primary DESC, e.mention_count DESC, e.id ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list impact candidates: %w", err)
	}
	defer rows.Close()

	items := make([]Candidate, 0)
	for rows.Next() {
		var item Candidate
		var importance sql.NullFloat64
		if err := rows.Scan(&item.ArticleID, &item.CoinID, &item.PublishedAt, &item.Sentiment, &importance, &item.Category, &item.IsBreaking); err != nil {
			return nil, fmt.Errorf("scan impact candidate: %w", err)
		}
		item.Importance = nullFloat64Ptr(importance)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate impact candidates: %w", err)
	}

	return items, nil
}

func (s *Store) UpsertCandidate(ctx context.Context, candidate Candidate, priceAtPublish *float64) error {
	var price any
	if priceAtPublish != nil {
		price = *priceAtPublish
	}

	_, err := s.pool.Exec(ctx, `
INSERT INTO news_price_impact (
    article_id, coin_id, price_at_publish, published_at, updated_at
) VALUES ($1, $2, $3, $4, NOW())
ON CONFLICT (article_id, coin_id)
DO UPDATE SET
    price_at_publish = COALESCE(news_price_impact.price_at_publish, EXCLUDED.price_at_publish),
    published_at = LEAST(news_price_impact.published_at, EXCLUDED.published_at),
    updated_at = NOW()`,
		candidate.ArticleID,
		strings.TrimSpace(candidate.CoinID),
		price,
		candidate.PublishedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("upsert impact candidate article=%d coin=%s: %w", candidate.ArticleID, candidate.CoinID, err)
	}

	return nil
}

func (s *Store) ListPendingRows(ctx context.Context, limit int) ([]ImpactRow, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.pool.Query(ctx, `
SELECT
    i.article_id,
    i.coin_id,
    i.published_at,
    i.price_at_publish,
    i.price_1h,
    i.price_4h,
    i.price_24h,
    i.snapshot_1h_done,
    i.snapshot_4h_done,
    i.snapshot_24h_done,
    COALESCE(m.sentiment::text, 'neutral'),
    m.importance_score,
    COALESCE(m.category::text, 'other'),
    COALESCE(m.is_breaking, FALSE)
FROM news_price_impact i
LEFT JOIN news_ai_metadata m ON m.article_id = i.article_id
WHERE NOT (i.snapshot_1h_done AND i.snapshot_4h_done AND i.snapshot_24h_done)
ORDER BY i.published_at ASC, i.article_id ASC, i.coin_id ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list pending impact rows: %w", err)
	}
	defer rows.Close()

	items := make([]ImpactRow, 0)
	for rows.Next() {
		var item ImpactRow
		var priceAtPublish, price1h, price4h, price24h sql.NullFloat64
		var importance sql.NullFloat64
		if err := rows.Scan(
			&item.ArticleID,
			&item.CoinID,
			&item.PublishedAt,
			&priceAtPublish,
			&price1h,
			&price4h,
			&price24h,
			&item.Snapshot1hDone,
			&item.Snapshot4hDone,
			&item.Snapshot24hDone,
			&item.Sentiment,
			&importance,
			&item.Category,
			&item.IsBreaking,
		); err != nil {
			return nil, fmt.Errorf("scan pending impact row: %w", err)
		}
		item.PriceAtPublish = nullFloat64Ptr(priceAtPublish)
		item.Price1h = nullFloat64Ptr(price1h)
		item.Price4h = nullFloat64Ptr(price4h)
		item.Price24h = nullFloat64Ptr(price24h)
		item.Importance = nullFloat64Ptr(importance)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending impact rows: %w", err)
	}

	return items, nil
}

func (s *Store) UpdateSnapshot(ctx context.Context, articleID int64, coinID, window string, price float64) error {
	var query string
	switch strings.ToLower(strings.TrimSpace(window)) {
	case "1h":
		query = `
UPDATE news_price_impact
SET price_1h = $3,
    snapshot_1h_done = TRUE,
    updated_at = NOW()
WHERE article_id = $1 AND coin_id = $2`
	case "4h":
		query = `
UPDATE news_price_impact
SET price_4h = $3,
    snapshot_4h_done = TRUE,
    updated_at = NOW()
WHERE article_id = $1 AND coin_id = $2`
	case "24h":
		query = `
UPDATE news_price_impact
SET price_24h = $3,
    snapshot_24h_done = TRUE,
    updated_at = NOW()
WHERE article_id = $1 AND coin_id = $2`
	default:
		return fmt.Errorf("unsupported impact window %q", window)
	}

	_, err := s.pool.Exec(ctx, query, articleID, strings.TrimSpace(coinID), price)
	if err != nil {
		return fmt.Errorf("update impact snapshot %s article=%d coin=%s: %w", window, articleID, coinID, err)
	}
	return nil
}

func (s *Store) InsertHistory(ctx context.Context, row ImpactRow) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO news_price_impact_history (
    time, article_id, coin_id, sentiment, importance_score, category, is_breaking,
    change_pct_1h, change_pct_4h, change_pct_24h
) VALUES (
    $1, $2, $3, $4::news_sentiment, $5, $6::news_category, $7, $8, $9, $10
) ON CONFLICT DO NOTHING`,
		row.PublishedAt.UTC(),
		row.ArticleID,
		strings.TrimSpace(row.CoinID),
		normalizeSentiment(row.Sentiment),
		row.Importance,
		normalizeCategory(row.Category),
		row.IsBreaking,
		nilIfZeroPercent(row.PriceAtPublish, row.Price1h),
		nilIfZeroPercent(row.PriceAtPublish, row.Price4h),
		nilIfZeroPercent(row.PriceAtPublish, row.Price24h),
	)
	if err != nil {
		return fmt.Errorf("insert impact history article=%d coin=%s: %w", row.ArticleID, row.CoinID, err)
	}
	return nil
}

func nilIfZeroPercent(base, price *float64) any {
	if base == nil || price == nil || *base <= 0 {
		return nil
	}
	return ((*price - *base) / *base) * 100
}

func normalizeSentiment(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "neutral"
	}
	return value
}

func normalizeCategory(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "other"
	}
	return value
}

func nullFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

var _ interface {
	ListCandidates(context.Context, int) ([]Candidate, error)
	UpsertCandidate(context.Context, Candidate, *float64) error
	ListPendingRows(context.Context, int) ([]ImpactRow, error)
	UpdateSnapshot(context.Context, int64, string, string, float64) error
	InsertHistory(context.Context, ImpactRow) error
} = (*Store)(nil)
