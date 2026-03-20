package ai

import "time"

type Article struct {
	ID                int64
	ExternalID        string
	Source            string
	SourceCredibility float64
	SourceURL         string
	Title             string
	BodyPreview       string
	FullURL           string
	ImageURL          string
	Author            string
	PublishedAt       time.Time
	FetchedAt         time.Time
	CPKind            string
	CPVotesPositive   int
	CPVotesNegative   int
	CPVotesImportant  int
}

type CoinEntity struct {
	CoinID string
	Symbol string
	Name   string
}

type ProtocolEntity struct {
	Slug string
	Name string
}

type Metadata struct {
	ArticleID           int64
	SummaryShort        string
	SummaryLong         string
	KeyPoints           []string
	Sentiment           string
	SentimentScore      float64
	Category            string
	Subcategory         string
	ImportanceScore     float64
	IsBreaking          bool
	BreakingType        string
	ModelUsed           string
	ProcessingLatencyMS int
}

type Entity struct {
	ArticleID    int64
	CoinID       string
	LlamaSlug    string
	EntityType   string
	EntityName   string
	IsPrimary    bool
	MentionCount int
	Confidence   float64
}

type Result struct {
	ArticlesProcessed int     `json:"articles_processed"`
	MetadataUpserted  int     `json:"metadata_upserted"`
	EntitiesUpserted  int     `json:"entities_upserted"`
	Failures          int     `json:"failures"`
	FailedArticles    []int64 `json:"failed_articles,omitempty"`
}
