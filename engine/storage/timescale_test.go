package storage

import (
	"context"
	"os"
	"testing"
)

func TestNewTimescalePoolFromEnv(t *testing.T) {
	t.Setenv("TIMESCALE_URL", "")
	if _, err := NewTimescalePoolFromEnv(context.Background()); err == nil {
		t.Fatal("expected an error when TIMESCALE_URL is missing")
	}
}

func TestEnsureTimescaleExtensionIntegration(t *testing.T) {
	url := os.Getenv("TIMESCALE_URL")
	if url == "" {
		t.Skip("TIMESCALE_URL is not set")
	}

	pool, err := NewTimescalePool(context.Background(), url)
	if err != nil {
		t.Fatalf("NewTimescalePool() returned error: %v", err)
	}
	defer pool.Close()

	if err := EnsureTimescaleExtension(context.Background(), pool); err != nil {
		t.Fatalf("EnsureTimescaleExtension() returned error: %v", err)
	}
}
