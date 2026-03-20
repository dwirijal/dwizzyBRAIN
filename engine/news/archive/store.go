package archive

import (
	"context"
	"fmt"

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
    a.id, a.external_id, a.source::text, a.source_url, a.title,
    COALESCE(a.body_preview, ''), COALESCE(a.full_url, ''), COALESCE(a.image_url, ''),
    COALESCE(a.author, ''), a.published_at, a.fetched_at, COALESCE(src.credibility_score, 0.5)
FROM news_articles a
LEFT JOIN news_sources src ON src.source_name = a.source
LEFT JOIN news_article_markdown_exports x ON x.article_id = a.id
WHERE a.is_active = TRUE AND x.article_id IS NULL
ORDER BY a.published_at DESC, a.id DESC
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
			&item.SourceURL,
			&item.Title,
			&item.BodyPreview,
			&item.FullURL,
			&item.ImageURL,
			&item.Author,
			&item.PublishedAt,
			&item.FetchedAt,
			&item.SourceCredibility,
		); err != nil {
			return nil, fmt.Errorf("scan news article: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending news articles: %w", err)
	}

	return items, nil
}

func (s *Store) LoadMetadata(ctx context.Context, articleID int64) (*Metadata, error) {
	row := s.pool.QueryRow(ctx, `
SELECT
    COALESCE(summary_short, ''),
    COALESCE(summary_long, ''),
    COALESCE(key_points, ARRAY[]::TEXT[]),
    COALESCE(sentiment::text, ''),
    sentiment_score,
    COALESCE(category::text, ''),
    COALESCE(subcategory, ''),
    importance_score,
    COALESCE(is_breaking, FALSE),
    COALESCE(breaking_type, '')
FROM news_ai_metadata
WHERE article_id = $1`, articleID)

	var meta Metadata
	if err := row.Scan(
		&meta.SummaryShort,
		&meta.SummaryLong,
		&meta.KeyPoints,
		&meta.Sentiment,
		&meta.SentimentScore,
		&meta.Category,
		&meta.Subcategory,
		&meta.ImportanceScore,
		&meta.IsBreaking,
		&meta.BreakingType,
	); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("load news ai metadata: %w", err)
	}
	return &meta, nil
}

func (s *Store) LoadEntities(ctx context.Context, articleID int64) ([]Entity, error) {
	rows, err := s.pool.Query(ctx, `
SELECT COALESCE(coin_id, ''), COALESCE(llama_slug, ''), entity_type, COALESCE(entity_name, ''), COALESCE(is_primary, FALSE)
FROM news_entities
WHERE article_id = $1
ORDER BY is_primary DESC, mention_count DESC, entity_name ASC`, articleID)
	if err != nil {
		return nil, fmt.Errorf("load news entities: %w", err)
	}
	defer rows.Close()

	items := make([]Entity, 0)
	for rows.Next() {
		var item Entity
		if err := rows.Scan(&item.CoinID, &item.LlamaSlug, &item.EntityType, &item.EntityName, &item.IsPrimary); err != nil {
			return nil, fmt.Errorf("scan news entity: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate news entities: %w", err)
	}
	return items, nil
}

func (s *Store) UpsertExport(ctx context.Context, rec ExportRecord) error {
	_, err := s.pool.Exec(ctx, `
INSERT INTO news_article_markdown_exports (
    article_id, title, drive_url, drive_path, file_name, content_folder_path, content_json_path, content_json_url, exported_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, COALESCE($9, NOW()))
ON CONFLICT (article_id) DO UPDATE SET
    title = EXCLUDED.title,
    drive_url = EXCLUDED.drive_url,
    drive_path = EXCLUDED.drive_path,
    file_name = EXCLUDED.file_name,
    content_folder_path = EXCLUDED.content_folder_path,
    content_json_path = EXCLUDED.content_json_path,
    content_json_url = EXCLUDED.content_json_url,
    exported_at = EXCLUDED.exported_at`,
		rec.ArticleID,
		rec.Title,
		rec.DriveURL,
		rec.DrivePath,
		rec.FileName,
		rec.ContentFolderPath,
		rec.ContentJSONPath,
		rec.ContentJSONURL,
		rec.ExportedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert news article markdown export article=%d: %w", rec.ArticleID, err)
	}
	return nil
}

func (s *Store) LoadExport(ctx context.Context, articleID int64) (ExportRecord, error) {
	row := s.pool.QueryRow(ctx, `
SELECT article_id, title, drive_url, drive_path, file_name, content_folder_path, content_json_path, content_json_url, exported_at
FROM news_article_markdown_exports
WHERE article_id = $1`, articleID)
	var rec ExportRecord
	if err := row.Scan(&rec.ArticleID, &rec.Title, &rec.DriveURL, &rec.DrivePath, &rec.FileName, &rec.ContentFolderPath, &rec.ContentJSONPath, &rec.ContentJSONURL, &rec.ExportedAt); err != nil {
		if err == pgx.ErrNoRows {
			return ExportRecord{}, err
		}
		return ExportRecord{}, fmt.Errorf("load news article markdown export: %w", err)
	}
	return rec, nil
}

func (s *Store) CountExports(ctx context.Context) (int, error) {
	var count int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM news_article_markdown_exports`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count news article markdown exports: %w", err)
	}
	return count, nil
}

func (s *Store) ArticleByID(ctx context.Context, articleID int64) (Article, error) {
	row := s.pool.QueryRow(ctx, `
SELECT
    a.id, a.external_id, a.source::text, a.source_url, a.title,
    COALESCE(a.body_preview, ''), COALESCE(a.full_url, ''), COALESCE(a.image_url, ''),
    COALESCE(a.author, ''), a.published_at, a.fetched_at, COALESCE(src.credibility_score, 0.5)
FROM news_articles a
LEFT JOIN news_sources src ON src.source_name = a.source
WHERE a.id = $1`, articleID)

	var item Article
	if err := row.Scan(
		&item.ID,
		&item.ExternalID,
		&item.Source,
		&item.SourceURL,
		&item.Title,
		&item.BodyPreview,
		&item.FullURL,
		&item.ImageURL,
		&item.Author,
		&item.PublishedAt,
		&item.FetchedAt,
		&item.SourceCredibility,
	); err != nil {
		return Article{}, err
	}

	meta, err := s.LoadMetadata(ctx, articleID)
	if err != nil {
		return Article{}, err
	}
	item.Metadata = meta
	entities, err := s.LoadEntities(ctx, articleID)
	if err != nil {
		return Article{}, err
	}
	item.Entities = entities
	return item, nil
}

func (a Article) PublishedYearMonth() (string, string) {
	return a.PublishedAt.UTC().Format("2006"), a.PublishedAt.UTC().Format("01")
}

var _ interface {
	ListPendingArticles(context.Context, int) ([]Article, error)
	LoadMetadata(context.Context, int64) (*Metadata, error)
	LoadEntities(context.Context, int64) ([]Entity, error)
	UpsertExport(context.Context, ExportRecord) error
	LoadExport(context.Context, int64) (ExportRecord, error)
	CountExports(context.Context) (int, error)
	ArticleByID(context.Context, int64) (Article, error)
} = (*Store)(nil)
