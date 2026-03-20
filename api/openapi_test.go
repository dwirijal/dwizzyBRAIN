package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAPISpecContainsMarketPaths(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal(openAPISpec, &doc); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}

	if doc["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %v", doc["openapi"])
	}

	paths, ok := doc["paths"].(map[string]any)
	if !ok {
		t.Fatalf("expected paths map")
	}

	for _, path := range []string{
		"/v1/health",
		"/v1/auth/discord/start",
		"/v1/auth/discord/callback",
		"/v1/auth/web3/nonce",
		"/v1/auth/web3/verify",
		"/v1/auth/refresh",
		"/v1/auth/logout",
		"/v1/auth/me",
		"/v1/market",
		"/v1/market/{id}",
		"/v1/market/{id}/ohlcv",
		"/v1/market/{id}/tickers",
		"/v1/market/{id}/orderbook",
		"/v1/market/{id}/arbitrage",
		"/v1/defi",
		"/v1/defi/protocols",
		"/v1/defi/protocols/{slug}",
		"/v1/defi/chains",
		"/v1/defi/dexes",
		"/v1/news",
		"/v1/news/{value}",
		"/v1/news/coin/{coin_id}",
		"/v1/news/trending",
		"/v1/quant/pattern",
		"/v1/quant/signals",
		"/v1/quant/signals/latest",
		"/v1/quant/signals/summary",
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
