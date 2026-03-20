package newsapi

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultListLimit     = 20
	maxListLimit         = 100
	defaultTrendingLimit = 10
)

type Service struct {
	db *pgxpool.Pool
}

type ArticleMetadata struct {
	SummaryShort        string     `json:"summary_short,omitempty"`
	SummaryLong         string     `json:"summary_long,omitempty"`
	KeyPoints           []string   `json:"key_points,omitempty"`
	Sentiment           string     `json:"sentiment,omitempty"`
	SentimentScore      *float64   `json:"sentiment_score,omitempty"`
	Category            string     `json:"category,omitempty"`
	Subcategory         string     `json:"subcategory,omitempty"`
	ImportanceScore     *float64   `json:"importance_score,omitempty"`
	IsBreaking          bool       `json:"is_breaking,omitempty"`
	BreakingType        string     `json:"breaking_type,omitempty"`
	ModelUsed           string     `json:"model_used,omitempty"`
	ProcessingLatencyMS *int       `json:"processing_latency_ms,omitempty"`
	ProcessedAt         *time.Time `json:"processed_at,omitempty"`
}

type Entity struct {
	CoinID       string   `json:"coin_id,omitempty"`
	LlamaSlug    string   `json:"llama_slug,omitempty"`
	EntityType   string   `json:"entity_type"`
	EntityName   string   `json:"entity_name,omitempty"`
	IsPrimary    bool     `json:"is_primary"`
	MentionCount int      `json:"mention_count"`
	Confidence   *float64 `json:"confidence,omitempty"`
}

type ArticleSummary struct {
	ID                int64            `json:"id"`
	ExternalID        string           `json:"external_id"`
	Source            string           `json:"source"`
	SourceName        string           `json:"source_name,omitempty"`
	SourceCredibility float64          `json:"source_credibility,omitempty"`
	Title             string           `json:"title"`
	BodyPreview       string           `json:"body_preview,omitempty"`
	FullURL           string           `json:"full_url,omitempty"`
	ImageURL          string           `json:"image_url,omitempty"`
	Author            string           `json:"author,omitempty"`
	PublishedAt       time.Time        `json:"published_at"`
	FetchedAt         time.Time        `json:"fetched_at"`
	IsProcessed       bool             `json:"is_processed"`
	Metadata          *ArticleMetadata `json:"metadata,omitempty"`
}

type ArticleDetail struct {
	ArticleSummary
	Entities     []Entity      `json:"entities,omitempty"`
	PriceImpacts []PriceImpact `json:"price_impact,omitempty"`
}

type PriceImpact struct {
	CoinID          string    `json:"coin_id"`
	PublishedAt     time.Time `json:"published_at"`
	PriceAtPublish  *float64  `json:"price_at_publish,omitempty"`
	Price1h         *float64  `json:"price_1h,omitempty"`
	Price4h         *float64  `json:"price_4h,omitempty"`
	Price24h        *float64  `json:"price_24h,omitempty"`
	ChangePct1h     *float64  `json:"change_pct_1h,omitempty"`
	ChangePct4h     *float64  `json:"change_pct_4h,omitempty"`
	ChangePct24h    *float64  `json:"change_pct_24h,omitempty"`
	Snapshot1hDone  bool      `json:"snapshot_1h_done"`
	Snapshot4hDone  bool      `json:"snapshot_4h_done"`
	Snapshot24hDone bool      `json:"snapshot_24h_done"`
}

type ArticlePage struct {
	Items []ArticleSummary `json:"items"`
	Total int              `json:"total"`
}

type TrendingCoin struct {
	CoinID        string   `json:"coin_id"`
	Symbol        string   `json:"symbol,omitempty"`
	Name          string   `json:"name,omitempty"`
	MentionCount  int      `json:"mention_count"`
	AvgSentiment  *float64 `json:"avg_sentiment_score,omitempty"`
	AvgImportance *float64 `json:"avg_importance_score,omitempty"`
}

type TrendingProtocol struct {
	Slug          string   `json:"slug"`
	Name          string   `json:"name,omitempty"`
	MentionCount  int      `json:"mention_count"`
	AvgImportance *float64 `json:"avg_importance_score,omitempty"`
}

type TrendingCategory struct {
	Category      string   `json:"category"`
	MentionCount  int      `json:"mention_count"`
	AvgImportance *float64 `json:"avg_importance_score,omitempty"`
}

type TrendingArticle struct {
	ArticleSummary
	ImportanceScore *float64 `json:"importance_score,omitempty"`
}

type TrendingResponse struct {
	Window      string             `json:"window"`
	GeneratedAt time.Time          `json:"generated_at"`
	Coins       []TrendingCoin     `json:"coins"`
	Protocols   []TrendingProtocol `json:"protocols"`
	Categories  []TrendingCategory `json:"categories"`
	TopArticles []TrendingArticle  `json:"top_articles"`
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) IsCategory(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "regulation", "hack_exploit", "partnership", "listing_delisting", "market_analysis", "macro", "technology", "adoption", "fundraising", "whale_movement", "defi", "nft", "layer2", "other":
		return true
	default:
		return false
	}
}

func (s *Service) List(ctx context.Context, limit, offset int, category string) (ArticlePage, error) {
	if s.db == nil {
		return ArticlePage{}, fmt.Errorf("postgres pool is required")
	}
	limit = clampListLimit(limit)
	if offset < 0 {
		offset = 0
	}
	category = strings.TrimSpace(category)

	countQuery := `
SELECT count(*)
FROM news_articles a
LEFT JOIN news_sources s ON s.source_name = a.source
LEFT JOIN news_ai_metadata m ON m.article_id = a.id
WHERE a.is_active = TRUE`
	dataQuery := articleQueryBase() + ` WHERE a.is_active = TRUE`
	countArgs := []any{}
	var total int
	if category != "" {
		countQuery += ` AND LOWER(COALESCE(NULLIF(m.category::text, ''), 'other')) = LOWER($1)`
		dataQuery += ` AND LOWER(COALESCE(NULLIF(m.category::text, ''), 'other')) = LOWER($1)`
		countArgs = append(countArgs, category)
	}
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return ArticlePage{}, fmt.Errorf("count news articles: %w", err)
	}
	dataQuery += ` ORDER BY a.published_at DESC, a.id DESC LIMIT $` + strconv.Itoa(len(countArgs)+1) + ` OFFSET $` + strconv.Itoa(len(countArgs)+2)
	queryArgs := append(countArgs, limit, offset)
	rows, err := s.db.Query(ctx, dataQuery, queryArgs...)
	if err != nil {
		return ArticlePage{}, fmt.Errorf("query news articles: %w", err)
	}
	defer rows.Close()

	items := make([]ArticleSummary, 0)
	for rows.Next() {
		item, _, err := scanArticleRow(rows)
		if err != nil {
			return ArticlePage{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ArticlePage{}, fmt.Errorf("iterate news articles: %w", err)
	}

	return ArticlePage{Items: items, Total: total}, nil
}

func (s *Service) Detail(ctx context.Context, id int64) (ArticleDetail, error) {
	if s.db == nil {
		return ArticleDetail{}, fmt.Errorf("postgres pool is required")
	}
	query := articleQueryBase() + ` WHERE a.id = $1 ORDER BY a.id DESC LIMIT 1`
	row, _, err := scanArticleRow(s.db.QueryRow(ctx, query, id))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ArticleDetail{}, fmt.Errorf("article %d not found", id)
		}
		return ArticleDetail{}, err
	}
	entities, err := s.entities(ctx, id)
	if err != nil {
		return ArticleDetail{}, err
	}
	impacts, err := s.priceImpacts(ctx, id)
	if err != nil {
		return ArticleDetail{}, err
	}
	return ArticleDetail{ArticleSummary: row, Entities: entities, PriceImpacts: impacts}, nil
}

func (s *Service) ByCoin(ctx context.Context, coinID string, limit, offset int) (ArticlePage, error) {
	if s.db == nil {
		return ArticlePage{}, fmt.Errorf("postgres pool is required")
	}
	coinID = strings.TrimSpace(coinID)
	if coinID == "" {
		return ArticlePage{}, fmt.Errorf("coin id is required")
	}
	limit = clampListLimit(limit)
	if offset < 0 {
		offset = 0
	}

	countQuery := `
SELECT count(DISTINCT a.id)
FROM news_articles a
JOIN news_entities e ON e.article_id = a.id
WHERE a.is_active = TRUE AND LOWER(COALESCE(NULLIF(e.coin_id, ''), '')) = LOWER($1)`
	var total int
	if err := s.db.QueryRow(ctx, countQuery, coinID).Scan(&total); err != nil {
		return ArticlePage{}, fmt.Errorf("count news articles by coin: %w", err)
	}

	idRows, err := s.db.Query(ctx, `
SELECT a.id
FROM news_articles a
JOIN news_entities e ON e.article_id = a.id
WHERE a.is_active = TRUE AND LOWER(COALESCE(NULLIF(e.coin_id, ''), '')) = LOWER($1)
GROUP BY a.id, a.published_at
ORDER BY a.published_at DESC, a.id DESC
LIMIT $2 OFFSET $3`, coinID, limit, offset)
	if err != nil {
		return ArticlePage{}, fmt.Errorf("query news article ids by coin: %w", err)
	}
	defer idRows.Close()

	ids := make([]int64, 0)
	for idRows.Next() {
		var id int64
		if err := idRows.Scan(&id); err != nil {
			return ArticlePage{}, fmt.Errorf("scan news coin article id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := idRows.Err(); err != nil {
		return ArticlePage{}, fmt.Errorf("iterate news article ids by coin: %w", err)
	}

	items := make([]ArticleSummary, 0, len(ids))
	for _, id := range ids {
		item, err := s.Detail(ctx, id)
		if err != nil {
			return ArticlePage{}, err
		}
		items = append(items, item.ArticleSummary)
	}

	return ArticlePage{Items: items, Total: total}, nil
}

func (s *Service) Trending(ctx context.Context, window time.Duration, limit int) (TrendingResponse, error) {
	if s.db == nil {
		return TrendingResponse{}, fmt.Errorf("postgres pool is required")
	}
	if window <= 0 {
		window = 24 * time.Hour
	}
	limit = clampTrendingLimit(limit)
	since := time.Now().UTC().Add(-window)

	coins, err := s.trendingCoins(ctx, since, limit)
	if err != nil {
		return TrendingResponse{}, err
	}
	protocols, err := s.trendingProtocols(ctx, since, limit)
	if err != nil {
		return TrendingResponse{}, err
	}
	categories, err := s.trendingCategories(ctx, since, limit)
	if err != nil {
		return TrendingResponse{}, err
	}
	articles, err := s.trendingArticles(ctx, since, limit)
	if err != nil {
		return TrendingResponse{}, err
	}

	return TrendingResponse{
		Window:      window.String(),
		GeneratedAt: time.Now().UTC(),
		Coins:       coins,
		Protocols:   protocols,
		Categories:  categories,
		TopArticles: articles,
	}, nil
}

func articleQueryBase() string {
	return `
SELECT
    a.id, a.external_id, a.source, COALESCE(s.display_name, '') AS source_name,
    COALESCE(s.credibility_score, 0.5),
    a.title, COALESCE(a.body_preview, ''), COALESCE(a.full_url, ''),
    COALESCE(a.image_url, ''), COALESCE(a.author, ''),
    a.published_at, a.fetched_at, a.is_processed,
    m.article_id,
    m.summary_short, m.summary_long, COALESCE(m.key_points, '{}'::text[]),
    m.sentiment, m.sentiment_score,
    COALESCE(m.category::text, 'other'), COALESCE(m.subcategory, ''),
    m.importance_score, COALESCE(m.is_breaking, FALSE), COALESCE(m.breaking_type, ''),
    COALESCE(m.model_used, ''), m.processing_latency_ms, m.processed_at
FROM news_articles a
LEFT JOIN news_sources s ON s.source_name = a.source
LEFT JOIN news_ai_metadata m ON m.article_id = a.id`
}

func scanArticleRow(row interface {
	Scan(dest ...any) error
}) (ArticleSummary, *ArticleMetadata, error) {
	var (
		item                ArticleSummary
		credibility         sql.NullFloat64
		metaExists          sql.NullInt64
		summaryShort        sql.NullString
		summaryLong         sql.NullString
		keyPoints           []string
		sentiment           sql.NullString
		sentimentScore      sql.NullFloat64
		category            sql.NullString
		subcategory         sql.NullString
		importanceScore     sql.NullFloat64
		isBreaking          sql.NullBool
		breakingType        sql.NullString
		modelUsed           sql.NullString
		processingLatencyMS sql.NullInt64
		processedAt         sql.NullTime
	)
	if err := row.Scan(
		&item.ID,
		&item.ExternalID,
		&item.Source,
		&item.SourceName,
		&credibility,
		&item.Title,
		&item.BodyPreview,
		&item.FullURL,
		&item.ImageURL,
		&item.Author,
		&item.PublishedAt,
		&item.FetchedAt,
		&item.IsProcessed,
		&metaExists,
		&summaryShort,
		&summaryLong,
		&keyPoints,
		&sentiment,
		&sentimentScore,
		&category,
		&subcategory,
		&importanceScore,
		&isBreaking,
		&breakingType,
		&modelUsed,
		&processingLatencyMS,
		&processedAt,
	); err != nil {
		return ArticleSummary{}, nil, err
	}

	item.SourceCredibility = nullFloat64Value(credibility, 0.5)
	item.PublishedAt = item.PublishedAt.UTC()
	item.FetchedAt = item.FetchedAt.UTC()

	var meta *ArticleMetadata
	if metaExists.Valid {
		meta = &ArticleMetadata{
			SummaryShort:        summaryShort.String,
			SummaryLong:         summaryLong.String,
			KeyPoints:           keyPoints,
			Sentiment:           sentiment.String,
			SentimentScore:      nullFloat64Ptr(sentimentScore),
			Category:            category.String,
			Subcategory:         subcategory.String,
			ImportanceScore:     nullFloat64Ptr(importanceScore),
			IsBreaking:          isBreaking.Bool,
			BreakingType:        breakingType.String,
			ModelUsed:           modelUsed.String,
			ProcessingLatencyMS: nullIntPtr(processingLatencyMS),
			ProcessedAt:         nullTimePtr(processedAt),
		}
	}

	return item, meta, nil
}

func (s *Service) entities(ctx context.Context, articleID int64) ([]Entity, error) {
	rows, err := s.db.Query(ctx, `
SELECT coin_id, llama_slug, entity_type, COALESCE(entity_name, ''), is_primary, mention_count, confidence
FROM news_entities
WHERE article_id = $1
ORDER BY is_primary DESC, mention_count DESC, entity_type ASC, id ASC`, articleID)
	if err != nil {
		return nil, fmt.Errorf("query news entities: %w", err)
	}
	defer rows.Close()

	items := make([]Entity, 0)
	for rows.Next() {
		var (
			item       Entity
			coinID     sql.NullString
			llamaSlug  sql.NullString
			entityType sql.NullString
			entityName sql.NullString
			confidence sql.NullFloat64
		)
		if err := rows.Scan(
			&coinID,
			&llamaSlug,
			&entityType,
			&entityName,
			&item.IsPrimary,
			&item.MentionCount,
			&confidence,
		); err != nil {
			return nil, fmt.Errorf("scan news entity: %w", err)
		}
		item.CoinID = coinID.String
		item.LlamaSlug = llamaSlug.String
		item.EntityType = entityType.String
		item.EntityName = entityName.String
		item.Confidence = nullFloat64Ptr(confidence)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate news entities: %w", err)
	}
	return items, nil
}

func (s *Service) priceImpacts(ctx context.Context, articleID int64) ([]PriceImpact, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    coin_id, published_at, price_at_publish, price_1h, price_4h, price_24h,
    change_pct_1h, change_pct_4h, change_pct_24h,
    snapshot_1h_done, snapshot_4h_done, snapshot_24h_done
FROM news_price_impact
WHERE article_id = $1
ORDER BY coin_id ASC`, articleID)
	if err != nil {
		return nil, fmt.Errorf("query news price impact: %w", err)
	}
	defer rows.Close()

	items := make([]PriceImpact, 0)
	for rows.Next() {
		var (
			item           PriceImpact
			priceAtPublish sql.NullFloat64
			price1h        sql.NullFloat64
			price4h        sql.NullFloat64
			price24h       sql.NullFloat64
			changePct1h    sql.NullFloat64
			changePct4h    sql.NullFloat64
			changePct24h   sql.NullFloat64
		)
		if err := rows.Scan(
			&item.CoinID,
			&item.PublishedAt,
			&priceAtPublish,
			&price1h,
			&price4h,
			&price24h,
			&changePct1h,
			&changePct4h,
			&changePct24h,
			&item.Snapshot1hDone,
			&item.Snapshot4hDone,
			&item.Snapshot24hDone,
		); err != nil {
			return nil, fmt.Errorf("scan news price impact: %w", err)
		}
		item.PriceAtPublish = nullFloat64Ptr(priceAtPublish)
		item.Price1h = nullFloat64Ptr(price1h)
		item.Price4h = nullFloat64Ptr(price4h)
		item.Price24h = nullFloat64Ptr(price24h)
		item.ChangePct1h = nullFloat64Ptr(changePct1h)
		item.ChangePct4h = nullFloat64Ptr(changePct4h)
		item.ChangePct24h = nullFloat64Ptr(changePct24h)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate news price impact: %w", err)
	}

	return items, nil
}

func (s *Service) trendingCoins(ctx context.Context, since time.Time, limit int) ([]TrendingCoin, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    e.coin_id,
    COALESCE(NULLIF(c.symbol, ''), '') AS symbol,
    COALESCE(NULLIF(c.name, ''), '') AS name,
    COUNT(*) AS mentions,
    AVG(COALESCE(m.sentiment_score, 0)) AS avg_sentiment,
    AVG(COALESCE(m.importance_score, 0)) AS avg_importance
FROM news_entities e
JOIN news_articles a ON a.id = e.article_id
LEFT JOIN news_ai_metadata m ON m.article_id = a.id
LEFT JOIN coins c ON LOWER(COALESCE(NULLIF(c.coin_id, ''), c.id)) = LOWER(e.coin_id)
WHERE a.published_at >= $1
  AND COALESCE(NULLIF(e.coin_id, ''), '') <> ''
GROUP BY e.coin_id, c.symbol, c.name
ORDER BY mentions DESC, avg_importance DESC
LIMIT $2`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query trending coins: %w", err)
	}
	defer rows.Close()

	items := make([]TrendingCoin, 0)
	for rows.Next() {
		var (
			item          TrendingCoin
			symbol        sql.NullString
			name          sql.NullString
			avgSentiment  sql.NullFloat64
			avgImportance sql.NullFloat64
		)
		if err := rows.Scan(&item.CoinID, &symbol, &name, &item.MentionCount, &avgSentiment, &avgImportance); err != nil {
			return nil, fmt.Errorf("scan trending coin: %w", err)
		}
		item.Symbol = symbol.String
		item.Name = name.String
		item.AvgSentiment = nullFloat64Ptr(avgSentiment)
		item.AvgImportance = nullFloat64Ptr(avgImportance)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trending coins: %w", err)
	}
	return items, nil
}

func (s *Service) trendingProtocols(ctx context.Context, since time.Time, limit int) ([]TrendingProtocol, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    e.llama_slug,
    COALESCE(NULLIF(p.name, ''), '') AS name,
    COUNT(*) AS mentions,
    AVG(COALESCE(m.importance_score, 0)) AS avg_importance
FROM news_entities e
JOIN news_articles a ON a.id = e.article_id
LEFT JOIN news_ai_metadata m ON m.article_id = a.id
LEFT JOIN defi_protocols p ON LOWER(COALESCE(NULLIF(p.slug, ''), '')) = LOWER(e.llama_slug)
WHERE a.published_at >= $1
  AND COALESCE(NULLIF(e.llama_slug, ''), '') <> ''
GROUP BY e.llama_slug, p.name
ORDER BY mentions DESC, avg_importance DESC
LIMIT $2`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query trending protocols: %w", err)
	}
	defer rows.Close()

	items := make([]TrendingProtocol, 0)
	for rows.Next() {
		var (
			item          TrendingProtocol
			name          sql.NullString
			avgImportance sql.NullFloat64
		)
		if err := rows.Scan(&item.Slug, &name, &item.MentionCount, &avgImportance); err != nil {
			return nil, fmt.Errorf("scan trending protocol: %w", err)
		}
		item.Name = name.String
		item.AvgImportance = nullFloat64Ptr(avgImportance)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trending protocols: %w", err)
	}
	return items, nil
}

func (s *Service) trendingCategories(ctx context.Context, since time.Time, limit int) ([]TrendingCategory, error) {
	rows, err := s.db.Query(ctx, `
SELECT
    COALESCE(m.category::text, 'other') AS category,
    COUNT(*) AS mentions,
    AVG(COALESCE(m.importance_score, 0)) AS avg_importance
FROM news_ai_metadata m
JOIN news_articles a ON a.id = m.article_id
WHERE a.published_at >= $1
GROUP BY COALESCE(m.category::text, 'other')
ORDER BY mentions DESC, avg_importance DESC
LIMIT $2`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query trending categories: %w", err)
	}
	defer rows.Close()

	items := make([]TrendingCategory, 0)
	for rows.Next() {
		var (
			item          TrendingCategory
			avgImportance sql.NullFloat64
		)
		if err := rows.Scan(&item.Category, &item.MentionCount, &avgImportance); err != nil {
			return nil, fmt.Errorf("scan trending category: %w", err)
		}
		item.AvgImportance = nullFloat64Ptr(avgImportance)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trending categories: %w", err)
	}
	return items, nil
}

func (s *Service) trendingArticles(ctx context.Context, since time.Time, limit int) ([]TrendingArticle, error) {
	rows, err := s.db.Query(ctx, articleQueryBase()+`
WHERE a.published_at >= $1
ORDER BY COALESCE(m.importance_score, 0) DESC, a.published_at DESC
LIMIT $2`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("query trending articles: %w", err)
	}
	defer rows.Close()

	items := make([]TrendingArticle, 0)
	for rows.Next() {
		item, meta, err := scanArticleRow(rows)
		if err != nil {
			return nil, err
		}
		top := TrendingArticle{ArticleSummary: item}
		if meta != nil {
			top.ImportanceScore = meta.ImportanceScore
		}
		items = append(items, top)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate trending articles: %w", err)
	}
	return items, nil
}

func clampListLimit(limit int) int {
	if limit <= 0 {
		return defaultListLimit
	}
	if limit > maxListLimit {
		return maxListLimit
	}
	return limit
}

func clampTrendingLimit(limit int) int {
	if limit <= 0 {
		return defaultTrendingLimit
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func nullFloat64Value(value sql.NullFloat64, fallback float64) float64 {
	if value.Valid {
		return value.Float64
	}
	return fallback
}

func nullFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}
	v := value.Float64
	return &v
}

func nullIntPtr(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	v := int(value.Int64)
	return &v
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	v := value.Time.UTC()
	return &v
}

var _ interface {
	IsCategory(string) bool
	List(context.Context, int, int, string) (ArticlePage, error)
	Detail(context.Context, int64) (ArticleDetail, error)
	ByCoin(context.Context, string, int, int) (ArticlePage, error)
	Trending(context.Context, time.Duration, int) (TrendingResponse, error)
} = (*Service)(nil)
