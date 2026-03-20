package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDocsRoute(t *testing.T) {
	mux := NewRouter(nil, nil, nil, nil, nil)

	for _, path := range []string{"/docs", "/docs/"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()

		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", path, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
			t.Fatalf("expected html content type for %s, got %q", path, ct)
		}
		if got := rec.Header().Get("X-OpenAPI-Version"); got != openAPIContractVersion {
			t.Fatalf("expected openapi version header for %s, got %q", path, got)
		}
		if got := rec.Header().Get("X-OpenAPI-SHA256"); got != openAPISpecSHA256 {
			t.Fatalf("expected openapi sha header for %s, got %q", path, got)
		}
		body := rec.Body.String()
		if !strings.Contains(body, "SwaggerUIBundle") {
			t.Fatalf("expected swagger ui bundle in docs response for %s", path)
		}
		if !strings.Contains(body, "/openapi.json") {
			t.Fatalf("expected openapi.json reference in docs response for %s", path)
		}
		if !strings.Contains(body, openAPIContractVersion) {
			t.Fatalf("expected contract version in docs response for %s", path)
		}
	}
}
