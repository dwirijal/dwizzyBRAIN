package ohlcv

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubIncrementalSyncer struct {
	err     error
	calls   int
	targets []SyncRequest
}

func (s *stubIncrementalSyncer) IncrementalSync(ctx context.Context, req SyncRequest) ([]Candle, error) {
	s.calls++
	s.targets = append(s.targets, req)
	if s.err != nil {
		return nil, s.err
	}
	return nil, nil
}

func TestSchedulerRunOnce(t *testing.T) {
	service := &stubIncrementalSyncer{}
	scheduler := NewScheduler(service, []SyncRequest{
		{CoinID: "bitcoin", Exchange: "binance", Timeframe: "1m"},
		{CoinID: "ethereum", Exchange: "kraken", Timeframe: "5m"},
	}, time.Minute)

	if err := scheduler.RunOnce(context.Background()); err != nil {
		t.Fatalf("RunOnce() returned error: %v", err)
	}
	if service.calls != 2 {
		t.Fatalf("expected 2 sync calls, got %d", service.calls)
	}
}

func TestSchedulerRunReturnsError(t *testing.T) {
	scheduler := NewScheduler(&stubIncrementalSyncer{err: errors.New("fail")}, []SyncRequest{
		{CoinID: "bitcoin", Exchange: "binance", Timeframe: "1m"},
	}, time.Minute)

	if err := scheduler.Run(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestSchedulerRunStopsOnCancel(t *testing.T) {
	service := &stubIncrementalSyncer{}
	scheduler := NewScheduler(service, []SyncRequest{
		{CoinID: "bitcoin", Exchange: "binance", Timeframe: "1m"},
	}, time.Hour)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := scheduler.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if service.calls != 1 {
		t.Fatalf("expected initial run once, got %d", service.calls)
	}
}
