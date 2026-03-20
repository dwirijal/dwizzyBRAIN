package ohlcv

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dwizzyBRAIN/engine/storage"
)

func TestTimescaleStoreIntegration(t *testing.T) {
	url := os.Getenv("TIMESCALE_URL")
	if url == "" {
		t.Skip("TIMESCALE_URL is not set")
	}

	pool, err := storage.NewTimescalePool(context.Background(), url)
	if err != nil {
		t.Fatalf("NewTimescalePool() returned error: %v", err)
	}
	defer pool.Close()

	store := NewTimescaleStore(pool)
	suffix := time.Now().UnixNano()
	coinID := fmt.Sprintf("ohlcv-test-%d", suffix)
	exchange := "binance"
	timeframe := "1m"
	symbol := fmt.Sprintf("T%d/USDT", suffix%1000000)
	t1 := time.Date(2026, 3, 18, 21, 0, 0, 0, time.UTC)
	t2 := t1.Add(time.Minute)

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM symbols WHERE exchange = $1 AND symbol = $2`, exchange, symbol)
		_, _ = pool.Exec(context.Background(), `
DELETE FROM ohlcv
WHERE coin_id = $1
  AND exchange = $2
  AND timeframe = $3`, coinID, exchange, timeframe)
	})

	input := []Candle{
		{
			Timestamp: t1, CoinID: coinID, Exchange: exchange, Symbol: symbol, Timeframe: timeframe,
			Open: 100, High: 110, Low: 95, Close: 108, Volume: 12.5, QuoteVolume: 1300, Trades: 20, IsClosed: true,
		},
		{
			Timestamp: t2, CoinID: coinID, Exchange: exchange, Symbol: symbol, Timeframe: timeframe,
			Open: 108, High: 112, Low: 107, Close: 111, Volume: 10.1, QuoteVolume: 1120, Trades: 15, IsClosed: true,
		},
	}

	if err := store.UpsertCandles(context.Background(), input); err != nil {
		t.Fatalf("UpsertCandles() returned error: %v", err)
	}

	got, err := store.GetCandles(context.Background(), coinID, exchange, timeframe, 10)
	if err != nil {
		t.Fatalf("GetCandles() returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 candles, got %d", len(got))
	}
	if !got[0].Timestamp.Equal(t2) {
		t.Fatalf("expected latest timestamp %s, got %s", t2, got[0].Timestamp)
	}

	latest, err := store.LatestTimestamp(context.Background(), coinID, exchange, timeframe)
	if err != nil {
		t.Fatalf("LatestTimestamp() returned error: %v", err)
	}
	if !latest.Equal(t2) {
		t.Fatalf("expected latest timestamp %s, got %s", t2, latest)
	}
}
