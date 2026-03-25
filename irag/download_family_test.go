package irag

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestDownloadFamilyRejectsNonDownloadRoute(t *testing.T) {
	t.Parallel()

	family := NewDownloadFamily(NewService(Config{Timeout: time.Second}, nil, nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/text/groq", nil)
	family.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for non-download route, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDownloadFamilyUnknownRouteReturnsNotFound(t *testing.T) {
	t.Parallel()

	family := NewDownloadFamily(NewService(Config{Timeout: time.Second}, nil, nil))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/does-not-exist?url=https://example.com/video", nil)
	family.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown download route, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestDownloadFamilyUsesShortProviderCodes(t *testing.T) {
	t.Parallel()

	var nexureHits int32
	nexure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&nexureHits, 1)
		if got := r.URL.Path; got != "/api/download/spotify" {
			t.Fatalf("unexpected upstream path: %s", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("spotify ok"))
	}))
	t.Cleanup(nexure.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, nexure.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/spotify?url=https://open.spotify.com/track/abc", nil)
	NewDownloadFamily(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-IRAG-Provider"); got != "n" {
		t.Fatalf("expected short provider code n, got %q", got)
	}
	if got := rec.Header().Get("X-IRAG-Upstream"); got != "n" {
		t.Fatalf("expected short upstream code n, got %q", got)
	}
	if got := rec.Header().Get("X-IRAG-Fallback-Used"); got != "false" {
		t.Fatalf("expected no fallback, got %q", got)
	}
	if got := rec.Body.String(); got != "spotify ok" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := atomic.LoadInt32(&nexureHits); got != 1 {
		t.Fatalf("expected one nexure hit, got %d", got)
	}
}

func TestDownloadFamilySurfacesFallbackChain(t *testing.T) {
	t.Parallel()

	var kanataHits int32
	kanata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&kanataHits, 1)
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("kanata failed"))
	}))
	t.Cleanup(kanata.Close)

	var nexureHits int32
	nexure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&nexureHits, 1)
		if got := r.URL.Path; got != "/api/download/youtube" {
			t.Fatalf("unexpected upstream path: %s", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("nexure fallback ok"))
	}))
	t.Cleanup(nexure.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, kanata.URL), Enabled: true},
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, nexure.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/youtube/video?url=https://youtube.com/watch?v=abc", nil)
	NewDownloadFamily(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("X-IRAG-Provider"); got != "n" {
		t.Fatalf("expected fallback provider code n, got %q", got)
	}
	if got := rec.Header().Get("X-IRAG-Upstream"); got != "n" {
		t.Fatalf("expected fallback upstream code n, got %q", got)
	}
	if got := rec.Header().Get("X-IRAG-Fallback-Used"); got != "true" {
		t.Fatalf("expected fallback used header true, got %q", got)
	}
	if got := rec.Body.String(); got != "nexure fallback ok" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := atomic.LoadInt32(&kanataHits); got != 1 {
		t.Fatalf("expected one kanata hit, got %d", got)
	}
	if got := atomic.LoadInt32(&nexureHits); got != 1 {
		t.Fatalf("expected one nexure hit, got %d", got)
	}
}
