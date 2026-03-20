package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadOptionalPrefersEnvOverFile(t *testing.T) {
	t.Setenv("TEST_SECRET", "env-value")
	file := writeTempSecretFile(t, "file-value")
	t.Setenv("TEST_SECRET_FILE", file)

	got, err := ReadOptional("TEST_SECRET")
	if err != nil {
		t.Fatalf("ReadOptional() returned error: %v", err)
	}
	if got != "env-value" {
		t.Fatalf("expected env-value, got %q", got)
	}
}

func TestReadOptionalUsesFileFallback(t *testing.T) {
	file := writeTempSecretFile(t, "file-value\n")
	t.Setenv("TEST_SECRET", "")
	t.Setenv("TEST_SECRET_FILE", file)

	got, err := ReadOptional("TEST_SECRET")
	if err != nil {
		t.Fatalf("ReadOptional() returned error: %v", err)
	}
	if got != "file-value" {
		t.Fatalf("expected file-value, got %q", got)
	}
}

func TestReadRequiredErrorsWhenMissing(t *testing.T) {
	t.Setenv("TEST_SECRET", "")
	t.Setenv("TEST_SECRET_FILE", "")

	if _, err := ReadRequired("TEST_SECRET"); err == nil {
		t.Fatal("expected error when secret is missing")
	}
}

func writeTempSecretFile(t *testing.T, value string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.txt")
	if err := os.WriteFile(path, []byte(value), 0o600); err != nil {
		t.Fatalf("write temp secret file: %v", err)
	}
	return path
}
