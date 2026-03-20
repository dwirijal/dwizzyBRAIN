package mapping

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dwizzyBRAIN/engine/storage"
)

func TestPostgresStoreIntegration(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	pool, err := storage.NewPostgresPool(context.Background(), url)
	if err != nil {
		t.Fatalf("NewPostgresPool() returned error: %v", err)
	}
	defer pool.Close()

	suffix := time.Now().UnixNano()
	coinID := fmt.Sprintf("resolver-test-%d", suffix)
	exchangeSymbol := fmt.Sprintf("RT%dUSDT", suffix)
	exchange := fmt.Sprintf("integration-%d", suffix)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from unknown_symbols where exchange = 'binance' and raw_symbol in ($1, $2)", "UNKN"+exchangeSymbol, exchangeSymbol)
		_, _ = pool.Exec(context.Background(), "delete from coin_exchange_mappings where coin_id = $1 and exchange = $2", coinID, exchange)
		_, _ = pool.Exec(context.Background(), "delete from coins where id = $1", coinID)
	})

	if _, err := pool.Exec(context.Background(), `
INSERT INTO coins (id, symbol, name, rank)
VALUES ($1, $2, $3, 999999)
ON CONFLICT (id) DO UPDATE SET symbol = EXCLUDED.symbol, name = EXCLUDED.name`, coinID, "rt", "Resolver Test"); err != nil {
		t.Fatalf("insert coin returned error: %v", err)
	}

	store := NewPostgresStore(pool)

	upsert := Mapping{
		CoinID:         coinID,
		Exchange:       exchange,
		ExchangeSymbol: exchangeSymbol,
		BaseAsset:      "RT" + fmt.Sprint(suffix),
		QuoteAsset:     "USDT",
		IsPrimary:      true,
	}
	if err := store.UpsertMapping(context.Background(), upsert, "active"); err != nil {
		t.Fatalf("UpsertMapping() returned error: %v", err)
	}
	upsert.BaseAsset = "RTX" + fmt.Sprint(suffix)
	upsert.IsPrimary = false
	if err := store.UpsertMapping(context.Background(), upsert, "active"); err != nil {
		t.Fatalf("UpsertMapping() second call returned error: %v", err)
	}

	got, err := store.GetPrimaryMapping(context.Background(), coinID, exchange)
	if err != nil {
		t.Fatalf("GetPrimaryMapping() returned error: %v", err)
	}
	if got.ExchangeSymbol != exchangeSymbol {
		t.Fatalf("expected exchange symbol %s, got %s", exchangeSymbol, got.ExchangeSymbol)
	}
	if got.BaseAsset != "RTX"+fmt.Sprint(suffix) {
		t.Fatalf("expected updated base asset, got %s", got.BaseAsset)
	}

	reverse, err := store.GetMappingBySymbol(context.Background(), exchange, exchangeSymbol)
	if err != nil {
		t.Fatalf("GetMappingBySymbol() returned error: %v", err)
	}
	if reverse.CoinID != coinID {
		t.Fatalf("expected coinID %s, got %s", coinID, reverse.CoinID)
	}

	rows, err := store.ListMappingsByExchange(context.Background(), exchange)
	if err != nil {
		t.Fatalf("ListMappingsByExchange() returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected one mapping row, got %d", len(rows))
	}

	if err := store.RecordUnknownSymbol(context.Background(), exchange, "UNKN"+exchangeSymbol, "UNKN"); err != nil {
		t.Fatalf("RecordUnknownSymbol() returned error: %v", err)
	}

	var count int
	if err := pool.QueryRow(context.Background(), `
SELECT count(*)
FROM unknown_symbols
	WHERE exchange = $1 AND raw_symbol = $2`, exchange, "UNKN"+exchangeSymbol).Scan(&count); err != nil {
		t.Fatalf("query unknown_symbols returned error: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected unknown symbol row count 1, got %d", count)
	}
}
