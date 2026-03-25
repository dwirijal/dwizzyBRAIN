package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAPISpecContainsGatewayPaths(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(openAPISpec, &doc); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}

	if doc["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %v", doc["openapi"])
	}
	info, ok := doc["info"].(map[string]any)
	if !ok {
		t.Fatalf("expected info map")
	}
	if info["version"] != "2.2.0" {
		t.Fatalf("expected api contract version 2.2.0, got %v", info["version"])
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths map")
	}

	for _, path := range []string{
		"/v1/health",
		"/v1/content/manhwa",
		"/v1/content/manhwa/{slug}",
		"/v1/content/manhwa/{slug}/units",
		"/v1/content/units/{slug}",
		"/v1/download/aio",
		"/v1/download/youtube/info",
		"/v1/download/youtube/video",
		"/v1/download/youtube/audio",
		"/v1/download/youtube/playlist",
		"/v1/download/youtube/subtitle",
		"/v1/download/instagram",
		"/v1/download/tiktok",
		"/v1/download/spotify",
		"/v1/auth/discord/start",
		"/v1/auth/discord/callback",
		"/v1/auth/web3/nonce",
		"/v1/auth/web3/verify",
		"/v1/auth/refresh",
		"/v1/auth/logout",
		"/v1/auth/me",
		"/v1/defi",
		"/v1/defi/protocols",
		"/v1/defi/protocols/{slug}",
		"/v1/defi/chains",
		"/v1/defi/dexes",
		"/v1/news",
		"/v1/news/{value}",
		"/v1/news/coin/{coin_id}",
		"/v1/news/trending",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("expected path %s in openapi spec", path)
		}
	}
}

func TestOpenAPIRoute(t *testing.T) {
	mux := NewRouter(nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected application/json content type, got %q", ct)
	}
	if got := rec.Header().Get("X-OpenAPI-Version"); got != openAPIContractVersion {
		t.Fatalf("expected openapi version header, got %q", got)
	}
	if got := rec.Header().Get("X-OpenAPI-SHA256"); got != openAPISpecSHA256 {
		t.Fatalf("expected openapi sha header, got %q", got)
	}
	if rec.Body.Len() == 0 {
		t.Fatalf("expected openapi body")
	}
}
