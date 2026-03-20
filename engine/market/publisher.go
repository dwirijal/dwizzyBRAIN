package market

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	redis "github.com/redis/go-redis/v9"

	"dwizzyBRAIN/shared/schema"
)

const (
	priceTTL       = 10 * time.Second
	ohlcvListLimit = 199
)

type Publisher struct {
	client redis.Cmdable
}

type OHLCVMessage struct {
	Symbol    string    `json:"symbol"`
	Exchange  string    `json:"exchange"`
	Timeframe string    `json:"timeframe"`
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

func NewPublisher(client redis.Cmdable) *Publisher {
	return &Publisher{client: client}
}

func (p *Publisher) PublishTicker(ctx context.Context, ticker schema.RawTicker) error {
	if p.client == nil {
		return fmt.Errorf("publisher client is required")
	}
	if err := ticker.Validate(); err != nil {
		return fmt.Errorf("validate ticker: %w", err)
	}

	key := fmt.Sprintf("price:%s:%s", cleanToken(ticker.Symbol), cleanToken(ticker.Exchange))
	value := strconv.FormatFloat(ticker.Price, 'f', -1, 64)

	if err := p.client.Set(ctx, key, value, priceTTL).Err(); err != nil {
		return fmt.Errorf("cache price %s: %w", key, err)
	}

	return nil
}

func (p *Publisher) PublishResolvedTicker(ctx context.Context, ticker schema.ResolvedTicker) error {
	if p.client == nil {
		return fmt.Errorf("publisher client is required")
	}
	if err := ticker.Validate(); err != nil {
		return fmt.Errorf("validate resolved ticker: %w", err)
	}

	raw := schema.RawTicker{
		Symbol:    ticker.Symbol,
		Exchange:  ticker.Exchange,
		Price:     ticker.Price,
		Bid:       ticker.Bid,
		Ask:       ticker.Ask,
		Volume:    ticker.Volume,
		Timestamp: ticker.Timestamp,
	}
	if err := p.PublishTicker(ctx, raw); err != nil {
		return err
	}

	coinKey := fmt.Sprintf("price:%s:%s", cleanToken(ticker.CoinID), cleanToken(ticker.Exchange))
	value := strconv.FormatFloat(ticker.Price, 'f', -1, 64)
	if err := p.client.Set(ctx, coinKey, value, priceTTL).Err(); err != nil {
		return fmt.Errorf("cache resolved price %s: %w", coinKey, err)
	}

	return nil
}

func (p *Publisher) PublishOHLCV(ctx context.Context, candle OHLCVMessage) error {
	if p.client == nil {
		return fmt.Errorf("publisher client is required")
	}
	if err := candle.Validate(); err != nil {
		return fmt.Errorf("validate candle: %w", err)
	}

	payload, err := json.Marshal(candle)
	if err != nil {
		return fmt.Errorf("marshal candle: %w", err)
	}

	listKey := fmt.Sprintf("ohlcv:%s:%s:%s", cleanToken(candle.Symbol), cleanToken(candle.Exchange), cleanToken(candle.Timeframe))
	if err := p.client.LPush(ctx, listKey, payload).Err(); err != nil {
		return fmt.Errorf("push candle %s: %w", listKey, err)
	}
	if err := p.client.LTrim(ctx, listKey, 0, ohlcvListLimit).Err(); err != nil {
		return fmt.Errorf("trim candle list %s: %w", listKey, err)
	}

	channel := fmt.Sprintf("ch:ohlcv:raw:%s:%s:%s", cleanToken(candle.Symbol), cleanToken(candle.Exchange), cleanToken(candle.Timeframe))
	if err := p.client.Publish(ctx, channel, payload).Err(); err != nil {
		return fmt.Errorf("publish candle %s: %w", channel, err)
	}

	return nil
}

func (m OHLCVMessage) Validate() error {
	if cleanToken(m.Symbol) == "" {
		return fmt.Errorf("symbol is required")
	}
	if cleanToken(m.Exchange) == "" {
		return fmt.Errorf("exchange is required")
	}
	if cleanToken(m.Timeframe) == "" {
		return fmt.Errorf("timeframe is required")
	}
	if m.Timestamp.IsZero() {
		return fmt.Errorf("timestamp is required")
	}

	return nil
}

func cleanToken(value string) string {
	return strings.TrimSpace(value)
}
