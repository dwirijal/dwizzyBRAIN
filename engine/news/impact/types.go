package impact

import "time"

type Candidate struct {
	ArticleID   int64
	CoinID      string
	PublishedAt time.Time
	Sentiment   string
	Importance  *float64
	Category    string
	IsBreaking  bool
}

type ImpactRow struct {
	ArticleID       int64
	CoinID          string
	PublishedAt     time.Time
	PriceAtPublish  *float64
	Price1h         *float64
	Price4h         *float64
	Price24h        *float64
	Snapshot1hDone  bool
	Snapshot4hDone  bool
	Snapshot24hDone bool
	Sentiment       string
	Importance      *float64
	Category        string
	IsBreaking      bool
}

type Result struct {
	CandidatesUpserted int     `json:"candidates_upserted"`
	SnapshotsUpdated   int     `json:"snapshots_updated"`
	HistoryInserted    int     `json:"history_inserted"`
	Failures           int     `json:"failures"`
	FailedArticles     []int64 `json:"failed_articles,omitempty"`
}
