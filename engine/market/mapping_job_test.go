package market

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
)

type stubMappingSyncer struct {
	results   []MappingSyncResult
	err       error
	calls     int
	exchanges []string
}

func (s *stubMappingSyncer) SyncAll(ctx context.Context, exchanges []string) ([]MappingSyncResult, error) {
	s.calls++
	s.exchanges = append([]string(nil), exchanges...)
	if s.err != nil {
		return nil, s.err
	}
	return s.results, nil
}

func TestMappingSyncJobRunOnce(t *testing.T) {
	var logs bytes.Buffer
	syncer := &stubMappingSyncer{
		results: []MappingSyncResult{
			{
				Exchange:   "kraken",
				Build:      mapping.BuildResult{Matched: 3, Unmatched: 1},
				Validation: mapping.ValidationResult{Validated: 2, Active: 1, Delisted: 1},
			},
		},
	}
	job := NewMappingSyncJob(syncer, []string{" Kraken ", " "}, time.Minute, log.New(&logs, "", 0))

	results, err := job.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce() returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if syncer.calls != 1 {
		t.Fatalf("expected syncer to be called once, got %d", syncer.calls)
	}
	if len(syncer.exchanges) != 1 || syncer.exchanges[0] != "kraken" {
		t.Fatalf("unexpected exchanges: %#v", syncer.exchanges)
	}
	if !strings.Contains(logs.String(), "exchange=kraken") {
		t.Fatalf("expected log output to contain exchange, got %q", logs.String())
	}
}

func TestMappingSyncJobRunReturnsError(t *testing.T) {
	job := NewMappingSyncJob(&stubMappingSyncer{err: errors.New("sync failed")}, []string{"kraken"}, time.Millisecond, nil)
	if err := job.Run(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestMappingSyncJobRunStopsOnContextCancel(t *testing.T) {
	syncer := &stubMappingSyncer{}
	job := NewMappingSyncJob(syncer, []string{"kraken"}, time.Hour, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := job.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if syncer.calls != 1 {
		t.Fatalf("expected one initial sync before shutdown, got %d", syncer.calls)
	}
}

func TestMappingSyncJobRunOnceRequiresExchange(t *testing.T) {
	job := NewMappingSyncJob(&stubMappingSyncer{}, nil, time.Minute, nil)
	if _, err := job.RunOnce(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}
