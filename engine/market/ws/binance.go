package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"dwizzyBRAIN/shared/schema"
)

const (
	defaultBinanceWSURL     = "wss://stream.binance.com:9443/ws/!ticker@arr"
	defaultBinanceReconnect = 23 * time.Hour
)

// BinanceWSClient streams Binance all-market tickers and normalizes them.
type BinanceWSClient struct {
	url            string
	dialer         *websocket.Dialer
	conn           *websocket.Conn
	reconnectAfter time.Duration
	sessionCtx     context.Context
	sessionCancel  context.CancelFunc
	now            func() time.Time
	requestHeader  http.Header
}

func NewBinanceWSClient() *BinanceWSClient {
	return &BinanceWSClient{
		url:            defaultBinanceWSURL,
		dialer:         websocket.DefaultDialer,
		reconnectAfter: defaultBinanceReconnect,
		now:            time.Now,
	}
}

func (c *BinanceWSClient) Connect(ctx context.Context) error {
	if c.conn != nil {
		return fmt.Errorf("binance websocket already connected")
	}

	sessionCtx, cancel := context.WithTimeout(ctx, c.reconnectAfter)
	conn, _, err := c.dialer.DialContext(sessionCtx, c.url, c.requestHeader)
	if err != nil {
		cancel()
		return fmt.Errorf("dial binance websocket: %w", err)
	}

	c.conn = conn
	c.sessionCtx = sessionCtx
	c.sessionCancel = cancel

	return nil
}

func (c *BinanceWSClient) ReadMessage() ([]schema.RawTicker, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("binance websocket is not connected")
	}

	select {
	case <-c.sessionCtx.Done():
		return nil, fmt.Errorf("binance websocket session expired: %w", c.sessionCtx.Err())
	default:
	}

	_, payload, err := c.conn.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read binance websocket message: %w", err)
	}

	tickers, err := parseBinanceTickerMessage(payload)
	if err != nil {
		return nil, err
	}

	return tickers, nil
}

func (c *BinanceWSClient) Close() error {
	if c.sessionCancel != nil {
		c.sessionCancel()
		c.sessionCancel = nil
	}

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.sessionCtx = nil

	if err != nil {
		return fmt.Errorf("close binance websocket: %w", err)
	}

	return nil
}

type binanceTickerMessage struct {
	EventTime json.RawMessage `json:"E"`
	Symbol    string          `json:"s"`
	Price     json.Number     `json:"c"`
	Bid       json.Number     `json:"b"`
	Ask       json.Number     `json:"a"`
	Volume    json.Number     `json:"v"`
}

func parseBinanceTickerMessage(payload []byte) ([]schema.RawTicker, error) {
	var rawMessages []binanceTickerMessage
	if err := json.Unmarshal(payload, &rawMessages); err != nil {
		return nil, fmt.Errorf("decode binance ticker payload: %w", err)
	}

	tickers := make([]schema.RawTicker, 0, len(rawMessages))
	for _, rawTicker := range rawMessages {
		normalized, err := normalizeBinanceTicker(rawTicker)
		if err != nil {
			return nil, err
		}
		tickers = append(tickers, normalized)
	}

	return tickers, nil
}

func normalizeBinanceTicker(rawTicker binanceTickerMessage) (schema.RawTicker, error) {
	price, err := parseBinanceNumber(rawTicker.Price, rawTicker.Symbol, "price")
	if err != nil {
		return schema.RawTicker{}, fmt.Errorf("parse binance price for %s: %w", rawTicker.Symbol, err)
	}

	bid, err := parseBinanceNumber(rawTicker.Bid, rawTicker.Symbol, "bid")
	if err != nil {
		return schema.RawTicker{}, fmt.Errorf("parse binance bid for %s: %w", rawTicker.Symbol, err)
	}

	ask, err := parseBinanceNumber(rawTicker.Ask, rawTicker.Symbol, "ask")
	if err != nil {
		return schema.RawTicker{}, fmt.Errorf("parse binance ask for %s: %w", rawTicker.Symbol, err)
	}

	volume, err := parseBinanceNumber(rawTicker.Volume, rawTicker.Symbol, "volume")
	if err != nil {
		return schema.RawTicker{}, fmt.Errorf("parse binance volume for %s: %w", rawTicker.Symbol, err)
	}

	eventTime, err := parseBinanceEventTime(rawTicker.EventTime)
	if err != nil {
		return schema.RawTicker{}, fmt.Errorf("parse binance event time for %s: %w", rawTicker.Symbol, err)
	}

	ticker := schema.RawTicker{
		Symbol:    rawTicker.Symbol,
		Price:     price,
		Exchange:  "binance",
		Bid:       bid,
		Ask:       ask,
		Volume:    volume,
		Timestamp: time.UnixMilli(eventTime).UTC(),
	}

	if err := ticker.Validate(); err != nil {
		return schema.RawTicker{}, fmt.Errorf("validate binance ticker %s: %w", rawTicker.Symbol, err)
	}

	return ticker, nil
}

func parseBinanceEventTime(payload json.RawMessage) (int64, error) {
	var asNumber json.Number
	if err := json.Unmarshal(payload, &asNumber); err == nil {
		return asNumber.Int64()
	}

	var asString string
	if err := json.Unmarshal(payload, &asString); err != nil {
		return 0, fmt.Errorf("invalid event time: %w", err)
	}

	return strconv.ParseInt(strings.TrimSpace(asString), 10, 64)
}

func parseBinanceNumber(value json.Number, symbol, field string) (float64, error) {
	if value == "" {
		return 0, fmt.Errorf("binance %s for %s is empty", field, symbol)
	}
	out, err := value.Float64()
	if err != nil {
		return 0, fmt.Errorf("binance %s for %s: %w", field, symbol, err)
	}
	return out, nil
}
