package irag

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

func TestIragFallbackJSONEnvelope(t *testing.T) {
	t.Parallel()

	var nexureHits int32
	nexure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&nexureHits, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"down"}`))
	}))
	t.Cleanup(nexure.Close)

	ryzumi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/api/ai/chatgpt" {
			t.Fatalf("unexpected upstream path: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":{"text":"hello from irag"}}`))
	}))
	t.Cleanup(ryzumi.Close)

	service := NewService(Config{
		Timeout:        2 * time.Second,
		CacheEnabled:   false,
		AllowedOrigins: []string{"*"},
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, nexure.URL), Enabled: true},
			ProviderRyzumi: {Name: ProviderRyzumi, BaseURL: mustParseURL(t, ryzumi.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/text/groq?ask=hello", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true, got %v", payload["ok"])
	}
	data, _ := payload["data"].(map[string]any)
	if data["text"] != "hello from irag" {
		t.Fatalf("unexpected data: %#v", data)
	}
	if atomic.LoadInt32(&nexureHits) != 1 {
		t.Fatalf("expected one nexure attempt, got %d", nexureHits)
	}
}

func TestIragAITextPathTranslation(t *testing.T) {
	t.Parallel()

	var nexureSeenPath, nexureSeenQuery string
	nexure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nexureSeenPath = r.URL.Path
		nexureSeenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":"hello from nexure"}`))
	}))
	t.Cleanup(nexure.Close)

	var ryzumiSeenPath, ryzumiSeenQuery string
	ryzumi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ryzumiSeenPath = r.URL.Path
		ryzumiSeenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":"hello from ryzumi"}`))
	}))
	t.Cleanup(ryzumi.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, nexure.URL), Enabled: true},
			ProviderRyzumi: {Name: ProviderRyzumi, BaseURL: mustParseURL(t, ryzumi.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/text/gpt?ask=Hello+world&model=gpt-5-mini", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected gpt 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if nexureSeenPath != "/api/ai/gpt" {
		t.Fatalf("unexpected nexure path: %s", nexureSeenPath)
	}
	nexureQuery, _ := url.ParseQuery(nexureSeenQuery)
	if got := nexureQuery.Get("ask"); got != "Hello world" {
		t.Fatalf("unexpected nexure ask: %s", got)
	}
	if got := nexureQuery.Get("model"); got != "gpt-5-mini" {
		t.Fatalf("unexpected nexure model: %s", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/ai/text/chatgpt-ryz?ask=Halo+dunia&session=abc123", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ryzumi 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ryzumiSeenPath != "/api/ai/chatgpt" {
		t.Fatalf("unexpected ryzumi path: %s", ryzumiSeenPath)
	}
	ryzumiQuery, _ := url.ParseQuery(ryzumiSeenQuery)
	if got := ryzumiQuery.Get("text"); got != "Halo dunia" {
		t.Fatalf("unexpected ryzumi text: %s", got)
	}
	if got := ryzumiQuery.Get("prompt"); got != "Halo dunia" {
		t.Fatalf("unexpected ryzumi prompt: %s", got)
	}
	if got := ryzumiQuery.Get("session"); got != "abc123" {
		t.Fatalf("unexpected ryzumi session: %s", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/ai/text/groq?ask=Hello+world", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected groq 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if nexureSeenPath != "/api/ai/groq" {
		t.Fatalf("unexpected groq path: %s", nexureSeenPath)
	}
	nexureQuery, _ = url.ParseQuery(nexureSeenQuery)
	if got := nexureQuery.Get("model"); got != "groq/compound" {
		t.Fatalf("unexpected groq model: %s", got)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/ai/text/qwen?ask=Hello+world", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected qwen 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ryzumiSeenPath != "/api/ai/qwen" {
		t.Fatalf("unexpected qwen path: %s", ryzumiSeenPath)
	}
	ryzumiQuery, _ = url.ParseQuery(ryzumiSeenQuery)
	if got := ryzumiQuery.Get("model"); got != "qwen3-coder-plus" {
		t.Fatalf("unexpected qwen model: %s", got)
	}
}

func TestIragAITextYTDLPTranslation(t *testing.T) {
	t.Parallel()

	var seenPath, seenQuery string
	ytdlp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"answer":"hello from ytdlp"}`))
	}))
	t.Cleanup(ytdlp.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderYTDLP: {Name: ProviderYTDLP, BaseURL: mustParseURL(t, ytdlp.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/text/gemini?ask=Hello+world", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected ytdlp 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/ai/gemini" {
		t.Fatalf("unexpected ytdlp path: %s", seenPath)
	}
	ytdlpQuery, _ := url.ParseQuery(seenQuery)
	if got := ytdlpQuery.Get("text"); got != "Hello world" {
		t.Fatalf("unexpected ytdlp text: %s", got)
	}
	if got := ytdlpQuery.Get("model"); got != "gemma-3-27b-it" {
		t.Fatalf("unexpected ytdlp model: %s", got)
	}
}

func TestIragLLMChatGPTCompletionsTranslation(t *testing.T) {
	t.Parallel()

	var seenPath, seenQuery string
	chocomilk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":"hello from chocomilk"}`))
	}))
	t.Cleanup(chocomilk.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderChocomilk: {Name: ProviderChocomilk, BaseURL: mustParseURL(t, chocomilk.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/llm/chatgpt/completions?prompt=Hello+world&model=gpt-4o-mini", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected llm 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/api/ai/chatgpt" {
		t.Fatalf("unexpected chocomilk path: %s", seenPath)
	}
	gotQuery, _ := url.ParseQuery(seenQuery)
	if got := gotQuery.Get("ask"); got != "Hello world" {
		t.Fatalf("unexpected ask: %s", got)
	}
	if got := gotQuery.Get("prompt"); got != "Hello world" {
		t.Fatalf("unexpected prompt: %s", got)
	}
	if got := gotQuery.Get("text"); got != "Hello world" {
		t.Fatalf("unexpected text: %s", got)
	}
	if got := gotQuery.Get("model"); got != "gpt-4o-mini" {
		t.Fatalf("unexpected model: %s", got)
	}
}

func TestIragYouTubeChocomilkTranslation(t *testing.T) {
	t.Parallel()

	var seenPath, seenQuery string
	chocomilk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":"hello from chocomilk youtube"}`))
	}))
	t.Cleanup(chocomilk.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderChocomilk: {Name: ProviderChocomilk, BaseURL: mustParseURL(t, chocomilk.URL), Enabled: true},
		},
	}, nil, nil)

	cases := []struct {
		name     string
		path     string
		wantPath string
		wantQ    string
		wantURL  string
	}{
		{
			name:     "search",
			path:     "/v1/youtube/search?q=lofi&page=2",
			wantPath: "/v1/youtube/search",
			wantQ:    "lofi",
		},
		{
			name:     "play",
			path:     "/v1/youtube/play?ask=lofi+beats",
			wantPath: "/v1/youtube/play",
			wantQ:    "lofi beats",
		},
		{
			name:     "info",
			path:     "/v1/youtube/info?url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3Dabc",
			wantPath: "/v1/youtube/info",
			wantURL:  "https://youtube.com/watch?v=abc",
		},
		{
			name:     "download",
			path:     "/v1/youtube/download?url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3Dabc&quality=1080",
			wantPath: "/v1/youtube/download",
			wantURL:  "https://youtube.com/watch?v=abc",
		},
	}

	for _, tt := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tt.path, nil)
		NewRouter(service).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d: %s", tt.name, rec.Code, rec.Body.String())
		}
		if seenPath != tt.wantPath {
			t.Fatalf("%s unexpected path: got %q want %q", tt.name, seenPath, tt.wantPath)
		}
		gotQuery, _ := url.ParseQuery(seenQuery)
		if tt.wantQ != "" {
			if got := gotQuery.Get("q"); got != tt.wantQ {
				t.Fatalf("%s unexpected q: got %q want %q", tt.name, got, tt.wantQ)
			}
		}
		if tt.wantURL != "" {
			if got := gotQuery.Get("url"); got != tt.wantURL {
				t.Fatalf("%s unexpected url: got %q want %q", tt.name, got, tt.wantURL)
			}
		}
	}
}

func TestIragYouTubeYTDLPFallbackTranslation(t *testing.T) {
	t.Parallel()

	var ytdlpPath, ytdlpQuery string
	ytdlp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ytdlpPath = r.URL.Path
		ytdlpQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":"hello from ytdlp youtube"}`))
	}))
	t.Cleanup(ytdlp.Close)

	chocomilk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(chocomilk.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderChocomilk: {Name: ProviderChocomilk, BaseURL: mustParseURL(t, chocomilk.URL), Enabled: true},
			ProviderYTDLP:     {Name: ProviderYTDLP, BaseURL: mustParseURL(t, ytdlp.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/youtube/download?url=https%3A%2F%2Fyoutube.com%2Fwatch%3Fv%3Dabc&quality=1080", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected fallback 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ytdlpPath != "/download/" {
		t.Fatalf("unexpected ytdlp path: %s", ytdlpPath)
	}
	gotQuery, _ := url.ParseQuery(ytdlpQuery)
	if got := gotQuery.Get("url"); got != "https://youtube.com/watch?v=abc" {
		t.Fatalf("unexpected ytdlp url: %s", got)
	}
}

func TestIragAIImagePathTranslation(t *testing.T) {
	t.Parallel()

	var seenPath, seenQuery string
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(image.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, image.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/image/deepimg?prompt=girl+wearing+glasses&style=anime&size=1:1", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected image 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/api/ai/deepimg" {
		t.Fatalf("unexpected image path: %s", seenPath)
	}
	imageQuery, _ := url.ParseQuery(seenQuery)
	if got := imageQuery.Get("prompt"); got != "girl wearing glasses" {
		t.Fatalf("unexpected prompt: %s", got)
	}
	if got := imageQuery.Get("style"); got != "anime" {
		t.Fatalf("unexpected style: %s", got)
	}
	if got := imageQuery.Get("size"); got != "1:1" {
		t.Fatalf("unexpected size: %s", got)
	}
}

func TestIragAIDirectAliasTranslation(t *testing.T) {
	t.Parallel()

	var textPath string
	nexure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		textPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":"hello from direct ai"}`))
	}))
	t.Cleanup(nexure.Close)

	var imagePath string
	kanata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		imagePath = r.URL.Path
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png"))
	}))
	t.Cleanup(kanata.Close)

	var ytdlpPath string
	ytdlp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ytdlpPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"result":"hello from direct ytdlp"}`))
	}))
	t.Cleanup(ytdlp.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, nexure.URL), Enabled: true},
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, kanata.URL), Enabled: true},
			ProviderYTDLP:  {Name: ProviderYTDLP, BaseURL: mustParseURL(t, ytdlp.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/ai4chat?ask=Hello+world", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected direct text 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if textPath != "/api/ai/ai4chat" {
		t.Fatalf("unexpected direct text path: %s", textPath)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/ai/animagine-xl-3?prompt=girl+cat", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected direct image 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if imagePath != "/ai/image" {
		t.Fatalf("unexpected direct image path: %s", imagePath)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/ai/gemini?ask=Hello+world", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected direct gemini 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ytdlpPath != "/ai/gemini" {
		t.Fatalf("unexpected direct gemini path: %s", ytdlpPath)
	}
}

func TestIragDirectAliasGuardsUnknownSlugs(t *testing.T) {
	t.Parallel()

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/unknown-slug", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected ai unknown slug 404, got %d: %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/v1/youtube/unknown-slug", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected youtube unknown slug 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestIragAIGenerateExactPathTranslation(t *testing.T) {
	t.Parallel()

	var seenPath, seenQuery string
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"code":200,"result":{"url":"https://cdn.example.com/image.png"}}`))
	}))
	t.Cleanup(image.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, image.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/generate?prompt=a+blue+rose", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected generate 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/ai/generate" {
		t.Fatalf("unexpected generate path: %s", seenPath)
	}
	values, _ := url.ParseQuery(seenQuery)
	if got := values.Get("prompt"); got != "a blue rose" {
		t.Fatalf("unexpected prompt: %s", got)
	}
}

func TestIragAIImageExactPathTranslation(t *testing.T) {
	t.Parallel()

	var seenPath, seenQuery string
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(image.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, image.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/image?prompt=a+red+cat", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected image 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/ai/image" {
		t.Fatalf("unexpected image path: %s", seenPath)
	}
	values, _ := url.ParseQuery(seenQuery)
	if got := values.Get("prompt"); got != "a red cat" {
		t.Fatalf("unexpected prompt: %s", got)
	}
}

func TestIragAIImageKanataPathTranslation(t *testing.T) {
	t.Parallel()

	var seenPath, seenQuery string
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(image.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, image.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/image/animagine-xl-3?prompt=1girl%2C+blue+eyes", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected image 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/ai/image" {
		t.Fatalf("unexpected kanata path: %s", seenPath)
	}
	imageQuery, _ := url.ParseQuery(seenQuery)
	if got := imageQuery.Get("prompt"); got != "1girl, blue eyes" {
		t.Fatalf("unexpected prompt: %s", got)
	}
}

func TestIragAIImageFallsBackToKanataOnBadRequest(t *testing.T) {
	t.Parallel()

	nexure := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad request"}`))
	}))
	t.Cleanup(nexure.Close)

	var kanataPath string
	kanata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kanataPath = r.URL.Path
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(kanata.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, nexure.URL), Enabled: true},
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, kanata.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/image/flux-schnell?prompt=a+cat+sitting+on+a+windowsill", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected fallback 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("unexpected fallback content-type: %s", got)
	}
	if kanataPath != "/ai/image" {
		t.Fatalf("unexpected kanata fallback path: %s", kanataPath)
	}
}

func TestIragAIImageRouteUsesLongTimeout(t *testing.T) {
	t.Parallel()

	var seenPath string
	image := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(image.Close)

	service := NewService(Config{
		Timeout:      10 * time.Millisecond,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, image.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/ai/image/deepimg?prompt=girl+wearing+glasses&style=anime&size=1:1", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected image 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("unexpected image content-type: %s", got)
	}
	if got := rec.Body.String(); got != "png-bytes" {
		t.Fatalf("unexpected image body: %q", got)
	}
	if seenPath != "/api/ai/deepimg" {
		t.Fatalf("unexpected upstream path: %s", seenPath)
	}
}

func TestIragAIProcessPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		provider  ProviderName
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "toanime nexure",
			path:     "/v1/ai/process/toanime?url=https%3A%2F%2Fexample.com%2Fimg.png&style=anime",
			provider: ProviderNexure,
			wantPath: "/api/ai/toanime",
			wantQuery: map[string]string{
				"url":       "https://example.com/img.png",
				"image_url": "https://example.com/img.png",
				"imgUrl":    "https://example.com/img.png",
				"style":     "anime",
			},
		},
		{
			name:     "colorize ryzumi",
			path:     "/v1/ai/process/colorize?url=https%3A%2F%2Fexample.com%2Fbw.png",
			provider: ProviderRyzumi,
			wantPath: "/api/ai/colorize",
			wantQuery: map[string]string{
				"url":       "https://example.com/bw.png",
				"image_url": "https://example.com/bw.png",
				"imgUrl":    "https://example.com/bw.png",
			},
		},
		{
			name:     "faceswap ryzumi",
			path:     "/v1/ai/process/faceswap?original=https%3A%2F%2Fexample.com%2Foriginal.png&face=https%3A%2F%2Fexample.com%2Fface.png",
			provider: ProviderRyzumi,
			wantPath: "/api/ai/faceswap",
			wantQuery: map[string]string{
				"original": "https://example.com/original.png",
				"face":     "https://example.com/face.png",
			},
		},
		{
			name:     "upscale ryzumi",
			path:     "/v1/ai/process/upscale?url=https%3A%2F%2Fexample.com%2Fimg.png&scale=4",
			provider: ProviderRyzumi,
			wantPath: "/api/ai/upscaler",
			wantQuery: map[string]string{
				"url":       "https://example.com/img.png",
				"image_url": "https://example.com/img.png",
				"imgUrl":    "https://example.com/img.png",
				"scale":     "4",
			},
		},
		{
			name:     "enhance2x chocomilk",
			path:     "/v1/i2i/enhance2x?url=https%3A%2F%2Fexample.com%2Fimg.png",
			provider: ProviderChocomilk,
			wantPath: "/v1/i2i/enhance",
			wantQuery: map[string]string{
				"url":       "https://example.com/img.png",
				"image_url": "https://example.com/img.png",
				"imgUrl":    "https://example.com/img.png",
			},
		},
		{
			name:     "nanobanana chocomilk",
			path:     "/v1/i2i/nano-banana?url=https%3A%2F%2Fexample.com%2Fimg.png&prompt=edit+this",
			provider: ProviderChocomilk,
			wantPath: "/v1/i2i/nano-banana",
			wantQuery: map[string]string{
				"url":       "https://example.com/img.png",
				"image_url": "https://example.com/img.png",
				"imgUrl":    "https://example.com/img.png",
			},
		},
		{
			name:     "nsfw-check nexure",
			path:     "/v1/ai/process/nsfw-check?url=https%3A%2F%2Fexample.com%2Fimg.png",
			provider: ProviderNexure,
			wantPath: "/api/tools/nsfw-check",
			wantQuery: map[string]string{
				"url":       "https://example.com/img.png",
				"image_url": "https://example.com/img.png",
				"imgUrl":    "https://example.com/img.png",
			},
		},
		{
			name:     "image2txt ryzumi",
			path:     "/v1/ai/process/image2txt?url=https%3A%2F%2Fexample.com%2Fimg.png",
			provider: ProviderRyzumi,
			wantPath: "/api/ai/image2txt",
			wantQuery: map[string]string{
				"url":       "https://example.com/img.png",
				"image_url": "https://example.com/img.png",
				"imgUrl":    "https://example.com/img.png",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				if strings.HasPrefix(tt.wantPath, "/api/") || strings.HasPrefix(tt.wantPath, "/v1/") {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"success":true,"result":"ok"}`))
					return
				}
				w.Header().Set("Content-Type", "image/png")
				_, _ = w.Write([]byte("png-bytes"))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected upstream path: got %q want %q", seenPath, tt.wantPath)
			}
			gotQuery, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := gotQuery.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
				}
			}
		})
	}
}

func TestIragWeebsPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "anime info",
			path:     "/v1/weebs/anime-info?query=shingeki+no+kyojin",
			wantPath: "/api/weebs/anime-info",
			wantQuery: map[string]string{
				"query": "shingeki no kyojin",
			},
		},
		{
			name:     "manga info",
			path:     "/v1/weebs/manga-info?query=naruto",
			wantPath: "/api/weebs/manga-info",
			wantQuery: map[string]string{
				"query": "naruto",
			},
		},
		{
			name:     "sfw waifu",
			path:     "/v1/weebs/sfw-waifu?tag=waifu",
			wantPath: "/api/weebs/sfw-waifu",
			wantQuery: map[string]string{
				"tag": "waifu",
			},
		},
		{
			name:     "whatanime",
			path:     "/v1/weebs/whatanime?url=https%3A%2F%2Fexample.com%2Fanime.png",
			wantPath: "/api/weebs/whatanime",
			wantQuery: map[string]string{
				"url": "https://example.com/anime.png",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"status":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					ProviderRyzumi: {Name: ProviderRyzumi, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected upstream path: got %q want %q", seenPath, tt.wantPath)
			}
			gotQuery, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := gotQuery.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
				}
			}
		})
	}
}

func TestIragNovelPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "home alias",
			path:     "/v1/novel",
			wantPath: "/v1/novel/home",
		},
		{
			name:     "hot search alias",
			path:     "/v1/novel/hot",
			wantPath: "/v1/novel/hot-search",
		},
		{
			name:     "search",
			path:     "/v1/novel/search?q=naruto&page=2",
			wantPath: "/v1/novel/search",
			wantQuery: map[string]string{
				"q":     "naruto",
				"query": "naruto",
				"page":  "2",
			},
		},
		{
			name:     "genre",
			path:     "/v1/novel/genre?genre=fantasy&page=3",
			wantPath: "/v1/novel/genre",
			wantQuery: map[string]string{
				"genre": "fantasy",
				"q":     "fantasy",
				"page":  "3",
			},
		},
		{
			name:     "chapters",
			path:     "/v1/novel/chapters?url=https%3A%2F%2Fexample.com%2Fchapter",
			wantPath: "/v1/novel/chapters",
			wantQuery: map[string]string{
				"url": "https://example.com/chapter",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					ProviderChocomilk: {Name: ProviderChocomilk, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: got %s want %s", seenPath, tt.wantPath)
			}
			values, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := values.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
				}
			}
		})
	}
}

func TestIragIslamicPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "quran list",
			path:     "/v1/islamic/quran",
			wantPath: "/quran/surah",
		},
		{
			name:     "quran detail",
			path:     "/v1/islamic/quran/1",
			wantPath: "/quran/surah/1",
		},
		{
			name:     "tafsir",
			path:     "/v1/islamic/tafsir?surah=1",
			wantPath: "/tafsir",
			wantQuery: map[string]string{
				"surah": "1",
			},
		},
		{
			name:     "topegon",
			path:     "/v1/islamic/topegon?text=saya+pergi+ke+masjid",
			wantPath: "/topegon",
			wantQuery: map[string]string{
				"text": "saya pergi ke masjid",
			},
		},
		{
			name:     "hadith",
			path:     "/v1/islamic/hadith/bukhari/1",
			wantPath: "/hadits",
			wantQuery: map[string]string{
				"book":      "bukhari",
				"hadith_id": "1",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			ytdlp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(ytdlp.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					ProviderYTDLP: {Name: ProviderYTDLP, BaseURL: mustParseURL(t, ytdlp.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected upstream path: %s", seenPath)
			}
			if len(tt.wantQuery) > 0 {
				gotQuery, _ := url.ParseQuery(seenQuery)
				for key, want := range tt.wantQuery {
					if got := gotQuery.Get(key); got != want {
						t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
					}
				}
			}
		})
	}
}

func TestIragToolsPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		provider  ProviderName
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "translate",
			path:     "/v1/tools/translate?text=hello&to=id&from=en",
			provider: ProviderKanata,
			wantPath: "/googletranslate",
			wantQuery: map[string]string{
				"text": "hello",
				"to":   "id",
				"from": "en",
			},
		},
		{
			name:     "weather",
			path:     "/v1/tools/weather?city=jakarta",
			provider: ProviderNexure,
			wantPath: "/api/info/weather",
			wantQuery: map[string]string{
				"city": "jakarta",
			},
		},
		{
			name:     "cekresi",
			path:     "/v1/tools/cekresi?resi=JP123456789ID&ekspedisi=jne",
			provider: ProviderNexure,
			wantPath: "/api/tools/cekresi",
			wantQuery: map[string]string{
				"noresi":    "JP123456789ID",
				"resi":      "JP123456789ID",
				"ekspedisi": "jne",
			},
		},
		{
			name:     "qr",
			path:     "/v1/tools/qr?text=hello&frame=qrcg-scan-me-bottom-frame",
			provider: ProviderNexure,
			wantPath: "/api/image/qr",
			wantQuery: map[string]string{
				"text":  "hello",
				"frame": "qrcg-scan-me-bottom-frame",
			},
		},
		{
			name:     "ssweb",
			path:     "/v1/tools/ssweb?url=example.com&mode=desktop",
			provider: ProviderNexure,
			wantPath: "/api/tools/ssweb",
			wantQuery: map[string]string{
				"url":  "example.com",
				"mode": "desktop",
			},
		},
		{
			name:     "ipinfo",
			path:     "/v1/tools/ipinfo/8.8.8.8",
			provider: ProviderKanata,
			wantPath: "/ipinfo/8.8.8.8",
		},
		{
			name:     "carbon",
			path:     "/v1/tools/carbon?code=console.log(1)",
			provider: ProviderKanata,
			wantPath: "/carbon",
			wantQuery: map[string]string{
				"code": "console.log(1)",
			},
		},
		{
			name:     "removebg",
			path:     "/v1/tools/removebg?image_url=https%3A%2F%2Fexample.com%2Fimg.png&mode=url",
			provider: ProviderYTDLP,
			wantPath: "/removebg",
			wantQuery: map[string]string{
				"image_url": "https://example.com/img.png",
				"mode":      "url",
			},
		},
		{
			name:     "pln",
			path:     "/v1/tools/pln?id_pel=520522604488",
			provider: ProviderNexure,
			wantPath: "/api/tools/pln",
			wantQuery: map[string]string{
				"id":     "520522604488",
				"id_pel": "520522604488",
			},
		},
		{
			name:     "pajak",
			path:     "/v1/tools/pajak/jabar?plat=B1234XYZ",
			provider: ProviderNexure,
			wantPath: "/api/tools/cek-pajak/jabar",
			wantQuery: map[string]string{
				"plat": "B 1234 XYZ",
			},
		},
		{
			name:     "brat",
			path:     "/v1/tools/brat?text=dwizzyOS",
			provider: ProviderNexure,
			wantPath: "/api/image/brat",
			wantQuery: map[string]string{
				"text": "dwizzyOS",
			},
		},
		{
			name:     "brat animated",
			path:     "/v1/tools/brat/animated?text=dwizzyOS",
			provider: ProviderNexure,
			wantPath: "/api/image/brat/animated",
			wantQuery: map[string]string{
				"text": "dwizzyOS",
			},
		},
		{
			name:     "iphonechat",
			path:     "/v1/tools/iphonechat?text=halo&user=dwizzy&jam=08%3A00&profile_url=https%3A%2F%2Fexample.com%2Favatar.png",
			provider: ProviderYTDLP,
			wantPath: "/maker/iqc",
			wantQuery: map[string]string{
				"text":        "halo",
				"user":        "dwizzy",
				"jam":         "08:00",
				"profile_url": "https://example.com/avatar.png",
			},
		},
		{
			name:     "shorturl",
			path:     "/v1/tools/shorturl?long_url=https%3A%2F%2Fexample.com%2Ffoo",
			provider: ProviderYTDLP,
			wantPath: "/shorturl",
			wantQuery: map[string]string{
				"url":      "https://example.com/foo",
				"long_url": "https://example.com/foo",
			},
		},
		{
			name:     "distance",
			path:     "/v1/tools/distance?from=Jakarta&to=Bandung",
			provider: ProviderYTDLP,
			wantPath: "/jarak",
			wantQuery: map[string]string{
				"from": "Jakarta",
				"to":   "Bandung",
				"dari": "Jakarta",
				"ke":   "Bandung",
			},
		},
		{
			name:     "gsmarena",
			path:     "/v1/tools/gsmarena?device=iPhone+15",
			provider: ProviderYTDLP,
			wantPath: "/gsmarena",
			wantQuery: map[string]string{
				"q":      "iPhone 15",
				"device": "iPhone 15",
			},
		},
		{
			name:     "cekbank",
			path:     "/v1/tools/cekbank?bank=BCA&no_rek=12345678",
			provider: ProviderYTDLP,
			wantPath: "/cekbank",
			wantQuery: map[string]string{
				"bank_code":      "BCA",
				"bank":           "BCA",
				"account_number": "12345678",
				"no_rek":         "12345678",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected upstream path: %s", seenPath)
			}
			if len(tt.wantQuery) > 0 {
				gotQuery, _ := url.ParseQuery(seenQuery)
				for key, want := range tt.wantQuery {
					if got := gotQuery.Get(key); got != want {
						t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
					}
				}
			}
		})
	}
}

func TestIragBMKGPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "earthquake",
			path:     "/v1/bmkg/earthquake",
			wantPath: "/bmkg/gempa",
		},
		{
			name:     "earthquake felt",
			path:     "/v1/bmkg/earthquake/felt",
			wantPath: "/bmkg/gempa/dirasakan",
		},
		{
			name:     "weather",
			path:     "/v1/bmkg/weather?provinsi=jawa-timur",
			wantPath: "/bmkg/cuaca",
			wantQuery: map[string]string{
				"provinsi": "jawa-timur",
			},
		},
		{
			name:     "weather village",
			path:     "/v1/bmkg/weather/village?adm4=31.71.03.1001",
			wantPath: "/bmkg/cuaca/desa",
			wantQuery: map[string]string{
				"adm4": "31.71.03.1001",
			},
		},
		{
			name:     "provinces",
			path:     "/v1/bmkg/provinces",
			wantPath: "/bmkg/cuaca/provinces",
		},
		{
			name:     "region search",
			path:     "/v1/bmkg/region/search?q=bandung",
			wantPath: "/bmkg/wilayah/search",
			wantQuery: map[string]string{
				"q": "bandung",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: got %s want %s", seenPath, tt.wantPath)
			}
			values, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := values.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
				}
			}
		})
	}
}

func TestIragUtilityUpscalePathTranslation(t *testing.T) {
	t.Parallel()

	var seenMethod, seenPath, seenQuery string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenMethod = r.Method
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(upstream.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/utility/upscale?imgUrl=https%3A%2F%2Fexample.com%2Fimg.png", strings.NewReader("imgUrl=https%3A%2F%2Fexample.com%2Fimg.png"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenMethod != http.MethodGet {
		t.Fatalf("unexpected upstream method: %s", seenMethod)
	}
	if seenPath != "/api/tools/upscale" {
		t.Fatalf("unexpected upstream path: %s", seenPath)
	}
	values, _ := url.ParseQuery(seenQuery)
	if got := values.Get("imgUrl"); got != "https://example.com/img.png" {
		t.Fatalf("unexpected imgUrl: %s", got)
	}
}

func TestIragToBase64LocalUtility(t *testing.T) {
	t.Parallel()

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
	}, nil, nil)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "hello.txt")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("hello irag")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/tobase64", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if ok, _ := payload["ok"].(bool); !ok {
		t.Fatalf("expected ok=true")
	}
	data, _ := payload["data"].(map[string]any)
	if got := data["base64"]; got != base64.StdEncoding.EncodeToString([]byte("hello irag")) {
		t.Fatalf("unexpected base64: %v", got)
	}
	if got := data["filename"]; got != "hello.txt" {
		t.Fatalf("unexpected filename: %v", got)
	}
}

func TestIragAnimeMangaFilmNewsPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		provider  ProviderName
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "otakudesu home alias",
			path:     "/v1/otakudesu",
			provider: ProviderNexure,
			wantPath: "/api/otakudesu",
		},
		{
			name:     "otakudesu episode alias",
			path:     "/v1/otakudesu/episode/shingeki-no-kyojin-1",
			provider: ProviderNexure,
			wantPath: "/api/otakudesu/episode/shingeki-no-kyojin-1",
		},
		{
			name:     "komiku latest alias",
			path:     "/v1/komiku",
			provider: ProviderKanata,
			wantPath: "/api/komiku/terbaru",
		},
		{
			name:     "komiku chapter alias",
			path:     "/v1/komiku/chapter/one-piece-1000",
			provider: ProviderKanata,
			wantPath: "/api/komiku/chapter",
			wantQuery: map[string]string{
				"slug": "one-piece-1000",
			},
		},
		{
			name:     "anime home",
			path:     "/v1/anime/home",
			provider: ProviderNexure,
			wantPath: "/api/otakudesu",
		},
		{
			name:     "anime search",
			path:     "/v1/anime/search?q=naruto",
			provider: ProviderNexure,
			wantPath: "/api/otakudesu/search",
			wantQuery: map[string]string{
				"q": "naruto",
			},
		},
		{
			name:     "anime detail",
			path:     "/v1/anime/detail/shingeki-no-kyojin",
			provider: ProviderNexure,
			wantPath: "/api/otakudesu/detail/shingeki-no-kyojin",
		},
		{
			name:     "anime batch",
			path:     "/v1/anime/batch/shingeki-no-kyojin",
			provider: ProviderKanata,
			wantPath: "/api/otakudesu/download/batch",
			wantQuery: map[string]string{
				"slug": "shingeki-no-kyojin",
			},
		},
		{
			name:     "manga search",
			path:     "/v1/manga/search?q=one+piece",
			provider: ProviderKanata,
			wantPath: "/api/komiku/search",
			wantQuery: map[string]string{
				"q": "one piece",
			},
		},
		{
			name:     "manga chapter",
			path:     "/v1/manga/chapter/one-piece-1000",
			provider: ProviderKanata,
			wantPath: "/api/komiku/chapter",
			wantQuery: map[string]string{
				"slug": "one-piece-1000",
			},
		},
		{
			name:     "film search",
			path:     "/v1/film/search?q=action",
			provider: ProviderKanata,
			wantPath: "/api/nontonfilm/search",
			wantQuery: map[string]string{
				"q": "action",
			},
		},
		{
			name:     "film lk21",
			path:     "/v1/lk21",
			provider: ProviderNexure,
			wantPath: "/api/lk21",
		},
		{
			name:     "film episode",
			path:     "/v1/lk21/episode/abc123",
			provider: ProviderNexure,
			wantPath: "/api/lk21/episode/abc123",
		},
		{
			name:     "news top",
			path:     "/v1/news/top",
			provider: ProviderKanata,
			wantPath: "/news/top",
		},
		{
			name:     "news cnn",
			path:     "/v1/news/cnn",
			provider: ProviderNexure,
			wantPath: "/api/info/cnn",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: got %s want %s", seenPath, tt.wantPath)
			}
			values, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := values.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
				}
			}
		})
	}
}

func TestIragToolsLongTailPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		provider  ProviderName
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "whois",
			path:     "/v1/tools/whois?domain=example.com",
			provider: ProviderNexure,
			wantPath: "/api/tool/whois",
			wantQuery: map[string]string{
				"domain": "example.com",
			},
		},
		{
			name:     "check hosting",
			path:     "/v1/tools/check-hosting?domain=example.com",
			provider: ProviderNexure,
			wantPath: "/api/tool/check-hosting",
			wantQuery: map[string]string{
				"domain": "example.com",
			},
		},
		{
			name:     "hargapangan",
			path:     "/v1/tools/hargapangan",
			provider: ProviderNexure,
			wantPath: "/api/tool/hargapangan",
		},
		{
			name:     "mc lookup",
			path:     "/v1/tools/mc-lookup?ip=play.example.com",
			provider: ProviderNexure,
			wantPath: "/api/tool/mc-lookup",
			wantQuery: map[string]string{
				"ip": "play.example.com",
			},
		},
		{
			name:     "qris converter",
			path:     "/v1/tools/qris-converter?url=https://example.com/qris.png&nominal=10000",
			provider: ProviderNexure,
			wantPath: "/api/tool/qris-converter",
			wantQuery: map[string]string{
				"url":     "https://example.com/qris.png",
				"nominal": "10000",
			},
		},
		{
			name:     "turnstile bypass",
			path:     "/v1/tools/turnstile-bypass?url=https://site.example.com",
			provider: ProviderNexure,
			wantPath: "/api/tools/turnstile-min",
			wantQuery: map[string]string{
				"url":     "https://site.example.com",
				"sitekey": "https://site.example.com",
			},
		},
		{
			name:     "turnstile sitekey",
			path:     "/v1/tools/turnstile/sitekey?url=https://site.example.com",
			provider: ProviderNexure,
			wantPath: "/api/tool/turnstile/sitekey",
			wantQuery: map[string]string{
				"url":     "https://site.example.com",
				"sitekey": "https://site.example.com",
			},
		},
		{
			name:     "yt transcript",
			path:     "/v1/tools/yt-transcript?url=https://youtube.com/watch?v=abc",
			provider: ProviderYTDLP,
			wantPath: "/api/tool/yt-transcript",
			wantQuery: map[string]string{
				"url": "https://youtube.com/watch?v=abc",
			},
		},
		{
			name:     "isrc",
			path:     "/v1/tools/isrc?isrc=USUM71703861",
			provider: ProviderYTDLP,
			wantPath: "/api/tool/isrc",
			wantQuery: map[string]string{
				"isrc": "USUM71703861",
				"q":    "USUM71703861",
			},
		},
		{
			name:     "subdofinder",
			path:     "/v1/tools/subdofinder?domain=example.com",
			provider: ProviderYTDLP,
			wantPath: "/subdofinder",
			wantQuery: map[string]string{
				"domain": "example.com",
			},
		},
		{
			name:     "currency converter",
			path:     "/v1/tools/currency-converter?amount=1&from=USD&to=IDR",
			provider: ProviderYTDLP,
			wantPath: "/kurs",
			wantQuery: map[string]string{
				"dari":   "USD",
				"ke":     "IDR",
				"jumlah": "1",
			},
		},
		{
			name:     "nsfw",
			path:     "/v1/tools/nsfw?url=https://example.com/image.png",
			provider: ProviderNexure,
			wantPath: "/api/tools/nsfw-check",
			wantQuery: map[string]string{
				"url":       "https://example.com/image.png",
				"image_url": "https://example.com/image.png",
			},
		},
		{
			name:     "cctv all",
			path:     "/v1/tools/cctv",
			provider: ProviderNexure,
			wantPath: "/api/bsw/cctv/all",
		},
		{
			name:     "cctv search",
			path:     "/v1/tools/cctv/search?query=bandung",
			provider: ProviderNexure,
			wantPath: "/api/bsw/cctv/search",
			wantQuery: map[string]string{
				"query": "bandung",
			},
		},
		{
			name:     "cctv detail",
			path:     "/v1/tools/cctv/detail/123",
			provider: ProviderNexure,
			wantPath: "/api/bsw/cctv/detail/123",
		},
		{
			name:     "dramabox home",
			path:     "/v1/dramabox",
			provider: ProviderNexure,
			wantPath: "/api/dramabox",
		},
		{
			name:     "dramabox search",
			path:     "/v1/dramabox/search?q=romance",
			provider: ProviderNexure,
			wantPath: "/api/dramabox/search",
			wantQuery: map[string]string{
				"q": "romance",
			},
		},
		{
			name:     "misc server info",
			path:     "/v1/misc/server-info",
			provider: ProviderNexure,
			wantPath: "/api/misc/server-info",
		},
		{
			name:     "misc ip whitelist check",
			path:     "/v1/misc/ip-whitelist-check?query=182.8.66.113",
			provider: ProviderRyzumi,
			wantPath: "/api/misc/ip-whitelist-check",
			wantQuery: map[string]string{
				"ip": "182.8.66.113",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: got %s want %s", seenPath, tt.wantPath)
			}
			values, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := values.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
				}
			}
		})
	}
}

func TestIragSearchAndStalkPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		provider  ProviderName
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "search youtube",
			path:     "/v1/search/youtube?q=lofi&page=2&limit=10",
			provider: ProviderRyzumi,
			wantPath: "/api/search/yt",
			wantQuery: map[string]string{
				"q":     "lofi",
				"query": "lofi",
				"page":  "2",
				"limit": "10",
			},
		},
		{
			name:     "search google image",
			path:     "/v1/search/google/image?q=bitcoin",
			provider: ProviderRyzumi,
			wantPath: "/api/search/gimage",
			wantQuery: map[string]string{
				"q":     "bitcoin",
				"query": "bitcoin",
			},
		},
		{
			name:     "search bilibili",
			path:     "/v1/search/bilibili?q=cover",
			provider: ProviderRyzumi,
			wantPath: "/api/search/bilibili",
			wantQuery: map[string]string{
				"q":     "cover",
				"query": "cover",
			},
		},
		{
			name:     "search bmkg",
			path:     "/v1/search/bmkg",
			provider: ProviderRyzumi,
			wantPath: "/api/search/bmkg",
		},
		{
			name:     "search chord",
			path:     "/v1/search/chord?q=lagu",
			provider: ProviderRyzumi,
			wantPath: "/api/search/chord",
			wantQuery: map[string]string{
				"q":     "lagu",
				"query": "lagu",
			},
		},
		{
			name:     "search anime",
			path:     "/v1/search/anime?q=naruto",
			provider: ProviderNexure,
			wantPath: "/api/otakudesu/search",
			wantQuery: map[string]string{
				"q":     "naruto",
				"query": "naruto",
			},
		},
		{
			name:     "search manga",
			path:     "/v1/search/manga?q=one+piece",
			provider: ProviderKanata,
			wantPath: "/komiku/search",
			wantQuery: map[string]string{
				"q":     "one piece",
				"query": "one piece",
			},
		},
		{
			name:     "search drama",
			path:     "/v1/search/drama?q=romance",
			provider: ProviderNexure,
			wantPath: "/api/dramabox/search",
			wantQuery: map[string]string{
				"q":     "romance",
				"query": "romance",
			},
		},
		{
			name:     "search harga emas",
			path:     "/v1/search/harga-emas",
			provider: ProviderRyzumi,
			wantPath: "/api/search/harga-emas",
		},
		{
			name:     "search jadwal sholat",
			path:     "/v1/search/jadwal-sholat?city=bandung",
			provider: ProviderRyzumi,
			wantPath: "/api/search/jadwal-sholat",
			wantQuery: map[string]string{
				"kota": "bandung",
				"city": "bandung",
			},
		},
		{
			name:     "search kurs bca",
			path:     "/v1/search/kurs-bca",
			provider: ProviderRyzumi,
			wantPath: "/api/search/kurs-bca",
		},
		{
			name:     "search lens",
			path:     "/v1/search/lens?url=https%3A%2F%2Fexample.com%2Fimg.png",
			provider: ProviderRyzumi,
			wantPath: "/api/search/lens",
			wantQuery: map[string]string{
				"url": "https://example.com/img.png",
			},
		},
		{
			name:     "search mahasiswa",
			path:     "/v1/search/mahasiswa?q=dwizzy",
			provider: ProviderRyzumi,
			wantPath: "/api/search/mahasiswa",
			wantQuery: map[string]string{
				"q":     "dwizzy",
				"query": "dwizzy",
			},
		},
		{
			name:     "search pixiv",
			path:     "/v1/search/pixiv?query=violet",
			provider: ProviderRyzumi,
			wantPath: "/api/search/pixiv",
			wantQuery: map[string]string{
				"q":     "violet",
				"query": "violet",
			},
		},
		{
			name:     "search tiktok",
			path:     "/v1/search/tiktok?q=cat",
			provider: ProviderChocomilk,
			wantPath: "/search/",
			wantQuery: map[string]string{
				"q":     "cat",
				"query": "cat",
			},
		},
		{
			name:     "search wallpaper",
			path:     "/v1/search/wallpaper?q=neon",
			provider: ProviderRyzumi,
			wantPath: "/api/search/wallpaper-moe",
			wantQuery: map[string]string{
				"q":     "neon",
				"query": "neon",
			},
		},
		{
			name:     "search film",
			path:     "/v1/search/film?q=action",
			provider: ProviderKanata,
			wantPath: "/nontonfilm/search",
			wantQuery: map[string]string{
				"q":     "action",
				"query": "action",
			},
		},
		{
			name:     "stalk instagram",
			path:     "/v1/stalk/instagram?username=dwizzy",
			provider: ProviderNexure,
			wantPath: "/api/stalk/instagram",
			wantQuery: map[string]string{
				"username": "dwizzy",
			},
		},
		{
			name:     "stalk github",
			path:     "/v1/stalk/github?username=dwizzy",
			provider: ProviderRyzumi,
			wantPath: "/api/stalk/github",
			wantQuery: map[string]string{
				"username": "dwizzy",
			},
		},
		{
			name:     "stalk mobile legends",
			path:     "/v1/stalk/mobile-legends?id=123456&server=1234",
			provider: ProviderRyzumi,
			wantPath: "/api/stalk/mobile-legends",
			wantQuery: map[string]string{
				"id":     "123456",
				"userId": "123456",
				"server": "1234",
				"zoneId": "1234",
			},
		},
		{
			name:     "stalk ml alias",
			path:     "/v1/stalk/ml?userId=123456&zoneId=1234",
			provider: ProviderRyzumi,
			wantPath: "/api/stalk/mobile-legends",
			wantQuery: map[string]string{
				"id":     "123456",
				"userId": "123456",
				"server": "1234",
				"zoneId": "1234",
			},
		},
		{
			name:     "stalk free fire alias",
			path:     "/v1/stalk/freefire?user_id=123456&zone_id=999",
			provider: ProviderRyzumi,
			wantPath: "/api/stalk/free-fire",
			wantQuery: map[string]string{
				"id":     "123456",
				"server": "999",
			},
		},
		{
			name:     "stalk valorant",
			path:     "/v1/stalk/valorant?name=dwizzy&tag=sea",
			provider: ProviderRyzumi,
			wantPath: "/api/stalk/valorant",
			wantQuery: map[string]string{
				"name": "dwizzy",
				"tag":  "sea",
			},
		},
		{
			name:     "stalk coc",
			path:     "/v1/stalk/clash-of-clans?name=dwizzy&token=abc123",
			provider: ProviderRyzumi,
			wantPath: "/api/stalk/clash-of-clans",
			wantQuery: map[string]string{
				"name":  "dwizzy",
				"token": "abc123",
			},
		},
		{
			name:     "stalk npm",
			path:     "/v1/stalk/npm?username=dwizzybrain",
			provider: ProviderRyzumi,
			wantPath: "/api/stalk/npm",
			wantQuery: map[string]string{
				"username": "dwizzybrain",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: got %s want %s", seenPath, tt.wantPath)
			}
			values, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := values.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %s want %s", key, got, want)
				}
			}
		})
	}
}

func TestIragRawDownloadPassthrough(t *testing.T) {
	t.Parallel()

	kanata := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/youtube/download" {
			t.Fatalf("unexpected upstream path: %s", got)
		}
		if got := r.URL.Query().Get("quality"); got != "720" {
			t.Fatalf("unexpected upstream quality: %s", got)
		}
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("png-bytes"))
	}))
	t.Cleanup(kanata.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, kanata.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/youtube/video?url=https://youtube.com/watch?v=abc", nil)
	NewRouter(service).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected raw content-type, got %q", got)
	}
	if got := rec.Body.String(); got != "png-bytes" {
		t.Fatalf("unexpected body: %q", got)
	}
}

func TestIragDownloadPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name     string
		path     string
		provider ProviderName
		wantPath string
		wantKey  string
		wantVal  string
	}

	tests := []tc{
		{
			name:     "aio",
			path:     "/v1/download/aio?url=https://www.tiktok.com/@u/video/1",
			provider: ProviderNexure,
			wantPath: "/api/download/aio",
			wantKey:  "url",
			wantVal:  "https://www.tiktok.com/@u/video/1",
		},
		{
			name:     "youtube video",
			path:     "/v1/download/youtube/video?url=https://youtube.com/watch?v=abc&quality=1080",
			provider: ProviderKanata,
			wantPath: "/youtube/download",
			wantKey:  "quality",
			wantVal:  "1080",
		},
		{
			name:     "youtube info",
			path:     "/v1/download/youtube/info?url=https://youtube.com/watch?v=abc",
			provider: ProviderKanata,
			wantPath: "/youtube/info",
			wantKey:  "url",
			wantVal:  "https://youtube.com/watch?v=abc",
		},
		{
			name:     "youtube audio",
			path:     "/v1/download/youtube/audio?url=https://youtube.com/watch?v=abc",
			provider: ProviderKanata,
			wantPath: "/youtube/download-audio",
			wantKey:  "url",
			wantVal:  "https://youtube.com/watch?v=abc",
		},
		{
			name:     "youtube playlist",
			path:     "/v1/download/youtube/playlist?url=https://youtube.com/playlist?list=abc",
			provider: ProviderYTDLP,
			wantPath: "/download/playlist",
			wantKey:  "url",
			wantVal:  "https://youtube.com/playlist?list=abc",
		},
		{
			name:     "youtube subtitle",
			path:     "/v1/download/youtube/subtitle?url=https://youtube.com/watch?v=abc&lang=id",
			provider: ProviderYTDLP,
			wantPath: "/download/ytsub",
			wantKey:  "lang",
			wantVal:  "id",
		},
		{
			name:     "youtube search",
			path:     "/v1/download/youtube/search?q=lofi",
			provider: ProviderRyzumi,
			wantPath: "/api/search/yt",
			wantKey:  "q",
			wantVal:  "lofi",
		},
		{
			name:     "tiktok hd",
			path:     "/v1/download/tiktok/hd?url=https://www.tiktok.com/@u/video/1",
			provider: ProviderYTDLP,
			wantPath: "/downloader/tiktokhd",
			wantKey:  "url",
			wantVal:  "https://www.tiktok.com/@u/video/1",
		},
		{
			name:     "douyin",
			path:     "/v1/download/douyin?url=https://www.douyin.com/video/1",
			provider: ProviderYTDLP,
			wantPath: "/downloader/tiktokhd",
			wantKey:  "url",
			wantVal:  "https://www.douyin.com/video/1",
		},
		{
			name:     "instagram story",
			path:     "/v1/download/instagram/story?url=https://instagram.com/stories/u/1",
			provider: ProviderNexure,
			wantPath: "/api/download/ig-story",
			wantKey:  "url",
			wantVal:  "https://instagram.com/stories/u/1",
		},
		{
			name:     "instagram post",
			path:     "/v1/download/instagram?url=https://instagram.com/p/abc",
			provider: ProviderNexure,
			wantPath: "/api/download/instagram",
			wantKey:  "url",
			wantVal:  "https://instagram.com/p/abc",
		},
		{
			name:     "spotify playlist",
			path:     "/v1/download/spotify/playlist?url=https://open.spotify.com/playlist/abc",
			provider: ProviderYTDLP,
			wantPath: "/spotify/download/playlist",
			wantKey:  "url",
			wantVal:  "https://open.spotify.com/playlist/abc",
		},
		{
			name:     "spotify track",
			path:     "/v1/download/spotify?url=https://open.spotify.com/track/abc",
			provider: ProviderNexure,
			wantPath: "/api/download/spotify",
			wantKey:  "url",
			wantVal:  "https://open.spotify.com/track/abc",
		},
		{
			name:     "facebook",
			path:     "/v1/download/facebook?url=https://facebook.com/watch?v=abc",
			provider: ProviderNexure,
			wantPath: "/api/download/facebook",
			wantKey:  "url",
			wantVal:  "https://facebook.com/watch?v=abc",
		},
		{
			name:     "threads",
			path:     "/v1/download/threads?url=https://threads.net/@u/post/abc",
			provider: ProviderNexure,
			wantPath: "/api/download/threads",
			wantKey:  "url",
			wantVal:  "https://threads.net/@u/post/abc",
		},
		{
			name:     "twitter",
			path:     "/v1/download/twitter?url=https://x.com/u/status/abc",
			provider: ProviderChocomilk,
			wantPath: "/v1/download/twitter",
			wantKey:  "url",
			wantVal:  "https://x.com/u/status/abc",
		},
		{
			name:     "pinterest",
			path:     "/v1/download/pinterest?url=https://pinterest.com/pin/abc",
			provider: ProviderKanata,
			wantPath: "/pinterest/fetch",
			wantKey:  "url",
			wantVal:  "https://pinterest.com/pin/abc",
		},
		{
			name:     "soundcloud",
			path:     "/v1/download/soundcloud?url=https://soundcloud.com/u/t/abc",
			provider: ProviderNexure,
			wantPath: "/api/download/soundcloud",
			wantKey:  "url",
			wantVal:  "https://soundcloud.com/u/t/abc",
		},
		{
			name:     "soundcloud playlist",
			path:     "/v1/download/soundcloud/playlist?url=https://soundcloud.com/u/sets/abc",
			provider: ProviderYTDLP,
			wantPath: "/downloader/soundcloud/playlist",
			wantKey:  "url",
			wantVal:  "https://soundcloud.com/u/sets/abc",
		},
		{
			name:     "gdrive",
			path:     "/v1/download/gdrive?url=https://drive.google.com/file/d/abc",
			provider: ProviderNexure,
			wantPath: "/api/download/gdrive",
			wantKey:  "url",
			wantVal:  "https://drive.google.com/file/d/abc",
		},
		{
			name:     "bilibili",
			path:     "/v1/download/bilibili?url=https://www.bilibili.tv/video/abc",
			provider: ProviderRyzumi,
			wantPath: "/api/downloader/bilibili",
			wantKey:  "url",
			wantVal:  "https://www.bilibili.tv/video/abc",
		},
		{
			name:     "bstation",
			path:     "/v1/download/bstation?url=https://www.bilibili.tv/video/abc",
			provider: ProviderRyzumi,
			wantPath: "/api/downloader/bilibili",
			wantKey:  "url",
			wantVal:  "https://www.bilibili.tv/video/abc",
		},
		{
			name:     "tidal",
			path:     "/v1/download/tidal?url=https://tidal.com/browse/track/abc",
			provider: ProviderChocomilk,
			wantPath: "/v1/download/tidal",
			wantKey:  "url",
			wantVal:  "https://tidal.com/browse/track/abc",
		},
		{
			name:     "deezer",
			path:     "/v1/download/deezer?url=https://deezer.com/track/abc",
			provider: ProviderChocomilk,
			wantPath: "/v1/download/deezer",
			wantKey:  "url",
			wantVal:  "https://deezer.com/track/abc",
		},
		{
			name:     "capcut",
			path:     "/v1/download/capcut?url=https://capcut.com/template/abc",
			provider: ProviderChocomilk,
			wantPath: "/v1/download/capcut",
			wantKey:  "url",
			wantVal:  "https://capcut.com/template/abc",
		},
		{
			name:     "scribd",
			path:     "/v1/download/scribd?url=https://scribd.com/doc/abc",
			provider: ProviderNexure,
			wantPath: "/api/download/scribd",
			wantKey:  "url",
			wantVal:  "https://scribd.com/doc/abc",
		},
		{
			name:     "mediafire",
			path:     "/v1/download/mediafire?url=https://www.mediafire.com/file/abc",
			provider: ProviderYTDLP,
			wantPath: "/downloader/mediafire",
			wantKey:  "url",
			wantVal:  "https://www.mediafire.com/file/abc",
		},
		{
			name:     "mega",
			path:     "/v1/download/mega?url=https://mega.nz/file/abc",
			provider: ProviderRyzumi,
			wantPath: "/api/downloader/mega",
			wantKey:  "url",
			wantVal:  "https://mega.nz/file/abc",
		},
		{
			name:     "terabox",
			path:     "/v1/download/terabox?url=https://terabox.com/s/abc",
			provider: ProviderRyzumi,
			wantPath: "/api/downloader/terabox",
			wantKey:  "url",
			wantVal:  "https://terabox.com/s/abc",
		},
		{
			name:     "pixeldrain",
			path:     "/v1/download/pixeldrain?url=https://pixeldrain.com/u/abc",
			provider: ProviderRyzumi,
			wantPath: "/api/downloader/pixeldrain",
			wantKey:  "url",
			wantVal:  "https://pixeldrain.com/u/abc",
		},
		{
			name:     "krakenfiles",
			path:     "/v1/download/krakenfiles?url=https://krakenfiles.com/view/abc",
			provider: ProviderRyzumi,
			wantPath: "/api/downloader/kfiles",
			wantKey:  "url",
			wantVal:  "https://krakenfiles.com/view/abc",
		},
		{
			name:     "danbooru",
			path:     "/v1/download/danbooru?url=https://danbooru.donmai.us/posts/abc",
			provider: ProviderRyzumi,
			wantPath: "/api/downloader/danbooru",
			wantKey:  "url",
			wantVal:  "https://danbooru.donmai.us/posts/abc",
		},
		{
			name:     "reddit",
			path:     "/v1/download/reddit?url=https://reddit.com/r/gif/abc",
			provider: ProviderNexure,
			wantPath: "/api/download/reddit",
			wantKey:  "url",
			wantVal:  "https://reddit.com/r/gif/abc",
		},
		{
			name:     "apple music",
			path:     "/v1/download/applemusic?url=https://music.apple.com/us/album/abc",
			provider: ProviderYTDLP,
			wantPath: "/download/applemusic",
			wantKey:  "url",
			wantVal:  "https://music.apple.com/us/album/abc",
		},
		{
			name:     "videy",
			path:     "/v1/download/videy?url=https://videy.co/v/abc",
			provider: ProviderYTDLP,
			wantPath: "/downloader/videy",
			wantKey:  "url",
			wantVal:  "https://videy.co/v/abc",
		},
		{
			name:     "sfile",
			path:     "/v1/download/sfile?url=https://sfile.mobi/abc",
			provider: ProviderYTDLP,
			wantPath: "/sfile",
			wantKey:  "url",
			wantVal:  "https://sfile.mobi/abc",
		},
		{
			name:     "shopee video",
			path:     "/v1/download/shopee/video?url=https://shopee.co.id/video/abc",
			provider: ProviderYTDLP,
			wantPath: "/shopee/video",
			wantKey:  "url",
			wantVal:  "https://shopee.co.id/video/abc",
		},
		{
			name:     "nhentai",
			path:     "/v1/download/nhentai?url=https://nhentai.net/g/abc",
			provider: ProviderYTDLP,
			wantPath: "/nhentai",
			wantKey:  "url",
			wantVal:  "https://nhentai.net/g/abc",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath string
			var seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: %s", seenPath)
			}
			values, _ := url.ParseQuery(seenQuery)
			if got := values.Get(tt.wantKey); got != tt.wantVal {
				t.Fatalf("unexpected %s: %s", tt.wantKey, got)
			}
		})
	}
}

func TestIragDownloadRouteUsesLongTimeout(t *testing.T) {
	t.Parallel()

	var seenPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(upstream.Close)

	service := NewService(Config{
		Timeout:      10 * time.Millisecond,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderYTDLP: {Name: ProviderYTDLP, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
		},
	}, nil, nil)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/download/youtube/playlist?url=https://youtube.com/playlist?list=abc", nil)
	NewRouter(service).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected playlist 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if seenPath != "/download/playlist" {
		t.Fatalf("unexpected path: %s", seenPath)
	}
}

func TestIragMediaAndUploadRootRoutes(t *testing.T) {
	t.Parallel()

	var mediaHits int32
	media := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&mediaHits, 1)
		if got := r.URL.Path; got != "/tv/now" {
			t.Fatalf("unexpected media upstream path: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"items":[{"title":"media"}]}}`))
	}))
	t.Cleanup(media.Close)

	var uploadHits int32
	upload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&uploadHits, 1)
		if got := r.URL.Path; got != "/api/upload" {
			t.Fatalf("unexpected upload upstream path: %s", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"data":{"uploaded":true}}`))
	}))
	t.Cleanup(upload.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: false,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderKanata: {Name: ProviderKanata, BaseURL: mustParseURL(t, media.URL), Enabled: true},
			ProviderNexure: {Name: ProviderNexure, BaseURL: mustParseURL(t, upload.URL), Enabled: true},
		},
	}, nil, nil)

	mediaRec := httptest.NewRecorder()
	NewRouter(service).ServeHTTP(mediaRec, httptest.NewRequest(http.MethodGet, "/v1/media/tv", nil))
	if mediaRec.Code != http.StatusOK {
		t.Fatalf("expected media 200, got %d: %s", mediaRec.Code, mediaRec.Body.String())
	}

	uploadBody := strings.NewReader(`field=value`)
	uploadReq := httptest.NewRequest(http.MethodPost, "/v1/upload", uploadBody)
	uploadReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	uploadRec := httptest.NewRecorder()
	NewRouter(service).ServeHTTP(uploadRec, uploadReq)
	if uploadRec.Code != http.StatusOK {
		t.Fatalf("expected upload 200, got %d: %s", uploadRec.Code, uploadRec.Body.String())
	}

	if atomic.LoadInt32(&mediaHits) != 1 {
		t.Fatalf("expected one media hit, got %d", mediaHits)
	}
	if atomic.LoadInt32(&uploadHits) != 1 {
		t.Fatalf("expected one upload hit, got %d", uploadHits)
	}
}

func TestIragUploadProviderPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name     string
		path     string
		provider ProviderName
		wantPath string
	}

	tests := []tc{
		{
			name:     "upload nexure",
			path:     "/v1/upload",
			provider: ProviderNexure,
			wantPath: "/api/upload",
		},
		{
			name:     "upload kanata",
			path:     "/v1/upload/kanata",
			provider: ProviderKanata,
			wantPath: "/upload",
		},
		{
			name:     "upload ryzumi",
			path:     "/v1/upload/ryzumi",
			provider: ProviderRyzumi,
			wantPath: "/api/uploader/ryzumicdn",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			body := strings.NewReader(`field=value`)
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			rec := httptest.NewRecorder()
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: got %s want %s", seenPath, tt.wantPath)
			}
		})
	}
}

func TestIragGrowAGardenPathTranslation(t *testing.T) {
	t.Parallel()

	type tc struct {
		name      string
		path      string
		provider  ProviderName
		wantPath  string
		wantQuery map[string]string
	}

	tests := []tc{
		{
			name:     "growagarden stock nexure",
			path:     "/v1/game/growagarden/stock",
			provider: ProviderNexure,
			wantPath: "/api/info/growagarden",
		},
		{
			name:     "growagarden crops ryzumi",
			path:     "/v1/game/growagarden/crops",
			provider: ProviderRyzumi,
			wantPath: "/api/tool/growagarden",
			wantQuery: map[string]string{
				"category": "crops",
				"type":     "crops",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var seenPath, seenQuery string
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				seenPath = r.URL.Path
				seenQuery = r.URL.RawQuery
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"ok":true}`))
			}))
			t.Cleanup(upstream.Close)

			service := NewService(Config{
				Timeout:      2 * time.Second,
				CacheEnabled: false,
				Upstreams: map[ProviderName]UpstreamConfig{
					tt.provider: {Name: tt.provider, BaseURL: mustParseURL(t, upstream.URL), Enabled: true},
				},
			}, nil, nil)

			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			NewRouter(service).ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
			}
			if seenPath != tt.wantPath {
				t.Fatalf("unexpected path: got %s want %s", seenPath, tt.wantPath)
			}
			values, _ := url.ParseQuery(seenQuery)
			for key, want := range tt.wantQuery {
				if got := values.Get(key); got != want {
					t.Fatalf("unexpected query %s: got %q want %q", key, got, want)
				}
			}
		})
	}
}

func TestIragCacheHit(t *testing.T) {
	t.Parallel()

	miniredis := miniredis.RunT(t)
	cacheClient := redis.NewClient(&redis.Options{Addr: miniredis.Addr()})
	t.Cleanup(func() { _ = cacheClient.Close() })

	var hits int32
	search := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&hits, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"items":[{"title":"one"}]}}`))
	}))
	t.Cleanup(search.Close)

	service := NewService(Config{
		Timeout:      2 * time.Second,
		CacheEnabled: true,
		Upstreams: map[ProviderName]UpstreamConfig{
			ProviderRyzumi: {Name: ProviderRyzumi, BaseURL: mustParseURL(t, search.URL), Enabled: true},
		},
	}, NewRedisCache(cacheClient), nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/search/google?q=hello", nil)
	rec1 := httptest.NewRecorder()
	NewRouter(service).ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("first request failed: %d %s", rec1.Code, rec1.Body.String())
	}

	rec2 := httptest.NewRecorder()
	NewRouter(service).ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/v1/search/google?q=hello", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("second request failed: %d %s", rec2.Code, rec2.Body.String())
	}
	if got := rec2.Header().Get("X-IRAG-Cache"); got != "HIT" {
		t.Fatalf("expected cache hit, got %q", got)
	}
	if atomic.LoadInt32(&hits) != 1 {
		t.Fatalf("expected one upstream hit, got %d", hits)
	}
}

func mustParseURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	return parsed
}
