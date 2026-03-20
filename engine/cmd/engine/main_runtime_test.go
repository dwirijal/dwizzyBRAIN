package main

import "testing"

func TestParseTickerPollTargets(t *testing.T) {
	got, err := parseTickerPollTargets("bitcoin:binance, ethereum:kraken")
	if err != nil {
		t.Fatalf("parseTickerPollTargets() returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(got))
	}
	if got[0].CoinID != "bitcoin" || got[0].Exchange != "binance" {
		t.Fatalf("unexpected first target: %+v", got[0])
	}
	if got[1].CoinID != "ethereum" || got[1].Exchange != "kraken" {
		t.Fatalf("unexpected second target: %+v", got[1])
	}
}

func TestParseTickerPollTargetsInvalid(t *testing.T) {
	if _, err := parseTickerPollTargets("broken"); err == nil {
		t.Fatal("expected error")
	}
}
