package main

import (
	"testing"
	"time"
)

func TestParseExchanges(t *testing.T) {
	got := parseExchanges(" binance, Kraken ,, mexc ")
	if len(got) != 3 {
		t.Fatalf("expected 3 exchanges, got %d", len(got))
	}
	if got[0] != "binance" || got[1] != "kraken" || got[2] != "mexc" {
		t.Fatalf("unexpected exchanges: %#v", got)
	}
}

func TestParseIntervalDefault(t *testing.T) {
	got, err := parseInterval("")
	if err != nil {
		t.Fatalf("parseInterval() returned error: %v", err)
	}
	if got != 6*time.Hour {
		t.Fatalf("expected 6h, got %s", got)
	}
}

func TestParseIntervalInvalid(t *testing.T) {
	if _, err := parseInterval("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseBoolEnv(t *testing.T) {
	if !parseBoolEnv("true", false) {
		t.Fatal("expected true")
	}
	if parseBoolEnv("false", true) {
		t.Fatal("expected false")
	}
	if !parseBoolEnv("", true) {
		t.Fatal("expected fallback true")
	}
}

func TestParseOHLCVTargets(t *testing.T) {
	got, err := parseOHLCVTargets("bitcoin:binance:1m, ethereum:kraken:5m")
	if err != nil {
		t.Fatalf("parseOHLCVTargets() returned error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(got))
	}
	if got[0].CoinID != "bitcoin" || got[0].Exchange != "binance" || got[0].Timeframe != "1m" {
		t.Fatalf("unexpected first target: %+v", got[0])
	}
}

func TestParseOHLCVTargetsInvalid(t *testing.T) {
	if _, err := parseOHLCVTargets("broken"); err == nil {
		t.Fatal("expected error")
	}
}

func TestParsePositiveIntEnv(t *testing.T) {
	got, err := parsePositiveIntEnv("COINGECKO_COLDLOAD_PAGES", "", 4)
	if err != nil {
		t.Fatalf("parsePositiveIntEnv() returned error: %v", err)
	}
	if got != 4 {
		t.Fatalf("expected fallback 4, got %d", got)
	}

	got, err = parsePositiveIntEnv("COINGECKO_COLDLOAD_PAGES", "6", 4)
	if err != nil {
		t.Fatalf("parsePositiveIntEnv() returned error: %v", err)
	}
	if got != 6 {
		t.Fatalf("expected 6, got %d", got)
	}
}

func TestParsePositiveIntEnvInvalid(t *testing.T) {
	if _, err := parsePositiveIntEnv("COINGECKO_COLDLOAD_PAGES", "0", 4); err == nil {
		t.Fatal("expected error")
	}
}
