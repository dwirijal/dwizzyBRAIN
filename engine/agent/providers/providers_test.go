package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIRAGProviderAsk(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"text":"hello from irag"}}`))
	}))
	defer server.Close()

	provider, err := NewIRAGProvider("irag-qwen", server.URL, "/v1/ai/text/qwen")
	if err != nil {
		t.Fatalf("NewIRAGProvider() returned error: %v", err)
	}

	got, err := provider.Ask(context.Background(), "summarize this")
	if err != nil {
		t.Fatalf("Ask() returned error: %v", err)
	}
	if got != "hello from irag" {
		t.Fatalf("expected parsed text, got %q", got)
	}
}

func TestProviderRetryableStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	provider, err := NewIRAGProvider("irag-groq", server.URL, "/v1/ai/text/groq")
	if err != nil {
		t.Fatalf("NewIRAGProvider() returned error: %v", err)
	}

	_, err = provider.Ask(context.Background(), "test")
	if err == nil {
		t.Fatal("expected retryable error")
	}
}
