package coverage

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dwizzyBRAIN/engine/storage"
)

func TestPostgresCoverageIntegration(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	pool, err := storage.NewPostgresPool(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgresPool() returned error: %v", err)
	}
	defer pool.Close()

	store := NewPostgresStore(pool)
	suffix := time.Now().UnixNano()
	coinID := fmt.Sprintf("coverage-test-%d", suffix)
	baseAsset := fmt.Sprintf("COV%d", suffix)
	exchangeSymbol := fmt.Sprintf("%sUSDT", baseAsset)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from coin_coverage where coin_id = $1", coinID)
		_, _ = pool.Exec(context.Background(), "delete from coin_exchange_mappings where coin_id = $1", coinID)
		_, _ = pool.Exec(context.Background(), "delete from coins where id = $1", coinID)
	})

	if _, err := pool.Exec(context.Background(), `
INSERT INTO coins (id, symbol, name, rank)
VALUES ($1, 'ct', 'Coverage Test', 50)`, coinID); err != nil {
		t.Fatalf("insert coin returned error: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
INSERT INTO coin_exchange_mappings (coin_id, exchange, exchange_symbol, base_asset, quote_asset, status, verified_at, is_primary)
VALUES ($1, 'binance', $2, $3, 'USDT', 'active', NOW(), TRUE)`, coinID, exchangeSymbol, baseAsset); err != nil {
		t.Fatalf("insert mapping returned error: %v", err)
	}

	detector := NewGapDetector(store)
	if _, err := detector.DetectAll(context.Background()); err != nil {
		t.Fatalf("DetectAll() returned error: %v", err)
	}

	coverage, err := store.GetCoverage(context.Background(), coinID)
	if err != nil {
		t.Fatalf("GetCoverage() returned error: %v", err)
	}
	if coverage.CoinID != coinID {
		t.Fatalf("expected coin %s, got %s", coinID, coverage.CoinID)
	}
	if coverage.Tier != "B" {
		t.Fatalf("expected tier B for rank-50 single exchange coin, got %s", coverage.Tier)
	}
	if !coverage.OnBinance {
		t.Fatal("expected on_binance to be true")
	}
}
