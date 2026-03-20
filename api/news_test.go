package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"dwizzyBRAIN/api/handler"
	newsapi "dwizzyBRAIN/api/news"
)

type fakeNewsReader struct{}

func (f *fakeNewsReader) IsCategory(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "defi", "regulation", "macro":
		return true
	default:
		return false
	}
}

func (f *fakeNewsReader) List(ctx context.Context, limit, offset int, category string) (newsapi.ArticlePage, error) {
	return newsapi.ArticlePage{
		Items: []newsapi.ArticleSummary{
			{
				ID:          1,
				ExternalID:  "ext-1",
				Source:      "coindesk",
				Title:       "Bitcoin breaks out",
				PublishedAt: time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
				FetchedAt:   time.Date(2026, 3, 19, 0, 5, 0, 0, time.UTC),
				IsProcessed: true,
			},
		},
		Total: 1,
	}, nil
}

func (f *fakeNewsReader) Detail(ctx context.Context, id int64) (newsapi.ArticleDetail, error) {
	return newsapi.ArticleDetail{
		ArticleSummary: newsapi.ArticleSummary{
			ID:          id,
			ExternalID:  "ext-detail",
			Source:      "coindesk",
			Title:       "Bitcoin detail",
			PublishedAt: time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
			FetchedAt:   time.Date(2026, 3, 19, 0, 5, 0, 0, time.UTC),
			IsProcessed: true,
		},
		Entities: []newsapi.Entity{
			{CoinID: "bitcoin", EntityType: "coin", IsPrimary: true, MentionCount: 2},
		},
	}, nil
}

func (f *fakeNewsReader) ByCoin(ctx context.Context, coinID string, limit, offset int) (newsapi.ArticlePage, error) {
	return f.List(ctx, limit, offset, coinID)
}

func (f *fakeNewsReader) Trending(ctx context.Context, window time.Duration, limit int) (newsapi.TrendingResponse, error) {
	return newsapi.TrendingResponse{
		Window:      window.String(),
		GeneratedAt: time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
		Coins: []newsapi.TrendingCoin{
			{CoinID: "bitcoin", MentionCount: 4},
		},
		Protocols: []newsapi.TrendingProtocol{
			{Slug: "uniswap", MentionCount: 2},
		},
		Categories: []newsapi.TrendingCategory{
			{Category: "defi", MentionCount: 3},
		},
		TopArticles: []newsapi.TrendingArticle{
			{ArticleSummary: newsapi.ArticleSummary{ID: 1, ExternalID: "ext-1", Source: "coindesk", Title: "Bitcoin breaks out", PublishedAt: time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC), FetchedAt: time.Date(2026, 3, 19, 0, 5, 0, 0, time.UTC), IsProcessed: true}},
		},
	}, nil
}

func TestNewsRoutes(t *testing.T) {
	mux := NewRouter(nil, nil, handler.NewNewsHandler(&fakeNewsReader{}), nil, nil)

	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "list", path: "/v1/news", want: `"total":1`},
		{name: "detail", path: "/v1/news/42", want: `"id":42`},
		{name: "category", path: "/v1/news/defi", want: `"title":"Bitcoin breaks out"`},
		{name: "coin", path: "/v1/news/coin/bitcoin", want: `"coin_id":"bitcoin"`},
		{name: "trending", path: "/v1/news/trending", want: `"window":"24h0m0s"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("expected json content type, got %q", ct)
			}
			if !json.Valid(rec.Body.Bytes()) {
				t.Fatalf("expected valid json body, got %s", rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("expected body to contain %q, got %s", tc.want, rec.Body.String())
			}
		})
	}
}
