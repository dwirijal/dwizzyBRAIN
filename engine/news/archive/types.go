package archive

import "time"

type Article struct {
	ID                int64     `json:"id"`
	ExternalID        string    `json:"external_id"`
	Source            string    `json:"source"`
	SourceURL         string    `json:"source_url"`
	Title             string    `json:"title"`
	BodyPreview       string    `json:"body_preview"`
	FullURL           string    `json:"full_url"`
	ImageURL          string    `json:"image_url"`
	Author            string    `json:"author"`
	PublishedAt       time.Time `json:"published_at"`
	FetchedAt         time.Time `json:"fetched_at"`
	SourceCredibility float64   `json:"source_credibility"`
	Metadata          *Metadata `json:"metadata,omitempty"`
	Entities          []Entity  `json:"entities,omitempty"`
}

type Metadata struct {
	SummaryShort    string   `json:"summary_short"`
	SummaryLong     string   `json:"summary_long"`
	KeyPoints       []string `json:"key_points"`
	Sentiment       string   `json:"sentiment"`
	SentimentScore  *float64 `json:"sentiment_score,omitempty"`
	Category        string   `json:"category"`
	Subcategory     string   `json:"subcategory"`
	ImportanceScore *float64 `json:"importance_score,omitempty"`
	IsBreaking      bool     `json:"is_breaking"`
	BreakingType    string   `json:"breaking_type"`
}

type Entity struct {
	CoinID     string `json:"coin_id"`
	LlamaSlug  string `json:"llama_slug"`
	EntityType string `json:"entity_type"`
	EntityName string `json:"entity_name"`
	IsPrimary  bool   `json:"is_primary"`
}

type ExportRecord struct {
	ArticleID         int64     `json:"article_id"`
	Title             string    `json:"title"`
	DriveURL          string    `json:"drive_url"`
	DrivePath         string    `json:"drive_path"`
	FileName          string    `json:"file_name"`
	ContentFolderPath string    `json:"content_folder_path"`
	ContentJSONPath   string    `json:"content_json_path"`
	ContentJSONURL    string    `json:"content_json_url"`
	ExportedAt        time.Time `json:"exported_at"`
}

type Result struct {
	ArticlesScanned  int
	ArticlesExported int
	Failures         int
	FailedArticles   []int64
}
