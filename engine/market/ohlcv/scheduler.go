package ohlcv

import (
	"context"
	"fmt"
	"time"
)

const defaultSyncInterval = time.Minute

type incrementalSyncer interface {
	IncrementalSync(ctx context.Context, req SyncRequest) ([]Candle, error)
}

type Scheduler struct {
	service  incrementalSyncer
	targets  []SyncRequest
	interval time.Duration
}

func NewScheduler(service incrementalSyncer, targets []SyncRequest, interval time.Duration) *Scheduler {
	if interval <= 0 {
		interval = defaultSyncInterval
	}

	return &Scheduler{
		service:  service,
		targets:  targets,
		interval: interval,
	}
}

func (s *Scheduler) RunOnce(ctx context.Context) error {
	if s.service == nil {
		return fmt.Errorf("ohlcv service is required")
	}

	for _, target := range s.targets {
		if _, err := s.service.IncrementalSync(ctx, target); err != nil {
			return fmt.Errorf("incremental sync %s %s %s: %w", target.CoinID, target.Exchange, target.Timeframe, err)
		}
	}

	return nil
}

func (s *Scheduler) Run(ctx context.Context) error {
	if err := s.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := s.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}
