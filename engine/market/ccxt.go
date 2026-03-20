package market

import (
	"context"
	"fmt"
	"strings"
	"time"

	ccxt "github.com/ccxt/ccxt/go/v4"

	"dwizzyBRAIN/shared/schema"
)

type exchangeFactory func(options map[string]interface{}) ccxtExchange

type ccxtExchange interface {
	LoadMarkets(params ...interface{}) (map[string]ccxt.MarketInterface, error)
	FetchTicker(symbol string, options ...ccxt.FetchTickerOptions) (ccxt.Ticker, error)
	FetchOHLCV(symbol string, options ...ccxt.FetchOHLCVOptions) ([]ccxt.OHLCV, error)
}

// CCXTManager wraps REST-based fallback polling for exchanges that are not on
// native websocket feeds yet.
type CCXTManager struct {
	exchanges map[string]ccxtExchange
	now       func() time.Time
}

func NewCCXTManager(exchangeIDs []string) (*CCXTManager, error) {
	manager := &CCXTManager{
		exchanges: make(map[string]ccxtExchange, len(exchangeIDs)),
		now:       time.Now,
	}

	for _, exchangeID := range exchangeIDs {
		exchange, err := newExchange(exchangeID)
		if err != nil {
			return nil, err
		}
		manager.exchanges[exchangeID] = exchange
	}

	return manager, nil
}

func (m *CCXTManager) Register(exchangeID string) error {
	exchange, err := newExchange(exchangeID)
	if err != nil {
		return err
	}

	m.exchanges[exchangeID] = exchange
	return nil
}

func (m *CCXTManager) PollTicker(ctx context.Context, exchangeID, symbol string) (schema.RawTicker, error) {
	exchangeID = strings.ToLower(strings.TrimSpace(exchangeID))
	symbol = strings.TrimSpace(symbol)

	if symbol == "" {
		return schema.RawTicker{}, fmt.Errorf("symbol is required")
	}

	exchange, ok := m.exchanges[exchangeID]
	if !ok {
		return schema.RawTicker{}, fmt.Errorf("exchange %q is not registered", exchangeID)
	}

	if err := ctx.Err(); err != nil {
		return schema.RawTicker{}, fmt.Errorf("poll ticker cancelled: %w", err)
	}

	if _, err := exchange.LoadMarkets(); err != nil {
		return schema.RawTicker{}, fmt.Errorf("load markets for %s: %w", exchangeID, err)
	}

	tickerValue, err := exchange.FetchTicker(symbol)
	if err != nil {
		return schema.RawTicker{}, fmt.Errorf("fetch ticker %s on %s: %w", symbol, exchangeID, err)
	}

	ticker, err := normalizeCCXTTicker(exchangeID, symbol, tickerValue, m.now)
	if err != nil {
		return schema.RawTicker{}, fmt.Errorf("normalize ticker %s on %s: %w", symbol, exchangeID, err)
	}

	return ticker, nil
}

func (m *CCXTManager) PollOHLCV(ctx context.Context, exchangeID, symbol, timeframe string, since time.Time, limit int) ([]ccxt.OHLCV, error) {
	exchangeID = strings.ToLower(strings.TrimSpace(exchangeID))
	symbol = strings.TrimSpace(symbol)
	timeframe = strings.TrimSpace(timeframe)

	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if timeframe == "" {
		return nil, fmt.Errorf("timeframe is required")
	}

	exchange, ok := m.exchanges[exchangeID]
	if !ok {
		return nil, fmt.Errorf("exchange %q is not registered", exchangeID)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("poll ohlcv cancelled: %w", err)
	}

	if _, err := exchange.LoadMarkets(); err != nil {
		return nil, fmt.Errorf("load markets for %s: %w", exchangeID, err)
	}

	options := []ccxt.FetchOHLCVOptions{
		ccxt.WithFetchOHLCVTimeframe(timeframe),
	}
	if !since.IsZero() {
		options = append(options, ccxt.WithFetchOHLCVSince(since.UTC().UnixMilli()))
	}
	if limit > 0 {
		options = append(options, ccxt.WithFetchOHLCVLimit(int64(limit)))
	}

	candles, err := exchange.FetchOHLCV(symbol, options...)
	if err != nil {
		return nil, fmt.Errorf("fetch ohlcv %s %s on %s: %w", symbol, timeframe, exchangeID, err)
	}

	return candles, nil
}

func (m *CCXTManager) LoadMarkets(ctx context.Context, exchangeID string) (map[string]ccxt.MarketInterface, error) {
	exchangeID = strings.ToLower(strings.TrimSpace(exchangeID))
	if exchangeID == "" {
		return nil, fmt.Errorf("exchange id is required")
	}

	exchange, ok := m.exchanges[exchangeID]
	if !ok {
		return nil, fmt.Errorf("exchange %q is not registered", exchangeID)
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("load markets cancelled: %w", err)
	}

	markets, err := exchange.LoadMarkets()
	if err != nil {
		return nil, fmt.Errorf("load markets for %s: %w", exchangeID, err)
	}

	return markets, nil
}

func newExchange(exchangeID string) (ccxtExchange, error) {
	factory, ok := exchangeFactories()[strings.ToLower(strings.TrimSpace(exchangeID))]
	if !ok {
		return nil, fmt.Errorf("unsupported exchange %q", exchangeID)
	}

	options := map[string]interface{}{
		"enableRateLimit": true,
	}

	return factory(options), nil
}

func exchangeFactories() map[string]exchangeFactory {
	return map[string]exchangeFactory{
		"binance": func(options map[string]interface{}) ccxtExchange { return ccxt.NewBinance(options) },
		"bybit":   func(options map[string]interface{}) ccxtExchange { return ccxt.NewBybit(options) },
		"gateio":  func(options map[string]interface{}) ccxtExchange { return ccxt.NewGateio(options) },
		"htx":     func(options map[string]interface{}) ccxtExchange { return ccxt.NewHtx(options) },
		"kraken":  func(options map[string]interface{}) ccxtExchange { return ccxt.NewKraken(options) },
		"kucoin":  func(options map[string]interface{}) ccxtExchange { return ccxt.NewKucoin(options) },
		"mexc":    func(options map[string]interface{}) ccxtExchange { return ccxt.NewMexc(options) },
		"okx":     func(options map[string]interface{}) ccxtExchange { return ccxt.NewOkx(options) },
	}
}

func normalizeCCXTTicker(exchangeID, fallbackSymbol string, raw ccxt.Ticker, now func() time.Time) (schema.RawTicker, error) {
	symbol := fallbackSymbol
	if raw.Symbol != nil && strings.TrimSpace(*raw.Symbol) != "" {
		symbol = *raw.Symbol
	}

	var timestamp time.Time
	if raw.Timestamp != nil && *raw.Timestamp > 0 {
		timestamp = time.UnixMilli(*raw.Timestamp).UTC()
	}
	if timestamp.IsZero() {
		timestamp = now().UTC()
	}

	ticker := schema.RawTicker{
		Symbol:    symbol,
		Exchange:  exchangeID,
		Price:     firstNonZero(pointerValue(raw.Last), pointerValue(raw.Close)),
		Bid:       pointerValue(raw.Bid),
		Ask:       pointerValue(raw.Ask),
		Volume:    firstNonZero(pointerValue(raw.BaseVolume), pointerValue(raw.QuoteVolume)),
		Timestamp: timestamp,
	}

	if err := ticker.Validate(); err != nil {
		return schema.RawTicker{}, err
	}

	return ticker, nil
}

func pointerValue[T ~float64](value *T) float64 {
	if value == nil {
		return 0
	}

	return float64(*value)
}

func firstNonZero(values ...float64) float64 {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}

	return 0
}
