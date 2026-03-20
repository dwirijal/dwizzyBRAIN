package agent

import (
	"context"
	"errors"
	"testing"

	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"
)

type stubProvider struct {
	name     string
	response string
	err      error
	calls    int
}

func (s *stubProvider) Name() string { return s.name }
func (s *stubProvider) Ask(ctx context.Context, prompt string) (string, error) {
	s.calls++
	return s.response, s.err
}

func TestAgentRouterFallsBackOnRetryableErrors(t *testing.T) {
	first := &stubProvider{
		name: "irag-groq",
		err:  NewRetryableProviderError("irag-groq", 429, errors.New("rate limited")),
	}
	second := &stubProvider{
		name:     "groq",
		response: "fallback response",
	}

	router := NewAgentRouter([]LLMProvider{first, second}, nil)
	got, err := router.Ask(context.Background(), "article-1", "summarize")
	if err != nil {
		t.Fatalf("Ask() returned error: %v", err)
	}
	if got != "fallback response" {
		t.Fatalf("expected fallback response, got %q", got)
	}
	if first.calls != 1 || second.calls != 1 {
		t.Fatalf("expected both providers to be called once, got %d and %d", first.calls, second.calls)
	}
}

func TestAgentRouterStopsOnNonRetryableError(t *testing.T) {
	first := &stubProvider{
		name: "irag-qwen",
		err:  errors.New("bad request"),
	}
	second := &stubProvider{
		name:     "groq",
		response: "should not be reached",
	}

	router := NewAgentRouter([]LLMProvider{first, second}, nil)
	_, err := router.Ask(context.Background(), "article-2", "summarize")
	if err == nil {
		t.Fatal("expected error")
	}
	if second.calls != 0 {
		t.Fatalf("expected second provider not to be called, got %d", second.calls)
	}
}

func TestAgentRouterDuplicateInflightLock(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	if err := client.Set(context.Background(), "inflight:ai:article-3", "1", 0).Err(); err != nil {
		t.Fatalf("Set() returned error: %v", err)
	}

	router := NewAgentRouter([]LLMProvider{&stubProvider{name: "groq", response: "ok"}}, client)
	_, err := router.Ask(context.Background(), "article-3", "summarize")
	if !errors.Is(err, ErrDuplicateRequest) {
		t.Fatalf("expected ErrDuplicateRequest, got %v", err)
	}
}
