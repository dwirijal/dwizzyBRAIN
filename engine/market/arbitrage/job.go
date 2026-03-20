package arbitrage

import (
	"context"
	"fmt"
	"log"
	"time"
)

type scanner interface {
	Scan(ctx context.Context) ([]Opportunity, error)
}

type Job struct {
	engine   scanner
	interval time.Duration
	logger   *log.Logger
}

func NewJob(engine scanner, interval time.Duration, logger *log.Logger) *Job {
	if interval <= 0 {
		interval = defaultScanInterval
	}

	return &Job{
		engine:   engine,
		interval: interval,
		logger:   logger,
	}
}

func (j *Job) RunOnce(ctx context.Context) ([]Opportunity, error) {
	if j.engine == nil {
		return nil, fmt.Errorf("arbitrage engine is required")
	}

	items, err := j.engine.Scan(ctx)
	if err != nil {
		return nil, err
	}

	if j.logger != nil {
		for _, item := range items {
			j.logger.Printf(
				"arbitrage detected coin=%s buy=%s sell=%s spread=%.4f depth_buy=%.2f depth_sell=%.2f",
				item.CoinID,
				item.BuyExchange,
				item.SellExchange,
				item.GrossSpreadPct,
				item.BuyDepthUSD,
				item.SellDepthUSD,
			)
		}
	}

	return items, nil
}

func (j *Job) Run(ctx context.Context) error {
	if _, err := j.RunOnce(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := j.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}
