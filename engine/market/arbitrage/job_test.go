package arbitrage

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"
)

type stubScanner struct {
	items []Opportunity
	err   error
	calls int
}

func (s *stubScanner) Scan(ctx context.Context) ([]Opportunity, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.items, nil
}

func TestJobRunOnce(t *testing.T) {
	var logs bytes.Buffer
	engine := &stubScanner{
		items: []Opportunity{
			{CoinID: "bitcoin", BuyExchange: "binance", SellExchange: "kraken", GrossSpreadPct: 1.2},
		},
	}
	job := NewJob(engine, time.Second, log.New(&logs, "", 0))

	items, err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 opportunity, got %d", len(items))
	}
	if engine.calls != 1 {
		t.Fatalf("expected one scan, got %d", engine.calls)
	}
	if !strings.Contains(logs.String(), "coin=bitcoin") {
		t.Fatalf("expected log output, got %q", logs.String())
	}
}

func TestJobRunReturnsScanError(t *testing.T) {
	job := NewJob(&stubScanner{err: errors.New("boom")}, time.Second, nil)
	if err := job.Run(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestJobRunStopsOnCancel(t *testing.T) {
	engine := &stubScanner{}
	job := NewJob(engine, time.Hour, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := job.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if engine.calls != 1 {
		t.Fatalf("expected initial scan once, got %d", engine.calls)
	}
}
