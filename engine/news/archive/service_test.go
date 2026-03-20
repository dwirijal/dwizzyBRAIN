package archive

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

type fakeArchiveStore struct {
	pending  []Article
	metadata map[int64]*Metadata
	entities map[int64][]Entity
	exports  []ExportRecord
}

func (s *fakeArchiveStore) ListPendingArticles(ctx context.Context, limit int) ([]Article, error) {
	items := append([]Article(nil), s.pending...)
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func (s *fakeArchiveStore) LoadMetadata(ctx context.Context, articleID int64) (*Metadata, error) {
	if s.metadata == nil {
		return nil, nil
	}
	meta := s.metadata[articleID]
	if meta == nil {
		return nil, nil
	}
	copy := *meta
	return &copy, nil
}

func (s *fakeArchiveStore) LoadEntities(ctx context.Context, articleID int64) ([]Entity, error) {
	if s.entities == nil {
		return nil, nil
	}
	items := append([]Entity(nil), s.entities[articleID]...)
	return items, nil
}

func (s *fakeArchiveStore) UpsertExport(ctx context.Context, rec ExportRecord) error {
	s.exports = append(s.exports, rec)
	return nil
}

type fakeArchiveUploader struct {
	uploads   []uploadedFile
	shareArgs []string
	link      string
}

type uploadedFile struct {
	source string
	remote string
	body   string
}

func (u *fakeArchiveUploader) UploadFile(ctx context.Context, sourcePath, remoteFilePath string) error {
	body, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	u.uploads = append(u.uploads, uploadedFile{
		source: sourcePath,
		remote: remoteFilePath,
		body:   string(body),
	})
	return nil
}

func (u *fakeArchiveUploader) ShareLink(ctx context.Context, remoteFilePath string) (string, error) {
	u.shareArgs = append(u.shareArgs, remoteFilePath)
	if u.link == "" {
		u.link = "https://drive.google.com/open?id=test"
	}
	return u.link, nil
}

func TestRenderMarkdown(t *testing.T) {
	summaryScore := 0.87
	importance := 0.95
	article := Article{
		ID:          42,
		ExternalID:  "coindesk-42",
		Source:      "coindesk",
		SourceURL:   "https://coindesk.com/feed",
		Title:       "Bitcoin rallies on ETF approval",
		BodyPreview: "Bitcoin moved higher after ETF approval.",
		FullURL:     "https://coindesk.com/article/42",
		Author:      "Jane Doe",
		PublishedAt: time.Date(2026, 3, 19, 11, 22, 33, 0, time.UTC),
		FetchedAt:   time.Date(2026, 3, 19, 11, 25, 00, 0, time.UTC),
		Metadata: &Metadata{
			SummaryShort:    "Bitcoin rallied after approval.",
			SummaryLong:     "The market reacted positively to the approval.",
			KeyPoints:       []string{"ETF approval", "market reaction"},
			Sentiment:       "bullish",
			SentimentScore:  &summaryScore,
			Category:        "markets",
			Subcategory:     "bitcoin",
			ImportanceScore: &importance,
			IsBreaking:      true,
			BreakingType:    "regulation",
		},
		Entities: []Entity{
			{CoinID: "bitcoin", EntityName: "Bitcoin", IsPrimary: true},
			{LlamaSlug: "etf", EntityName: "ETF"},
		},
	}

	md := RenderMarkdown(article)
	checks := []string{
		"---",
		"id: 42",
		"title: \"Bitcoin rallies on ETF approval\"",
		"source: \"coindesk\"",
		"category: \"markets\"",
		"sentiment_score: `0.8700`",
		"# Bitcoin rallies on ETF approval",
		"## Summary",
		"## Key Points",
		"## Entities",
		"- Bitcoin (primary)",
		"## Metadata",
	}
	for _, want := range checks {
		if !strings.Contains(md, want) {
			t.Fatalf("markdown missing %q\n%s", want, md)
		}
	}
}

func TestServiceRunOnceExportsArticle(t *testing.T) {
	fixedNow := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	fixedPublished := time.Date(2026, 3, 18, 9, 30, 0, 0, time.UTC)
	store := &fakeArchiveStore{
		pending: []Article{
			{
				ID:          101,
				ExternalID:  "rss-101",
				Source:      "coindesk",
				SourceURL:   "https://coindesk.com/feed",
				Title:       "Bitcoin rallies on ETF approval",
				BodyPreview: "Bitcoin moved higher after ETF approval.",
				FullURL:     "https://coindesk.com/article/101",
				Author:      "Jane Doe",
				PublishedAt: fixedPublished,
				FetchedAt:   fixedPublished.Add(10 * time.Minute),
			},
		},
		metadata: map[int64]*Metadata{
			101: {
				SummaryShort:    "Bitcoin rallied after approval.",
				SummaryLong:     "The market reacted positively to the approval.",
				KeyPoints:       []string{"ETF approval", "market reaction"},
				Sentiment:       "bullish",
				SentimentScore:  ptrFloat64(0.87),
				Category:        "markets",
				Subcategory:     "bitcoin",
				ImportanceScore: ptrFloat64(0.95),
				IsBreaking:      true,
				BreakingType:    "regulation",
			},
		},
		entities: map[int64][]Entity{
			101: {
				{CoinID: "bitcoin", EntityName: "Bitcoin", IsPrimary: true},
			},
		},
	}
	uploader := &fakeArchiveUploader{
		link: "https://drive.google.com/open?id=drive-md-link",
	}
	svc := NewService(store, uploader, "Projects/DwizzyOS/dwizzyBrain/news", 10)
	svc.now = func() time.Time { return fixedNow }
	svc.tempDirFunc = t.TempDir

	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() error = %v", err)
	}
	if result.ArticlesScanned != 1 || result.ArticlesExported != 1 || result.Failures != 0 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(uploader.uploads) != 1 {
		t.Fatalf("expected 1 upload, got %d", len(uploader.uploads))
	}
	if got := uploader.uploads[0].remote; got != "Projects/DwizzyOS/dwizzyBrain/news/articles/coindesk/2026/03/101-bitcoin-rallies-on-etf-approval/content.md" {
		t.Fatalf("unexpected remote path: %s", got)
	}
	if !strings.Contains(uploader.uploads[0].body, "# Bitcoin rallies on ETF approval") {
		t.Fatalf("expected rendered markdown in upload body, got:\n%s", uploader.uploads[0].body)
	}
	if len(store.exports) != 1 {
		t.Fatalf("expected 1 export row, got %d", len(store.exports))
	}
	if store.exports[0].Title != "Bitcoin rallies on ETF approval" {
		t.Fatalf("unexpected stored title: %q", store.exports[0].Title)
	}
	if store.exports[0].DriveURL != "https://drive.google.com/open?id=drive-md-link" {
		t.Fatalf("unexpected stored drive url: %q", store.exports[0].DriveURL)
	}
	if !strings.HasSuffix(store.exports[0].DrivePath, "content.md") {
		t.Fatalf("unexpected drive path: %q", store.exports[0].DrivePath)
	}
	if len(uploader.shareArgs) != 1 {
		t.Fatalf("expected 1 share call, got %d", len(uploader.shareArgs))
	}
	if uploader.shareArgs[0] != store.exports[0].DrivePath {
		t.Fatalf("markdown share called with %q, want %q", uploader.shareArgs[0], store.exports[0].DrivePath)
	}
	if store.exports[0].ContentFolderPath != "Projects/DwizzyOS/dwizzyBrain/news/articles/coindesk/2026/03/101-bitcoin-rallies-on-etf-approval" {
		t.Fatalf("unexpected folder path: %q", store.exports[0].ContentFolderPath)
	}
	if store.exports[0].ContentJSONPath != "" || store.exports[0].ContentJSONURL != "" {
		t.Fatalf("expected empty json fields, got path=%q url=%q", store.exports[0].ContentJSONPath, store.exports[0].ContentJSONURL)
	}
}

func ptrFloat64(v float64) *float64 {
	return &v
}
