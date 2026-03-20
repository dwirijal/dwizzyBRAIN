package schema

import (
	"testing"
	"time"
)

func TestRawTickerValidate(t *testing.T) {
	ticker := RawTicker{
		Symbol:    "BTCUSDT",
		Exchange:  "binance",
		Price:     64000,
		Timestamp: time.Unix(1710000000, 0),
	}

	if err := ticker.Validate(); err != nil {
		t.Fatalf("Validate() returned error: %v", err)
	}
}

func TestRawTickerValidateMissingFields(t *testing.T) {
	ticker := RawTicker{}

	if err := ticker.Validate(); err == nil {
		t.Fatal("Validate() expected an error for missing required fields")
	}
}
