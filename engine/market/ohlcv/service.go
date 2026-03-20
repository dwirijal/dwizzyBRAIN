package ohlcv

import (
	"context"
	"fmt"
	"strings"
	"time"

	"dwizzyBRAIN/engine/market"
	"dwizzyBRAIN/engine/market/mapping"

	ccxt "github.com/ccxt/ccxt/go/v4"
)

type symbolResolver interface {
	ResolveExchangeSymbol(ctx context.Context, coinID, exchange string) (mapping.Mapping, error)
}

type fetcher interface {
	PollOHLCV(ctx context.Context, exchangeID, symbol, timeframe string, since time.Time, limit int) ([]ccxt.OHLCV, error)
}

type store interface {
	UpsertCandles(ctx context.Context, candles []Candle) error
	GetCandles(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]Candle, error)
	LatestTimestamp(ctx context.Context, coinID, exchange, timeframe string) (time.Time, error)
}

type publisher interface {
	PublishOHLCV(ctx context.Context, candle market.OHLCVMessage) error
}

type SyncRequest struct {
	CoinID    string
	Exchange  string
	Timeframe string
	Since     time.Time
	Limit     int
}

type Candle struct {
	Timestamp   time.Time `json:"timestamp"`
	CoinID      string    `json:"coin_id"`
	Exchange    string    `json:"exchange"`
	Symbol      string    `json:"symbol"`
	Timeframe   string    `json:"timeframe"`
	Open        float64   `json:"open"`
	High        float64   `json:"high"`
	Low         float64   `json:"low"`
	Close       float64   `json:"close"`
	Volume      float64   `json:"volume"`
	QuoteVolume float64   `json:"quote_volume"`
	Trades      int       `json:"trades"`
	IsClosed    bool      `json:"is_closed"`
}

type Service struct {
	resolver  symbolResolver
	fetcher   fetcher
	store     store
	publisher publisher
	now       func() time.Time
}

func NewService(resolver symbolResolver, fetcher fetcher, store store, publisher publisher) *Service {
	return &Service{
		resolver:  resolver,
		fetcher:   fetcher,
		store:     store,
		publisher: publisher,
		now:       time.Now,
	}
}

func (s *Service) GetOHLCV(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]Candle, error) {
	if s.store == nil {
		return nil, fmt.Errorf("ohlcv store is required")
	}

	return s.store.GetCandles(ctx, strings.TrimSpace(coinID), normalizeExchange(exchange), normalizeTimeframe(timeframe), limit)
}

func (s *Service) BackfillOHLCV(ctx context.Context, req SyncRequest) ([]Candle, error) {
	return s.sync(ctx, req, req.Since)
}

func (s *Service) IncrementalSync(ctx context.Context, req SyncRequest) ([]Candle, error) {
	if s.store == nil {
		return nil, fmt.Errorf("ohlcv store is required")
	}

	since := req.Since
	if latest, err := s.store.LatestTimestamp(ctx, strings.TrimSpace(req.CoinID), normalizeExchange(req.Exchange), normalizeTimeframe(req.Timeframe)); err != nil {
		return nil, fmt.Errorf("latest ohlcv timestamp: %w", err)
	} else if !latest.IsZero() {
		since = latest
	}

	return s.sync(ctx, req, since)
}

func (s *Service) sync(ctx context.Context, req SyncRequest, since time.Time) ([]Candle, error) {
	if s.resolver == nil {
		return nil, fmt.Errorf("symbol resolver is required")
	}
	if s.fetcher == nil {
		return nil, fmt.Errorf("ohlcv fetcher is required")
	}
	if s.store == nil {
		return nil, fmt.Errorf("ohlcv store is required")
	}

	req.CoinID = strings.TrimSpace(req.CoinID)
	req.Exchange = normalizeExchange(req.Exchange)
	req.Timeframe = normalizeTimeframe(req.Timeframe)
	if req.CoinID == "" || req.Exchange == "" || req.Timeframe == "" {
		return nil, fmt.Errorf("coin_id, exchange, and timeframe are required")
	}

	resolved, err := s.resolver.ResolveExchangeSymbol(ctx, req.CoinID, req.Exchange)
	if err != nil {
		return nil, fmt.Errorf("resolve exchange symbol: %w", err)
	}

	rawCandles, err := s.fetcher.PollOHLCV(ctx, req.Exchange, resolved.ExchangeSymbol, req.Timeframe, since, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("poll ohlcv: %w", err)
	}

	candles, err := s.normalizeCandles(req, resolved.ExchangeSymbol, rawCandles)
	if err != nil {
		return nil, fmt.Errorf("normalize candles: %w", err)
	}
	if len(candles) == 0 {
		return nil, nil
	}

	if err := s.store.UpsertCandles(ctx, candles); err != nil {
		return nil, fmt.Errorf("upsert candles: %w", err)
	}

	if s.publisher != nil {
		for _, candle := range candles {
			if err := s.publisher.PublishOHLCV(ctx, market.OHLCVMessage{
				Symbol:    candle.Symbol,
				Exchange:  candle.Exchange,
				Timeframe: candle.Timeframe,
				Timestamp: candle.Timestamp,
				Open:      candle.Open,
				High:      candle.High,
				Low:       candle.Low,
				Close:     candle.Close,
				Volume:    candle.Volume,
			}); err != nil {
				return nil, fmt.Errorf("publish candle %s %s %s: %w", candle.Exchange, candle.Symbol, candle.Timeframe, err)
			}
		}
	}

	return candles, nil
}

func (s *Service) normalizeCandles(req SyncRequest, symbol string, raw []ccxt.OHLCV) ([]Candle, error) {
	tfDuration, err := timeframeDuration(req.Timeframe)
	if err != nil {
		return nil, err
	}

	now := s.now().UTC()
	candles := make([]Candle, 0, len(raw))
	for _, item := range raw {
		ts := time.UnixMilli(item.Timestamp).UTC()
		if ts.IsZero() {
			return nil, fmt.Errorf("ohlcv timestamp is required")
		}

		candles = append(candles, Candle{
			Timestamp: ts,
			CoinID:    req.CoinID,
			Exchange:  req.Exchange,
			Symbol:    symbol,
			Timeframe: req.Timeframe,
			Open:      item.Open,
			High:      item.High,
			Low:       item.Low,
			Close:     item.Close,
			Volume:    item.Volume,
			IsClosed:  ts.Add(tfDuration).Before(now) || ts.Add(tfDuration).Equal(now),
		})
	}

	return candles, nil
}

func timeframeDuration(timeframe string) (time.Duration, error) {
	switch normalizeTimeframe(timeframe) {
	case "1m":
		return time.Minute, nil
	case "5m":
		return 5 * time.Minute, nil
	case "15m":
		return 15 * time.Minute, nil
	case "1h":
		return time.Hour, nil
	case "4h":
		return 4 * time.Hour, nil
	case "1d":
		return 24 * time.Hour, nil
	case "1w":
		return 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported timeframe %q", timeframe)
	}
}

func normalizeExchange(exchange string) string {
	return strings.ToLower(strings.TrimSpace(exchange))
}

func normalizeTimeframe(timeframe string) string {
	return strings.ToLower(strings.TrimSpace(timeframe))
}
