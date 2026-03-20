package news

import (
	"database/sql"
	"time"
)

type Source struct {
	SourceName           string
	DisplayName          string
	BaseURL              sql.NullString
	RSSURL               sql.NullString
	LogoURL              sql.NullString
	CredibilityScore     float64
	PollIntervalSeconds  int
	IsActive             bool
	FetchType            string
	ArticlesFetchedTotal int64
	LastFetchedAt        sql.NullTime
	LastSuccessAt        sql.NullTime
	ConsecutiveFailures  int
}

type Article struct {
	ExternalID       string
	Source           string
	SourceURL        string
	Title            string
	BodyPreview      string
	FullURL          string
	ImageURL         string
	Author           string
	PublishedAt      time.Time
	FetchedAt        time.Time
	CPKind           string
	CPVotesPositive  int
	CPVotesNegative  int
	CPVotesImportant int
}

type Result struct {
	SourcesProcessed int      `json:"sources_processed"`
	ArticlesFetched  int      `json:"articles_fetched"`
	ArticlesInserted int      `json:"articles_inserted"`
	Failures         int      `json:"failures"`
	FailedSources    []string `json:"failed_sources,omitempty"`
}

func (s Source) BaseURLValue() string {
	if s.BaseURL.Valid {
		return s.BaseURL.String
	}
	return ""
}

func (s Source) RSSURLValue() string {
	if s.RSSURL.Valid {
		return s.RSSURL.String
	}
	return ""
}

func (s Source) LogoURLValue() string {
	if s.LogoURL.Valid {
		return s.LogoURL.String
	}
	return ""
}
