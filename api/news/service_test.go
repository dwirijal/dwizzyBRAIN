package newsapi

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeArticleRow struct {
	scanFn func(dest ...any) error
}

func (f fakeArticleRow) Scan(dest ...any) error {
	return f.scanFn(dest...)
}

func TestServiceIsCategory(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "exact", value: "defi", want: true},
		{name: "normalized", value: "  ReGuLaTiOn ", want: true},
		{name: "unknown", value: "random-topic", want: false},
		{name: "empty", value: "", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := svc.IsCategory(tc.value); got != tc.want {
				t.Fatalf("IsCategory(%q) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}

func TestClampListLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "negative", limit: -1, want: defaultListLimit},
		{name: "zero", limit: 0, want: defaultListLimit},
		{name: "within", limit: 1, want: 1},
		{name: "above-max", limit: maxListLimit + 1, want: maxListLimit},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := clampListLimit(tc.limit); got != tc.want {
				t.Fatalf("clampListLimit(%d) = %d, want %d", tc.limit, got, tc.want)
			}
		})
	}
}

func TestClampTrendingLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "negative", limit: -1, want: defaultTrendingLimit},
		{name: "zero", limit: 0, want: defaultTrendingLimit},
		{name: "within", limit: 7, want: 7},
		{name: "above-max", limit: 51, want: 50},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := clampTrendingLimit(tc.limit); got != tc.want {
				t.Fatalf("clampTrendingLimit(%d) = %d, want %d", tc.limit, got, tc.want)
			}
		})
	}
}

func TestNullHelpers(t *testing.T) {
	t.Parallel()

	if got := nullFloat64Value(sql.NullFloat64{}, 0.5); got != 0.5 {
		t.Fatalf("nullFloat64Value fallback = %v, want 0.5", got)
	}
	if ptr := nullFloat64Ptr(sql.NullFloat64{}); ptr != nil {
		t.Fatalf("nullFloat64Ptr invalid = %v, want nil", *ptr)
	}
	if ptr := nullIntPtr(sql.NullInt64{}); ptr != nil {
		t.Fatalf("nullIntPtr invalid = %v, want nil", *ptr)
	}
	if ptr := nullTimePtr(sql.NullTime{}); ptr != nil {
		t.Fatalf("nullTimePtr invalid = %v, want nil", *ptr)
	}

	f := 1.25
	if got := nullFloat64Value(sql.NullFloat64{Valid: true, Float64: f}, 0); got != f {
		t.Fatalf("nullFloat64Value valid = %v, want %v", got, f)
	}
	if ptr := nullFloat64Ptr(sql.NullFloat64{Valid: true, Float64: f}); ptr == nil || *ptr != f {
		t.Fatalf("nullFloat64Ptr valid = %v, want %v", ptr, f)
	}
	if ptr := nullIntPtr(sql.NullInt64{Valid: true, Int64: 42}); ptr == nil || *ptr != 42 {
		t.Fatalf("nullIntPtr valid = %v, want 42", ptr)
	}

	local := time.Date(2026, 3, 19, 2, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	ptr := nullTimePtr(sql.NullTime{Valid: true, Time: local})
	if ptr == nil {
		t.Fatal("nullTimePtr valid = nil, want non-nil")
	}
	if ptr.Location() != time.UTC {
		t.Fatalf("nullTimePtr zone = %v, want UTC", ptr.Location())
	}
	if !ptr.Equal(local.UTC()) {
		t.Fatalf("nullTimePtr time = %v, want %v", ptr, local.UTC())
	}
}

func TestServiceMethodsRequireDB(t *testing.T) {
	t.Parallel()

	svc := NewService(nil)
	ctx := context.Background()
	const expected = "postgres pool is required"

	assertErr := func(name string, err error) {
		t.Helper()
		if err == nil {
			t.Fatalf("%s: expected error", name)
		}
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("%s: error = %q, want contains %q", name, err.Error(), expected)
		}
	}

	_, err := svc.List(ctx, 10, 0, "")
	assertErr("List", err)

	_, err = svc.Detail(ctx, 1)
	assertErr("Detail", err)

	_, err = svc.ByCoin(ctx, "bitcoin", 10, 0)
	assertErr("ByCoin", err)

	_, err = svc.Trending(ctx, time.Hour, 5)
	assertErr("Trending", err)
}

func TestScanArticleRow(t *testing.T) {
	t.Parallel()

	published := time.Date(2026, 3, 19, 2, 30, 0, 0, time.FixedZone("UTC+7", 7*60*60))
	fetched := published.Add(30 * time.Minute)
	processed := fetched.Add(1 * time.Minute)
	latency := int64(123)
	row := fakeArticleRow{
		scanFn: func(dest ...any) error {
			*(dest[0].(*int64)) = 1001
			*(dest[1].(*string)) = "ext-1"
			*(dest[2].(*string)) = "coindesk"
			*(dest[3].(*string)) = "CoinDesk"
			*(dest[4].(*sql.NullFloat64)) = sql.NullFloat64{Valid: true, Float64: 0.9}
			*(dest[5].(*string)) = "Title"
			*(dest[6].(*string)) = "Preview"
			*(dest[7].(*string)) = "https://example.com/news/1"
			*(dest[8].(*string)) = "https://example.com/image.png"
			*(dest[9].(*string)) = "author"
			*(dest[10].(*time.Time)) = published
			*(dest[11].(*time.Time)) = fetched
			*(dest[12].(*bool)) = true
			*(dest[13].(*sql.NullInt64)) = sql.NullInt64{Valid: true, Int64: 1}
			*(dest[14].(*sql.NullString)) = sql.NullString{Valid: true, String: "short"}
			*(dest[15].(*sql.NullString)) = sql.NullString{Valid: true, String: "long"}
			*(dest[16].(*[]string)) = []string{"k1", "k2"}
			*(dest[17].(*sql.NullString)) = sql.NullString{Valid: true, String: "positive"}
			*(dest[18].(*sql.NullFloat64)) = sql.NullFloat64{Valid: true, Float64: 0.75}
			*(dest[19].(*sql.NullString)) = sql.NullString{Valid: true, String: "defi"}
			*(dest[20].(*sql.NullString)) = sql.NullString{Valid: true, String: "lending"}
			*(dest[21].(*sql.NullFloat64)) = sql.NullFloat64{Valid: true, Float64: 0.82}
			*(dest[22].(*sql.NullBool)) = sql.NullBool{Valid: true, Bool: true}
			*(dest[23].(*sql.NullString)) = sql.NullString{Valid: true, String: "market-move"}
			*(dest[24].(*sql.NullString)) = sql.NullString{Valid: true, String: "model-x"}
			*(dest[25].(*sql.NullInt64)) = sql.NullInt64{Valid: true, Int64: latency}
			*(dest[26].(*sql.NullTime)) = sql.NullTime{Valid: true, Time: processed}
			return nil
		},
	}

	item, meta, err := scanArticleRow(row)
	if err != nil {
		t.Fatalf("scanArticleRow error: %v", err)
	}
	if item.ID != 1001 || item.ExternalID != "ext-1" {
		t.Fatalf("unexpected article identity: %+v", item)
	}
	if item.PublishedAt.Location() != time.UTC || item.FetchedAt.Location() != time.UTC {
		t.Fatalf("expected UTC times, got published=%v fetched=%v", item.PublishedAt.Location(), item.FetchedAt.Location())
	}
	if meta == nil {
		t.Fatal("expected metadata")
	}
	if meta.ImportanceScore == nil || *meta.ImportanceScore != 0.82 {
		t.Fatalf("unexpected importance score: %+v", meta.ImportanceScore)
	}
	if meta.ProcessingLatencyMS == nil || *meta.ProcessingLatencyMS != int(latency) {
		t.Fatalf("unexpected latency: %+v", meta.ProcessingLatencyMS)
	}
	if meta.ProcessedAt == nil || !meta.ProcessedAt.Equal(processed.UTC()) {
		t.Fatalf("unexpected processed_at: %+v", meta.ProcessedAt)
	}
}

func TestScanArticleRowErrorWrap(t *testing.T) {
	t.Parallel()

	row := fakeArticleRow{
		scanFn: func(dest ...any) error {
			return errors.New("boom")
		},
	}

	_, _, err := scanArticleRow(row)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped scan error, got %q", err.Error())
	}
}
