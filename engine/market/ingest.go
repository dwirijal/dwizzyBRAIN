package market

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/shared/schema"
)

type tickerResolver interface {
	ResolveCoinID(ctx context.Context, exchange, rawSymbol string) (mapping.Mapping, error)
}

type resolvedTickerPublisher interface {
	PublishResolvedTicker(ctx context.Context, ticker schema.ResolvedTicker) error
}

type rawTickerBatchReader interface {
	ReadMessage() ([]schema.RawTicker, error)
}

type rawTickerPoller interface {
	PollTicker(ctx context.Context, exchangeID, symbol string) (schema.RawTicker, error)
}

type IngestionService struct {
	resolver  tickerResolver
	publisher resolvedTickerPublisher
}

func NewIngestionService(resolver tickerResolver, publisher resolvedTickerPublisher) *IngestionService {
	return &IngestionService{
		resolver:  resolver,
		publisher: publisher,
	}
}

func (s *IngestionService) ResolveTicker(ctx context.Context, raw schema.RawTicker) (schema.ResolvedTicker, error) {
	if s.resolver == nil {
		return schema.ResolvedTicker{}, fmt.Errorf("ticker resolver is required")
	}
	if err := raw.Validate(); err != nil {
		return schema.ResolvedTicker{}, fmt.Errorf("validate raw ticker: %w", err)
	}

	resolvedMapping, err := s.resolver.ResolveCoinID(ctx, raw.Exchange, raw.Symbol)
	if err != nil {
		return schema.ResolvedTicker{}, fmt.Errorf("resolve ticker %s on %s: %w", raw.Symbol, raw.Exchange, err)
	}

	ticker := schema.ResolvedTicker{
		CoinID:         resolvedMapping.CoinID,
		Symbol:         raw.Symbol,
		Exchange:       raw.Exchange,
		BaseAsset:      resolvedMapping.BaseAsset,
		QuoteAsset:     resolvedMapping.QuoteAsset,
		Price:          raw.Price,
		Bid:            raw.Bid,
		Ask:            raw.Ask,
		Volume:         raw.Volume,
		Timestamp:      raw.Timestamp,
		ResolvedSymbol: resolvedMapping.ExchangeSymbol,
	}

	if err := ticker.Validate(); err != nil {
		return schema.ResolvedTicker{}, fmt.Errorf("validate resolved ticker: %w", err)
	}

	return ticker, nil
}

func (s *IngestionService) ProcessTicker(ctx context.Context, raw schema.RawTicker) (schema.ResolvedTicker, error) {
	ticker, err := s.ResolveTicker(ctx, raw)
	if err != nil {
		return schema.ResolvedTicker{}, err
	}

	if s.publisher != nil {
		if err := s.publisher.PublishResolvedTicker(ctx, ticker); err != nil {
			return schema.ResolvedTicker{}, fmt.Errorf("publish resolved ticker: %w", err)
		}
	}

	return ticker, nil
}

func (s *IngestionService) ProcessBatch(ctx context.Context, raws []schema.RawTicker) ([]schema.ResolvedTicker, error) {
	if len(raws) == 0 {
		return nil, nil
	}

	resolved := make([]schema.ResolvedTicker, 0, len(raws))
	for _, raw := range raws {
		ticker, err := s.ProcessTicker(ctx, raw)
		if err != nil {
			return nil, fmt.Errorf("process batch ticker %s on %s: %w", raw.Symbol, raw.Exchange, err)
		}
		resolved = append(resolved, ticker)
	}

	return resolved, nil
}

func (s *IngestionService) ProcessBatchBestEffort(ctx context.Context, raws []schema.RawTicker) ([]schema.ResolvedTicker, []error) {
	if len(raws) == 0 {
		return nil, nil
	}

	resolved := make([]schema.ResolvedTicker, 0, len(raws))
	errs := make([]error, 0)
	for _, raw := range raws {
		ticker, err := s.ProcessTicker(ctx, raw)
		if err != nil {
			if errors.Is(err, mapping.ErrMappingNotFound) {
				errs = append(errs, err)
				continue
			}
			errs = append(errs, err)
			continue
		}
		resolved = append(resolved, ticker)
	}

	return resolved, errs
}

type WSIngestionRunner struct {
	reader    rawTickerBatchReader
	ingestion *IngestionService
}

func NewWSIngestionRunner(reader rawTickerBatchReader, ingestion *IngestionService) *WSIngestionRunner {
	return &WSIngestionRunner{
		reader:    reader,
		ingestion: ingestion,
	}
}

func (r *WSIngestionRunner) ReadAndProcess(ctx context.Context) ([]schema.ResolvedTicker, error) {
	if r.reader == nil {
		return nil, fmt.Errorf("raw ticker reader is required")
	}
	if r.ingestion == nil {
		return nil, fmt.Errorf("ingestion service is required")
	}

	raws, err := r.reader.ReadMessage()
	if err != nil {
		return nil, fmt.Errorf("read raw ticker batch: %w", err)
	}

	return r.ingestion.ProcessBatch(ctx, raws)
}

type RESTIngestionRunner struct {
	poller    rawTickerPoller
	ingestion *IngestionService
}

func NewRESTIngestionRunner(poller rawTickerPoller, ingestion *IngestionService) *RESTIngestionRunner {
	return &RESTIngestionRunner{
		poller:    poller,
		ingestion: ingestion,
	}
}

func (r *RESTIngestionRunner) PollAndProcess(ctx context.Context, exchangeID, symbol string) (schema.ResolvedTicker, error) {
	if r.poller == nil {
		return schema.ResolvedTicker{}, fmt.Errorf("raw ticker poller is required")
	}
	if r.ingestion == nil {
		return schema.ResolvedTicker{}, fmt.Errorf("ingestion service is required")
	}
	if strings.TrimSpace(exchangeID) == "" {
		return schema.ResolvedTicker{}, fmt.Errorf("exchange id is required")
	}
	if strings.TrimSpace(symbol) == "" {
		return schema.ResolvedTicker{}, fmt.Errorf("symbol is required")
	}

	raw, err := r.poller.PollTicker(ctx, exchangeID, symbol)
	if err != nil {
		return schema.ResolvedTicker{}, fmt.Errorf("poll raw ticker %s on %s: %w", symbol, exchangeID, err)
	}

	return r.ingestion.ProcessTicker(ctx, raw)
}
