package storage

import (
	"context"
	"os"
	"testing"
)

func TestNewPostgresPoolFromEnv(t *testing.T) {
	t.Setenv("POSTGRES_URL", "")
	t.Setenv("NEON_DATABASE_URL", "")
	if _, err := NewPostgresPoolFromEnv(context.Background()); err == nil {
		t.Fatal("expected an error when POSTGRES_URL is missing")
	} else if err.Error() != "POSTGRES_URL is required (NEON_DATABASE_URL supported as compatibility fallback)" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewPostgresPoolFromEnvUsesNeonFallback(t *testing.T) {
	t.Setenv("POSTGRES_URL", "")
	t.Setenv("NEON_DATABASE_URL", "postgres://localhost:5432/example")

	if _, err := NewPostgresPoolFromEnv(context.Background()); err == nil {
		t.Fatal("expected connection failure after accepting NEON_DATABASE_URL fallback")
	} else if err.Error() == "POSTGRES_URL is required" || err.Error() == "POSTGRES_URL or NEON_DATABASE_URL is required" {
		t.Fatalf("expected function to accept NEON_DATABASE_URL fallback, got %v", err)
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
