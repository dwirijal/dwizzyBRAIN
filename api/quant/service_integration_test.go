package quantapi

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dwizzyBRAIN/engine/storage"
)

func TestServiceLivePatternSearch(t *testing.T) {
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

	service := NewService(pool)
	result, err := service.Pattern(ctx, "BTC/USDT", "1m", 20, 30)
	if err != nil {
		t.Fatalf("pattern search: %v", err)
	}

	if result.Query.Symbol != "BTC/USDT" {
		t.Fatalf("expected BTC/USDT query symbol, got %s", result.Query.Symbol)
	}
	if result.Query.Timeframe != "1m" {
		t.Fatalf("expected 1m query timeframe, got %s", result.Query.Timeframe)
	}
	if len(result.Query.Fingerprint) != 30 {
		t.Fatalf("expected 30-dim fingerprint, got %d", len(result.Query.Fingerprint))
	}
	if len(result.Matches) == 0 {
		t.Fatalf("expected at least one similarity match")
	}
	if !result.LowConfidence {
		t.Fatalf("expected low confidence with current sample size")
	}
}

func TestServiceLiveSignalEndpoints(t *testing.T) {
	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("POSTGRES_URL is required for integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := storage.NewPostgresPoolFromEnv(ctx)
	if err != nil {
		t.Fatalf("new postgres pool: %v", err)
	}
	defer pool.Close()

	service := NewService(pool)

	symbol := fmt.Sprintf("BTC/USDT-QUANTTEST-%d", time.Now().UnixNano())
	timeframe := "1m"
	exchange := "binance"
	coinID := "bitcoin"
	base := time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC)

	_, err = pool.Exec(ctx, `
INSERT INTO signals (
    coin_id,
    exchange,
    symbol,
    timeframe,
    quant_score,
    signal_type,
    strength,
    funding_rate,
    funding_sentiment,
    volume_spike,
    price_deviation,
    anomaly_score,
    price_at_signal,
    created_at
)
VALUES
    ($1, $2, $3, $4, 74.50, 'buy', 'strong', 0.0005, 'neutral', TRUE, FALSE, 0.723, 65000.0, $5),
    ($1, $2, $3, $4, 61.20, 'hold', 'moderate', NULL, 'neutral', FALSE, TRUE, 0.411, 66000.0, $6),
    ($1, $2, $3, $4, 42.90, 'sell', 'weak', -0.0006, 'short_bias', TRUE, TRUE, 0.855, 64500.0, $7)
`, coinID, exchange, symbol, timeframe, base, base.Add(time.Hour), base.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("insert test signals: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM signals WHERE symbol = $1 AND timeframe = $2`, symbol, timeframe)
	}()

	latest, err := service.SignalLatest(ctx, symbol, timeframe, exchange)
	if err != nil {
		t.Fatalf("latest signal: %v", err)
	}
	if latest.Signal.SignalType != "sell" {
		t.Fatalf("expected latest sell signal, got %s", latest.Signal.SignalType)
	}
	if latest.Signal.QuantScore == nil || *latest.Signal.QuantScore != 42.90 {
		t.Fatalf("expected latest quant score 42.90, got %+v", latest.Signal.QuantScore)
	}

	history, err := service.SignalHistory(ctx, symbol, timeframe, exchange, 2)
	if err != nil {
		t.Fatalf("signal history: %v", err)
	}
	if history.Limit != 2 {
		t.Fatalf("expected history limit 2, got %d", history.Limit)
	}
	if len(history.Items) != 2 {
		t.Fatalf("expected 2 history items, got %d", len(history.Items))
	}
	if history.Items[0].SignalType != "sell" || history.Items[1].SignalType != "hold" {
		t.Fatalf("expected history ordered newest-first, got %+v", history.Items)
	}

	summary, err := service.SignalSummary(ctx, symbol, timeframe, exchange, 3)
	if err != nil {
		t.Fatalf("signal summary: %v", err)
	}
	if summary.Count != 3 {
		t.Fatalf("expected summary count 3, got %d", summary.Count)
	}
	if summary.Latest == nil || summary.Latest.SignalType != "sell" {
		t.Fatalf("expected summary latest sell, got %+v", summary.Latest)
	}
	if summary.AvgQuantScore == nil || *summary.AvgQuantScore < 59.5 || *summary.AvgQuantScore > 59.6 {
		t.Fatalf("expected avg quant score around 59.53, got %+v", summary.AvgQuantScore)
	}
	if got := summary.SignalTypeCounts["buy"]; got != 1 {
		t.Fatalf("expected one buy signal, got %d", got)
	}
	if got := summary.SignalTypeCounts["sell"]; got != 1 {
		t.Fatalf("expected one sell signal, got %d", got)
	}
	if got := summary.SignalTypeCounts["hold"]; got != 1 {
		t.Fatalf("expected one hold signal, got %d", got)
	}
	if summary.VolumeSpikeRate == nil || *summary.VolumeSpikeRate < 0.66 || *summary.VolumeSpikeRate > 0.67 {
		t.Fatalf("expected volume spike rate around 0.666, got %+v", summary.VolumeSpikeRate)
	}
}
