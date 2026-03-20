package market

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	redis "github.com/redis/go-redis/v9"

	"dwizzyBRAIN/shared/schema"
)

func TestPublisherPublishTicker(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	publisher := NewPublisher(client)
	ticker := schema.RawTicker{
		Symbol:    "BTCUSDT",
		Exchange:  "binance",
		Price:     64000.12,
		Bid:       63999.50,
		Ask:       64000.50,
		Volume:    1234.56,
		Timestamp: time.Unix(1710000000, 0).UTC(),
	}

	if err := publisher.PublishTicker(context.Background(), ticker); err != nil {
		t.Fatalf("PublishTicker() returned error: %v", err)
	}

	value, err := server.Get("price:BTCUSDT:binance")
	if err != nil {
		t.Fatalf("expected hot price cache key: %v", err)
	}

	if value != "64000.12" {
		t.Fatalf("expected cached price 64000.12, got %s", value)
	}

	ttl := server.TTL("price:BTCUSDT:binance")
	if ttl <= 0 || ttl > 10*time.Second {
		t.Fatalf("expected TTL between 1ns and 10s, got %s", ttl)
	}
}

func TestPublisherPublishResolvedTicker(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	publisher := NewPublisher(client)
	ticker := schema.ResolvedTicker{
		CoinID:         "bitcoin",
		Symbol:         "BTCUSDT",
		Exchange:       "binance",
		BaseAsset:      "BTC",
		QuoteAsset:     "USDT",
		ResolvedSymbol: "BTCUSDT",
		Price:          64000.12,
		Bid:            63999.50,
		Ask:            64000.50,
		Volume:         1234.56,
		Timestamp:      time.Unix(1710000000, 0).UTC(),
	}

	if err := publisher.PublishResolvedTicker(context.Background(), ticker); err != nil {
		t.Fatalf("PublishResolvedTicker() returned error: %v", err)
	}

	value, err := server.Get("price:bitcoin:binance")
	if err != nil {
		t.Fatalf("expected coin_id hot cache key: %v", err)
	}
	if value != "64000.12" {
		t.Fatalf("expected cached coin_id price 64000.12, got %s", value)
	}
}

func TestPublisherPublishOHLCV(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	publisher := NewPublisher(client)
	ctx := context.Background()
	pubsub := client.Subscribe(ctx, "ch:ohlcv:raw:BTCUSDT:binance:1m")
	defer pubsub.Close()

	if _, err := pubsub.Receive(ctx); err != nil {
		t.Fatalf("Receive() returned error: %v", err)
	}

	candle := OHLCVMessage{
		Symbol:    "BTCUSDT",
		Exchange:  "binance",
		Timeframe: "1m",
		Timestamp: time.Unix(1710000000, 0).UTC(),
		Open:      63950,
		High:      64100,
		Low:       63900,
		Close:     64050,
		Volume:    456.78,
	}

	if err := publisher.PublishOHLCV(ctx, candle); err != nil {
		t.Fatalf("PublishOHLCV() returned error: %v", err)
	}

	values, err := server.List("ohlcv:BTCUSDT:binance:1m")
	if err != nil {
		t.Fatalf("expected candle list key: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("expected one candle payload in list, got %d", len(values))
	}

	var stored OHLCVMessage
	if err := json.Unmarshal([]byte(values[0]), &stored); err != nil {
		t.Fatalf("json.Unmarshal() returned error: %v", err)
	}

	if stored.Close != candle.Close {
		t.Fatalf("expected close %.2f, got %.2f", candle.Close, stored.Close)
	}

	message, err := pubsub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatalf("ReceiveMessage() returned error: %v", err)
	}

	if message.Channel != "ch:ohlcv:raw:BTCUSDT:binance:1m" {
		t.Fatalf("unexpected channel %s", message.Channel)
	}
}

func TestPublisherPublishOHLCVTrimsList(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()

	publisher := NewPublisher(client)
	ctx := context.Background()

	for i := 0; i < 205; i++ {
		candle := OHLCVMessage{
			Symbol:    "ETHUSDT",
			Exchange:  "binance",
			Timeframe: "1m",
			Timestamp: time.Unix(int64(i), 0).UTC(),
			Open:      float64(i),
			High:      float64(i) + 1,
			Low:       float64(i) - 1,
			Close:     float64(i),
			Volume:    float64(i),
		}

		if err := publisher.PublishOHLCV(ctx, candle); err != nil {
			t.Fatalf("PublishOHLCV() returned error on iteration %d: %v", i, err)
		}
	}

	values, err := server.List("ohlcv:ETHUSDT:binance:1m")
	if err != nil {
		t.Fatalf("expected candle list key: %v", err)
	}

	if got := len(values); got != 200 {
		t.Fatalf("expected list trimmed to 200 items, got %d", got)
	}
}
