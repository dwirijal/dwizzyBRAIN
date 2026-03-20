package storage

import (
	"context"
	"os"
	"testing"
)

func TestNewPostgresPoolFromEnv(t *testing.T) {
	t.Setenv("POSTGRES_URL", "")
	if _, err := NewPostgresPoolFromEnv(context.Background()); err == nil {
		t.Fatal("expected an error when POSTGRES_URL is missing")
	}
}

func TestNewPostgresPoolIntegration(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	pool, err := NewPostgresPool(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgresPool() returned error: %v", err)
	}
	defer pool.Close()
}
