package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMigratorApplyDirIntegration(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	pool, err := NewPostgresPool(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgresPool() returned error: %v", err)
	}
	defer pool.Close()

	tableName := fmt.Sprintf("migration_test_%d", time.Now().UnixNano())
	baseVersion := time.Now().UnixNano() % 1_000_000_000_000
	versionOne := baseVersion + 1
	versionTwo := baseVersion + 2
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "drop table if exists "+tableName)
		_, _ = pool.Exec(context.Background(), fmt.Sprintf("delete from schema_migrations where version in (%d, %d)", versionOne, versionTwo))
	})

	dir := t.TempDir()
	migrationOne := filepath.Join(dir, fmt.Sprintf("%d_create_table.sql", versionOne))
	migrationTwo := filepath.Join(dir, fmt.Sprintf("%d_insert_row.sql", versionTwo))

	if err := os.WriteFile(migrationOne, []byte("create table if not exists "+tableName+" (id integer primary key);"), 0o644); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}
	if err := os.WriteFile(migrationTwo, []byte("insert into "+tableName+" (id) values (1) on conflict do nothing;"), 0o644); err != nil {
		t.Fatalf("WriteFile() returned error: %v", err)
	}

	migrator := NewMigrator(pool)
	applied, err := migrator.ApplyDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("ApplyDir() returned error: %v", err)
	}

	if len(applied) != 2 {
		t.Fatalf("expected 2 applied migrations, got %d (%s)", len(applied), strings.Join(applied, ","))
	}

	var count int
	if err := pool.QueryRow(context.Background(), "select count(*) from "+tableName).Scan(&count); err != nil {
		t.Fatalf("QueryRow() returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 inserted row, got %d", count)
	}

	reapplied, err := migrator.ApplyDir(context.Background(), dir)
	if err != nil {
		t.Fatalf("ApplyDir() second run returned error: %v", err)
	}
	if len(reapplied) != 2 {
		t.Fatalf("expected migration list length to remain 2 on second run, got %d", len(reapplied))
	}
}
