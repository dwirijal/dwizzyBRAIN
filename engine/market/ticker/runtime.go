package ticker

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"dwizzyBRAIN/engine/market"
	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/engine/market/ws"
	"dwizzyBRAIN/shared/schema"
)

type streamIngestion interface {
	ProcessBatchBestEffort(ctx context.Context, raws []schema.RawTicker) ([]schema.ResolvedTicker, []error)
}

type pollingIngestion interface {
	ProcessTicker(ctx context.Context, raw schema.RawTicker) (schema.ResolvedTicker, error)
}

type symbolLookup interface {
	ResolveExchangeSymbol(ctx context.Context, coinID, exchange string) (mapping.Mapping, error)
}

type tickerPoller interface {
	PollTicker(ctx context.Context, exchangeID, symbol string) (schema.RawTicker, error)
}

type BinanceStreamJob struct {
	client    *ws.BinanceWSClient
	ingestion streamIngestion
	agg       *Aggregator
	logger    *log.Logger
	retryWait time.Duration
}

func NewBinanceStreamJob(client *ws.BinanceWSClient, ingestion streamIngestion, agg *Aggregator, logger *log.Logger) *BinanceStreamJob {
	return &BinanceStreamJob{
		client:    client,
		ingestion: ingestion,
		agg:       agg,
		logger:    logger,
		retryWait: 2 * time.Second,
	}
}

func (j *BinanceStreamJob) Run(ctx context.Context) error {
	if j.client == nil {
		return fmt.Errorf("binance client is required")
	}
	if j.ingestion == nil {
		return fmt.Errorf("ingestion service is required")
	}
	if j.agg == nil {
		return fmt.Errorf("aggregator is required")
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		if err := j.client.Connect(ctx); err != nil {
			if err := waitOrCancel(ctx, j.retryWait); err != nil {
				return err
			}
			continue
		}

		err := j.readLoop(ctx)
		_ = j.client.Close()
		if err == nil || errors.Is(err, context.Canceled) {
			return err
		}
		if j.logger != nil {
			j.logger.Printf("binance stream reconnecting after error: %v", err)
		}
		if err := waitOrCancel(ctx, j.retryWait); err != nil {
			return err
		}
	}
}

func (j *BinanceStreamJob) readLoop(ctx context.Context) error {
	for {
		raws, err := j.client.ReadMessage()
		if err != nil {
			return err
		}
		resolved, errs := j.ingestion.ProcessBatchBestEffort(ctx, raws)
		for _, item := range resolved {
			if _, err := j.agg.Update(item); err != nil && j.logger != nil {
				j.logger.Printf("aggregate resolved ticker failed: %v", err)
			}
		}
		if len(errs) > 0 && j.logger != nil {
			j.logger.Printf("binance batch processed with %d skipped/error tickers", len(errs))
		}
	}
}

type PollTarget struct {
	CoinID   string
	Exchange string
}

type CCXTPollJob struct {
	poller    tickerPoller
	resolver  symbolLookup
	ingestion pollingIngestion
	agg       *Aggregator
	targets   []PollTarget
	interval  time.Duration
	logger    *log.Logger
}

func NewCCXTPollJob(poller tickerPoller, resolver symbolLookup, ingestion pollingIngestion, agg *Aggregator, targets []PollTarget, interval time.Duration, logger *log.Logger) *CCXTPollJob {
	if interval <= 0 {
		interval = 10 * time.Second
	}

	return &CCXTPollJob{
		poller:    poller,
		resolver:  resolver,
		ingestion: ingestion,
		agg:       agg,
		targets:   targets,
		interval:  interval,
		logger:    logger,
	}
}

func (j *CCXTPollJob) RunOnce(ctx context.Context) error {
	if j.poller == nil || j.resolver == nil || j.ingestion == nil || j.agg == nil {
		return fmt.Errorf("poller, resolver, ingestion, and aggregator are required")
	}

	for _, target := range j.targets {
		mapped, err := j.resolver.ResolveExchangeSymbol(ctx, target.CoinID, target.Exchange)
		if err != nil {
			return fmt.Errorf("resolve symbol for %s on %s: %w", target.CoinID, target.Exchange, err)
		}
		raw, err := j.poller.PollTicker(ctx, target.Exchange, mapped.ExchangeSymbol)
		if err != nil {
			return fmt.Errorf("poll ticker for %s on %s: %w", target.CoinID, target.Exchange, err)
		}
		resolved, err := j.ingestion.ProcessTicker(ctx, raw)
		if err != nil {
			return fmt.Errorf("process ticker for %s on %s: %w", target.CoinID, target.Exchange, err)
		}
		if _, err := j.agg.Update(resolved); err != nil {
			return fmt.Errorf("aggregate ticker for %s on %s: %w", target.CoinID, target.Exchange, err)
		}
		if j.logger != nil {
			j.logger.Printf("ccxt poll updated coin=%s exchange=%s symbol=%s", target.CoinID, target.Exchange, mapped.ExchangeSymbol)
		}
	}

	return nil
}

func (j *CCXTPollJob) Run(ctx context.Context) error {
	if err := j.RunOnce(ctx); err != nil {
		return err
	}

	t := time.NewTicker(j.interval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := j.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}

type SpreadJob struct {
	recorder *SpreadRecorder
	interval time.Duration
	logger   *log.Logger
}

func NewSpreadJob(recorder *SpreadRecorder, interval time.Duration, logger *log.Logger) *SpreadJob {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	return &SpreadJob{recorder: recorder, interval: interval, logger: logger}
}

func (j *SpreadJob) RunOnce(ctx context.Context) error {
	if j.recorder == nil {
		return fmt.Errorf("spread recorder is required")
	}
	records, err := j.recorder.Record(ctx)
	if err != nil {
		return err
	}
	if j.logger != nil && len(records) > 0 {
		j.logger.Printf("spread recorder stored %d records", len(records))
	}
	return nil
}

func (j *SpreadJob) Run(ctx context.Context) error {
	if err := j.RunOnce(ctx); err != nil {
		return err
	}
	t := time.NewTicker(j.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := j.RunOnce(ctx); err != nil {
				return err
			}
		}
	}
}

func waitOrCancel(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

var (
	_ streamIngestion = (*market.IngestionService)(nil)
)
