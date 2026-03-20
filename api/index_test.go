package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIndexRoute(t *testing.T) {
	mux := NewRouter(nil, nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Fatalf("expected json content type, got %q", ct)
	}
	if got := rec.Header().Get("X-OpenAPI-Version"); got != openAPIContractVersion {
		t.Fatalf("expected openapi version header, got %q", got)
	}
	if got := rec.Header().Get("X-OpenAPI-SHA256"); got != openAPISpecSHA256 {
		t.Fatalf("expected openapi sha header, got %q", got)
	}
	body := rec.Body.String()
	for _, want := range []string{`"docs": "/docs"`, `"openapi": "/openapi.json"`, `"health": "/v1/health"`, `"auth": "/v1/auth/discord/start"`} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected %s in body, got %s", want, body)
		}
	}
}
