package storage

import (
	"context"
	"os"
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestNewValkeyClientFromEnv(t *testing.T) {
	t.Setenv("VALKEY_URL", "")
	if _, err := NewValkeyClientFromEnv(context.Background()); err == nil {
		t.Fatal("expected an error when VALKEY_URL is missing")
	}

	server := miniredis.RunT(t)
	t.Setenv("VALKEY_URL", "redis://"+server.Addr())

	client, err := NewValkeyClientFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewValkeyClientFromEnv() returned error: %v", err)
	}
	defer client.Close()
}

func TestNewValkeyClientInvalidURL(t *testing.T) {
	client, err := NewValkeyClient(context.Background(), "://bad-url")
	if err == nil {
		defer client.Close()
		t.Fatal("expected invalid URL error")
	}
}

func TestNewValkeyClientUsesEnvironmentValue(t *testing.T) {
	server := miniredis.RunT(t)
	if err := os.Setenv("VALKEY_URL", "redis://"+server.Addr()); err != nil {
		t.Fatalf("Setenv() returned error: %v", err)
	}
	defer os.Unsetenv("VALKEY_URL")

	client, err := NewValkeyClientFromEnv(context.Background())
	if err != nil {
		t.Fatalf("NewValkeyClientFromEnv() returned error: %v", err)
	}
	defer client.Close()
}
