package download

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"dwizzyBRAIN/irag"
)

func TestNewServiceBuildsDownloadFamilyFromIRAGConfig(t *testing.T) {
	t.Parallel()

	var hits int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		if got := r.URL.Path; got != "/api/download/spotify" {
			t.Fatalf("unexpected upstream path: %s", got)
		}
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("download ok"))
	}))
	t.Cleanup(upstream.Close)

	service := NewService(irag.Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[irag.ProviderName]irag.UpstreamConfig{
			irag.ProviderNexure: {
				Name:    irag.ProviderNexure,
				BaseURL: mustParseURL(t, upstream.URL),
				Enabled: true,
			},
		},
	}, nil, nil)

	if !service.Enabled() {
		t.Fatal("expected service to be enabled")
	}
	if service.family == nil {
		t.Fatal("expected download family to be built")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/spotify?url=https://open.spotify.com/track/abc", nil)
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "download ok" {
		t.Fatalf("unexpected body: %q", got)
	}
	if got := rec.Header().Get("X-IRAG-Provider"); got != "n" {
		t.Fatalf("expected short provider code n, got %q", got)
	}
	if got := atomic.LoadInt32(&hits); got != 1 {
		t.Fatalf("expected one upstream hit, got %d", got)
	}
}

func TestServiceUnavailableWithoutConfiguredIRAG(t *testing.T) {
	t.Parallel()

	service := NewService(irag.Config{Timeout: time.Second}, nil, nil)

	if service.Enabled() {
		t.Fatal("expected service to be disabled")
	}
	if service.family != nil {
		t.Fatal("expected no download family when IRAG is unavailable")
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/spotify", nil)
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestServiceDelegatesToIRAGDownloadFamily(t *testing.T) {
	t.Parallel()

	var called int32
	service := &Service{
		family: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&called, 1)
			if got := r.URL.Path; got != "/v1/download/youtube" {
				t.Fatalf("unexpected path forwarded to family: %s", got)
			}
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte("delegated"))
		}),
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/youtube?url=https://youtube.com/watch?v=abc", nil)
	service.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected delegated status 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "delegated" {
		t.Fatalf("unexpected delegated body: %q", got)
	}
	if got := atomic.LoadInt32(&called); got != 1 {
		t.Fatalf("expected one delegated call, got %d", got)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()

	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("parse url %q: %v", raw, err)
	}
	return parsed
}
