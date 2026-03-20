package arbitrage

import "testing"

func TestFormatDiscordAlert(t *testing.T) {
	msg := FormatDiscordAlert(Opportunity{
		CoinID:         "bitcoin",
		BuyExchange:    "binance",
		SellExchange:   "kraken",
		BuyPrice:       100,
		SellPrice:      102,
		GrossSpreadPct: 2,
	})
	if msg == "" {
		t.Fatal("expected non-empty alert message")
	}
}
