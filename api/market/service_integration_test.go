package marketapi

import (
	"context"
	"os"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/ohlcv"
	engticker "dwizzyBRAIN/engine/market/ticker"
	"dwizzyBRAIN/engine/storage"

	redis "github.com/redis/go-redis/v9"
)

func TestServiceLiveMarketRead(t *testing.T) {
	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("POSTGRES_URL is required for integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := storage.NewPostgresPoolFromEnv(ctx)
	if err != nil {
		t.Fatalf("new postgres pool: %v", err)
	}
	defer pool.Close()

	var cache redis.Cmdable
	if os.Getenv("VALKEY_URL") != "" {
		client, err := storage.NewValkeyClientFromEnv(ctx)
		if err != nil {
			t.Fatalf("new valkey client: %v", err)
		}
		defer client.Close()
		cache = client
	}

	var ohlcvStore *ohlcv.TimescaleStore
	var spreadStore *engticker.SpreadStore
	if os.Getenv("TIMESCALE_URL") != "" {
		tsPool, err := storage.NewTimescalePoolFromEnv(ctx)
		if err != nil {
			t.Fatalf("new timescale pool: %v", err)
		}
		defer tsPool.Close()
		ohlcvStore = ohlcv.NewTimescaleStore(tsPool)
		spreadStore = engticker.NewSpreadStore(tsPool)
	}

	service := NewService(pool, ohlcvStore, spreadStore, cache)

	items, total, err := service.List(ctx, 5, 0)
	if err != nil {
		t.Fatalf("list market snapshots: %v", err)
	}
	if total <= 0 {
		t.Fatalf("expected positive total, got %d", total)
	}
	if len(items) == 0 {
		t.Fatalf("expected at least one market snapshot")
	}

	detail, err := service.Detail(ctx, "bitcoin")
	if err != nil {
		t.Fatalf("detail market snapshot: %v", err)
	}
	if detail.CoinID != "bitcoin" {
		t.Fatalf("expected bitcoin detail, got %s", detail.CoinID)
	}
	if len(detail.Mappings) == 0 {
		t.Fatalf("expected bitcoin mappings to be present")
	}
	if detail.Availability.Tier == "" {
		t.Fatalf("expected bitcoin coverage tier")
	}

	if _, err := service.OHLCV(ctx, "bitcoin", "binance", "1m", 5); err != nil {
		t.Fatalf("ohlcv market read: %v", err)
	}
	if _, err := service.Tickers(ctx, "bitcoin"); err != nil {
		t.Fatalf("tickers market read: %v", err)
	}
	if _, err := service.Arbitrage(ctx, "bitcoin", 5); err != nil {
		t.Fatalf("arbitrage market read: %v", err)
	}
}
