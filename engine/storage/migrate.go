package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrator struct {
	pool *pgxpool.Pool
}

var migrationVersionPattern = regexp.MustCompile(`^(\d+)`)

func NewMigrator(pool *pgxpool.Pool) *Migrator {
	return &Migrator{pool: pool}
}

func (m *Migrator) ApplyDir(ctx context.Context, dir string) ([]string, error) {
	if m.pool == nil {
		return nil, fmt.Errorf("migration pool is required")
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	migrationFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		migrationFiles = append(migrationFiles, entry.Name())
	}
	sort.Strings(migrationFiles)

	if err := m.ensureVersionTable(ctx); err != nil {
		return nil, err
	}

	applied := make([]string, 0, len(migrationFiles))
	for _, name := range migrationFiles {
		if err := m.applyFile(ctx, dir, name); err != nil {
			return applied, err
		}

		applied = append(applied, name)
	}

	return applied, nil
}

func (m *Migrator) ensureVersionTable(ctx context.Context) error {
	const createTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version TEXT PRIMARY KEY,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`

	if _, err := m.pool.Exec(ctx, createTable); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	if _, err := m.pool.Exec(ctx, `
ALTER TABLE schema_migrations
ADD COLUMN IF NOT EXISTS applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`); err != nil {
		return fmt.Errorf("ensure schema_migrations.applied_at column: %w", err)
	}
	if _, err := m.pool.Exec(ctx, `
ALTER TABLE schema_migrations
ADD COLUMN IF NOT EXISTS dirty BOOLEAN NOT NULL DEFAULT FALSE`); err != nil {
		return fmt.Errorf("ensure schema_migrations.dirty column: %w", err)
	}

	return nil
}

func (m *Migrator) applyFile(ctx context.Context, dir, name string) error {
	version, err := migrationVersion(name)
	if err != nil {
		return err
	}

	var appliedAt time.Time
	err = m.pool.QueryRow(ctx, "select applied_at from schema_migrations where version::text = $1", version).Scan(&appliedAt)
	if err == nil {
		return nil
	}
	if err != nil && err != pgx.ErrNoRows {
		return fmt.Errorf("check migration %s: %w", version, err)
	}

	path := filepath.Join(dir, name)
	sqlBytes, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", path, err)
	}

	tx, err := m.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration tx %s: %w", version, err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
		return fmt.Errorf("execute migration %s: %w", version, err)
	}
	if _, err := tx.Exec(ctx, "insert into schema_migrations (version, dirty) values ($1, FALSE)", version); err != nil {
		return fmt.Errorf("record migration %s: %w", version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", version, err)
	}

	return nil
}

func migrationVersion(name string) (string, error) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	matches := migrationVersionPattern.FindStringSubmatch(base)
	if len(matches) != 2 {
		return "", fmt.Errorf("migration %s must start with a numeric version", name)
	}

	version := strings.TrimLeft(matches[1], "0")
	if version == "" {
		version = "0"
	}

	return version, nil
}
