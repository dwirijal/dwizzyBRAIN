package market

import (
	"testing"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

func TestNormalizeCCXTTicker(t *testing.T) {
	symbol := "BTC/USDT"
	last := 64000.25
	bid := 63999.50
	ask := 64000.75
	baseVolume := 1234.56
	timestamp := int64(1710000000123)
	raw := ccxt.Ticker{
		Symbol:     &symbol,
		Last:       &last,
		Bid:        &bid,
		Ask:        &ask,
		BaseVolume: &baseVolume,
		Timestamp:  &timestamp,
	}

	ticker, err := normalizeCCXTTicker("kraken", "BTC/USDT", raw, time.Now)
	if err != nil {
		t.Fatalf("normalizeCCXTTicker() returned error: %v", err)
	}

	if ticker.Exchange != "kraken" {
		t.Fatalf("expected exchange kraken, got %s", ticker.Exchange)
	}

	if ticker.Price != *raw.Last {
		t.Fatalf("expected price %.2f, got %.2f", *raw.Last, ticker.Price)
	}
}

func TestNormalizeCCXTTickerFallbackTimestamp(t *testing.T) {
	now := func() time.Time {
		return time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	}

	symbol := "ETH/USDT"
	last := 3200.10
	raw := ccxt.Ticker{
		Symbol: &symbol,
		Last:   &last,
	}

	ticker, err := normalizeCCXTTicker("kucoin", "ETH/USDT", raw, now)
	if err != nil {
		t.Fatalf("normalizeCCXTTicker() returned error: %v", err)
	}

	if !ticker.Timestamp.Equal(now()) {
		t.Fatalf("expected fallback timestamp %s, got %s", now(), ticker.Timestamp)
	}
}
