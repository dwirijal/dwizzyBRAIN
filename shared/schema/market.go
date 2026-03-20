package schema

import (
	"fmt"
	"strings"
	"time"
)

// RawTicker is the normalized market payload shared across ingestion paths.
type RawTicker struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Exchange  string    `json:"exchange"`
	Bid       float64   `json:"bid"`
	Ask       float64   `json:"ask"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

func (t RawTicker) Validate() error {
	if strings.TrimSpace(t.Symbol) == "" {
		return fmt.Errorf("symbol is required")
	}
	if strings.TrimSpace(t.Exchange) == "" {
		return fmt.Errorf("exchange is required")
	}
	if t.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	return nil
}

type ResolvedTicker struct {
	CoinID         string    `json:"coin_id"`
	Symbol         string    `json:"symbol"`
	Exchange       string    `json:"exchange"`
	BaseAsset      string    `json:"base_asset"`
	QuoteAsset     string    `json:"quote_asset"`
	Price          float64   `json:"price"`
	Bid            float64   `json:"bid"`
	Ask            float64   `json:"ask"`
	Volume         float64   `json:"volume"`
	Timestamp      time.Time `json:"timestamp"`
	ResolvedSymbol string    `json:"resolved_symbol"`
}

func (t ResolvedTicker) Validate() error {
	if strings.TrimSpace(t.CoinID) == "" {
		return fmt.Errorf("coin_id is required")
	}
	if strings.TrimSpace(t.Symbol) == "" {
		return fmt.Errorf("symbol is required")
	}
	if strings.TrimSpace(t.Exchange) == "" {
		return fmt.Errorf("exchange is required")
	}
	if t.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	return nil
}
