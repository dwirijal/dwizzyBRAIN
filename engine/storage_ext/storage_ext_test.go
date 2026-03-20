package storageext

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

type fakeCommandRunner struct {
	name string
	args []string
	out  string
}

func (f *fakeCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	f.name = name
	f.args = append([]string{}, args...)
	return nil
}

func (f *fakeCommandRunner) Output(ctx context.Context, name string, args ...string) (string, error) {
	f.name = name
	f.args = append([]string{}, args...)
	return f.out, nil
}

func TestRcloneBridgeCommands(t *testing.T) {
	runner := RcloneRunner{Bin: "rclone", Exec: &fakeCommandRunner{}}
	gdrive := NewGDriveBackup(runner, "gdrive", "backups")
	if err := gdrive.Backup(context.Background(), "/tmp/dump", "20260319"); err != nil {
		t.Fatalf("gdrive backup returned error: %v", err)
	}

	fake := runner.Exec.(*fakeCommandRunner)
	if fake.name != "rclone" {
		t.Fatalf("expected rclone binary, got %q", fake.name)
	}
	if len(fake.args) < 4 || fake.args[0] != "copy" || fake.args[1] != "/tmp/dump" || fake.args[2] != "gdrive:backups/20260319" {
		t.Fatalf("unexpected gdrive args: %#v", fake.args)
	}

	r2Runner := RcloneRunner{Bin: "rclone", Exec: &fakeCommandRunner{}}
	r2 := NewR2Sync(r2Runner, "r2", "logos")
	if err := r2.SyncDirectory(context.Background(), "/tmp/logos"); err != nil {
		t.Fatalf("r2 sync returned error: %v", err)
	}
	fakeR2 := r2Runner.Exec.(*fakeCommandRunner)
	if len(fakeR2.args) < 4 || fakeR2.args[0] != "sync" || fakeR2.args[1] != "/tmp/logos" || fakeR2.args[2] != "r2:logos" {
		t.Fatalf("unexpected r2 args: %#v", fakeR2.args)
	}
}

func TestTelegramBridgeUploadAndCache(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New() returned error: %v", err)
	}
	defer pool.Close()

	if err := ensureTelegramCacheTable(ctx, pool); err != nil {
		t.Fatalf("ensureTelegramCacheTable() returned error: %v", err)
	}

	mr := miniredis.RunT(t)
	cache := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer cache.Close()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/sendDocument") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() returned error: %v", err)
		}
		if got := r.FormValue("chat_id"); got != "-100123" {
			t.Fatalf("unexpected chat_id %q", got)
		}
		file, header, err := r.FormFile("document")
		if err != nil {
			t.Fatalf("FormFile() returned error: %v", err)
		}
		defer file.Close()
		body, _ := io.ReadAll(file)
		if string(body) != "hello storage" {
			t.Fatalf("unexpected upload body: %q", string(body))
		}
		if header.Filename != "report.pdf" {
			t.Fatalf("unexpected filename %q", header.Filename)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"result": map[string]any{
				"message_id": 99,
				"document": map[string]any{
					"file_id":        "FILE123",
					"file_unique_id": "UNIQ123",
					"file_name":      "report.pdf",
					"file_size":      1234,
					"mime_type":      "application/pdf",
				},
			},
		})
	}))
	defer server.Close()

	bridge := &TelegramBridge{
		db:        pool,
		cache:     cache,
		http:      server.Client(),
		botToken:  "token",
		channelID: "-100123",
		baseURL:   server.URL,
	}

	rec, err := bridge.UploadDocument(ctx, TelegramUploadRequest{
		FileKey:     "chart:bitcoin:2026-03-19",
		FileName:    "report.pdf",
		ContentType: "application/pdf",
		FileType:    "report",
		CoinID:      "bitcoin",
		Timeframe:   "1d",
		DateContext: "2026-03-19",
		Content:     strings.NewReader("hello storage"),
	})
	if err != nil {
		t.Fatalf("UploadDocument() returned error: %v", err)
	}
	if rec.FileID == "" || rec.FileKey == "" {
		t.Fatalf("expected upload record, got %+v", rec)
	}

	got, err := bridge.Get(ctx, rec.FileKey)
	if err != nil {
		t.Fatalf("Get() returned error: %v", err)
	}
	if got.FileID != rec.FileID {
		t.Fatalf("expected cached file id %q, got %q", rec.FileID, got.FileID)
	}
	if !mr.Exists(bridge.CacheKey(rec.FileKey)) {
		t.Fatal("expected telegram file cache mirror in valkey")
	}
}

func ensureTelegramCacheTable(ctx context.Context, pool *pgxpool.Pool) error {
	statements := []string{
		`DO $$
BEGIN
    CREATE TYPE telegram_file_type AS ENUM ('chart', 'csv_export', 'backup_db', 'backup_valkey', 'report', 'other');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;`,
		`CREATE TABLE IF NOT EXISTS telegram_file_cache (
			id BIGSERIAL PRIMARY KEY,
			file_key TEXT NOT NULL UNIQUE,
			file_id TEXT NOT NULL,
			file_unique_id TEXT,
			message_id BIGINT,
			channel_id TEXT,
			file_type telegram_file_type NOT NULL DEFAULT 'other',
			file_name TEXT,
			file_size_bytes BIGINT,
			mime_type TEXT,
			coin_id TEXT,
			timeframe TEXT,
			date_context DATE,
			uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_accessed_at TIMESTAMPTZ,
			access_count INTEGER NOT NULL DEFAULT 0
		)`,
	}
	for _, stmt := range statements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
