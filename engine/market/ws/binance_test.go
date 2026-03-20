package ws

import (
	"testing"
	"time"
)

func TestParseBinanceTickerMessage(t *testing.T) {
	payload := []byte(`[
		{"E":1710000000123,"s":"BTCUSDT","c":"64000.12","b":"63999.99","a":"64000.50","v":"1234.56"},
		{"E":1710000001123,"s":"ETHUSDT","c":"3200.10","b":"3199.90","a":"3200.20","v":"987.65"}
	]`)

	tickers, err := parseBinanceTickerMessage(payload)
	if err != nil {
		t.Fatalf("parseBinanceTickerMessage() returned error: %v", err)
	}

	if len(tickers) != 2 {
		t.Fatalf("expected 2 tickers, got %d", len(tickers))
	}

	if tickers[0].Symbol != "BTCUSDT" {
		t.Fatalf("expected BTCUSDT, got %s", tickers[0].Symbol)
	}

	if tickers[0].Exchange != "binance" {
		t.Fatalf("expected exchange binance, got %s", tickers[0].Exchange)
	}

	expected := time.UnixMilli(1710000000123).UTC()
	if !tickers[0].Timestamp.Equal(expected) {
		t.Fatalf("expected timestamp %s, got %s", expected, tickers[0].Timestamp)
	}
}

func TestParseBinanceTickerMessageInvalidNumber(t *testing.T) {
	payload := []byte(`[{"E":1710000000123,"s":"BTCUSDT","c":"nope","b":"1","a":"1","v":"1"}]`)

	if _, err := parseBinanceTickerMessage(payload); err == nil {
		t.Fatal("parseBinanceTickerMessage() expected an error for invalid numeric fields")
	}
}
