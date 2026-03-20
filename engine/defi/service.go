package defi

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	defaultProtocolLimit = 500
	defaultHistoryLimit  = 50
	defaultHistoryPoints = 365
	defaultTop50Tier     = "top50"
	defaultTop300Tier    = "top300"
	defaultOtherTier     = "other"
)

type Service struct {
	client               protocolClient
	store                protocolStore
	protocolLimit        int
	protocolHistoryLimit int
	chainHistoryLimit    int
	historyPoints        int
	now                  func() time.Time
}

type Result struct {
	ProtocolsFetched    int
	ProtocolsUpserted   int
	ProtocolsBackfilled int
	ChainsFetched       int
	ChainsUpserted      int
	ChainsBackfilled    int
}

type protocolClient interface {
	Protocols(ctx context.Context) ([]ProtocolListItem, error)
	Chains(ctx context.Context) ([]ChainListItem, error)
	Protocol(ctx context.Context, slug string) (ProtocolDetail, error)
	ChainHistory(ctx context.Context, chain string) ([]ChainTVLPoint, error)
}

type protocolStore interface {
	LookupCoinIDBySymbol(ctx context.Context, symbol string) (string, error)
	UpsertProtocols(ctx context.Context, items []ProtocolUpsert) error
	UpsertProtocolCoverage(ctx context.Context, items []ProtocolCoverage) error
	UpsertChains(ctx context.Context, items []ChainUpsert) error
	UpsertProtocolLatest(ctx context.Context, items []ProtocolLatest) error
	InsertProtocolHistory(ctx context.Context, records []ProtocolHistoryRecord) error
	InsertChainHistory(ctx context.Context, records []ChainHistoryRecord) error
}

func NewService(client protocolClient, store protocolStore, protocolLimit, protocolHistoryLimit, chainHistoryLimit, historyPoints int) *Service {
	if protocolLimit <= 0 {
		protocolLimit = defaultProtocolLimit
	}
	if protocolHistoryLimit <= 0 {
		protocolHistoryLimit = defaultHistoryLimit
	}
	if chainHistoryLimit <= 0 {
		chainHistoryLimit = 15
	}
	if historyPoints <= 0 {
		historyPoints = defaultHistoryPoints
	}
	return &Service{
		client:               client,
		store:                store,
		protocolLimit:        protocolLimit,
		protocolHistoryLimit: protocolHistoryLimit,
		chainHistoryLimit:    chainHistoryLimit,
		historyPoints:        historyPoints,
		now:                  time.Now,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.client == nil {
		return Result{}, fmt.Errorf("defi client is required")
	}
	if s.store == nil {
		return Result{}, fmt.Errorf("defi store is required")
	}

	protocols, err := s.client.Protocols(ctx)
	if err != nil {
		return Result{}, err
	}
	sort.Slice(protocols, func(i, j int) bool {
		if protocols[i].TVL == protocols[j].TVL {
			return protocols[i].Slug < protocols[j].Slug
		}
		return protocols[i].TVL > protocols[j].TVL
	})
	if len(protocols) > s.protocolLimit {
		protocols = protocols[:s.protocolLimit]
	}

	protocolUpserts := make([]ProtocolUpsert, 0, len(protocols))
	coverageUpserts := make([]ProtocolCoverage, 0, len(protocols))
	protocolLatest := make([]ProtocolLatest, 0, len(protocols))
	for i, item := range protocols {
		coinID, err := s.resolveCoinID(ctx, item)
		if err != nil {
			return Result{}, err
		}
		now := s.now().UTC()
		protocolUpserts = append(protocolUpserts, ProtocolUpsert{
			Slug:        item.Slug,
			Name:        item.Name,
			Description: item.Description,
			Logo:        item.Logo,
			Category:    normalizeCategory(item.Category),
			URL:         item.URL,
			Twitter:     item.Twitter,
			CoinID:      coinID,
			AuditStatus: auditStatus(item.Audits),
			Oracles:     oraclesFromChains(item.Chains),
			UpdatedAt:   now,
		})
		coverageUpserts = append(coverageUpserts, ProtocolCoverage{
			Slug:      item.Slug,
			Tier:      tierForIndex(i),
			UpdatedAt: now,
		})
		protocolLatest = append(protocolLatest, ProtocolLatest{
			Slug:      item.Slug,
			TVLUSD:    item.TVL,
			Change1D:  floatPtr(item.Change1D),
			Change7D:  floatPtr(item.Change7D),
			UpdatedAt: now,
		})
	}

	if err := s.store.UpsertProtocols(ctx, protocolUpserts); err != nil {
		return Result{}, err
	}
	if err := s.store.UpsertProtocolCoverage(ctx, coverageUpserts); err != nil {
		return Result{}, err
	}
	if err := s.store.UpsertProtocolLatest(ctx, protocolLatest); err != nil {
		return Result{}, err
	}

	protocolBackfillCount := s.protocolHistoryLimit
	if protocolBackfillCount > len(protocols) {
		protocolBackfillCount = len(protocols)
	}
	backfilled, err := s.backfillProtocols(ctx, protocols[:protocolBackfillCount])
	if err != nil {
		return Result{}, err
	}

	chains, err := s.client.Chains(ctx)
	if err != nil {
		return Result{}, err
	}
	chainUpserts := make([]ChainUpsert, 0, len(chains))
	for _, item := range chains {
		chainUpserts = append(chainUpserts, ChainUpsert{
			Chain:     normalizeChain(item.Chain, item.Name),
			TVLUSD:    item.TVL,
			Change1D:  floatPtr(item.Change1D),
			Change7D:  floatPtr(item.Change7D),
			UpdatedAt: s.now().UTC(),
		})
	}
	if err := s.store.UpsertChains(ctx, chainUpserts); err != nil {
		return Result{}, err
	}

	chainBackfillCount := s.chainHistoryLimit
	if chainBackfillCount > len(chains) {
		chainBackfillCount = len(chains)
	}
	chainBackfilled, err := s.backfillChains(ctx, chains[:chainBackfillCount])
	if err != nil {
		return Result{}, err
	}

	return Result{
		ProtocolsFetched:    len(protocols),
		ProtocolsUpserted:   len(protocolUpserts),
		ProtocolsBackfilled: backfilled,
		ChainsFetched:       len(chains),
		ChainsUpserted:      len(chainUpserts),
		ChainsBackfilled:    chainBackfilled,
	}, nil
}

func (s *Service) resolveCoinID(ctx context.Context, item ProtocolListItem) (string, error) {
	if s.store == nil {
		return "", fmt.Errorf("defi store is required")
	}

	if coinID, err := s.store.LookupCoinIDBySymbol(ctx, item.Symbol); err != nil {
		return "", err
	} else if coinID != "" {
		return coinID, nil
	}

	if coinID, err := s.store.LookupCoinIDBySymbol(ctx, item.Slug); err != nil {
		return "", err
	} else if coinID != "" {
		return coinID, nil
	}

	return "", nil
}

func (s *Service) backfillProtocols(ctx context.Context, items []ProtocolListItem) (int, error) {
	total := 0
	for _, item := range items {
		detail, err := s.client.Protocol(ctx, item.Slug)
		if err != nil {
			return total, fmt.Errorf("fetch protocol %s: %w", item.Slug, err)
		}
		records := buildHistoryRecords(item.Slug, detail, s.historyPoints, s.now())
		if len(records) == 0 {
			continue
		}
		if err := s.store.InsertProtocolHistory(ctx, records); err != nil {
			return total, err
		}
		total++
	}
	return total, nil
}

func (s *Service) backfillChains(ctx context.Context, items []ChainListItem) (int, error) {
	total := 0
	for _, item := range items {
		history, err := s.client.ChainHistory(ctx, item.Chain)
		if err != nil {
			return total, fmt.Errorf("fetch chain history %s: %w", item.Chain, err)
		}
		records := buildChainHistoryRecords(item.Chain, history, s.historyPoints, s.now())
		if len(records) == 0 {
			continue
		}
		if err := s.store.InsertChainHistory(ctx, records); err != nil {
			return total, err
		}
		total++
	}
	return total, nil
}

func buildHistoryRecords(slug string, detail ProtocolDetail, maxPoints int, now time.Time) []ProtocolHistoryRecord {
	records := make([]ProtocolHistoryRecord, 0)
	if len(detail.TVL) > 0 {
		points := trimHistoryPoints(detail.TVL, maxPoints)
		for _, point := range points {
			if point.Date <= 0 {
				continue
			}
			records = append(records, ProtocolHistoryRecord{
				Time:         time.Unix(point.Date, 0).UTC(),
				ProtocolSlug: slug,
				Chain:        "total",
				TVLUSD:       point.TotalLiquidityUSD,
				Metadata: map[string]any{
					"source": "protocol",
					"type":   "total",
					"slug":   slug,
				},
			})
		}
	}

	for chain, history := range detail.ChainTvls {
		points := trimHistoryPoints(history.TVL, maxPoints)
		for _, point := range points {
			if point.Date <= 0 {
				continue
			}
			records = append(records, ProtocolHistoryRecord{
				Time:         time.Unix(point.Date, 0).UTC(),
				ProtocolSlug: slug,
				Chain:        chain,
				TVLUSD:       point.TotalLiquidityUSD,
				Metadata: map[string]any{
					"source": "protocol",
					"type":   "chain",
					"slug":   slug,
					"chain":  chain,
				},
			})
		}
	}

	return records
}

func buildChainHistoryRecords(chain string, points []ChainTVLPoint, maxPoints int, now time.Time) []ChainHistoryRecord {
	if len(points) == 0 {
		return nil
	}
	points = trimChainHistoryPoints(points, maxPoints)
	records := make([]ChainHistoryRecord, 0, len(points))
	for _, point := range points {
		if point.Date <= 0 {
			continue
		}
		records = append(records, ChainHistoryRecord{
			Time:   time.Unix(point.Date, 0).UTC(),
			Chain:  chain,
			TVLUSD: point.TVL,
		})
	}
	return records
}

func trimHistoryPoints(points []TVLPoint, maxPoints int) []TVLPoint {
	if maxPoints <= 0 || len(points) <= maxPoints {
		return points
	}
	start := len(points) - maxPoints
	if start < 0 {
		start = 0
	}
	return points[start:]
}

func trimChainHistoryPoints(points []ChainTVLPoint, maxPoints int) []ChainTVLPoint {
	if maxPoints <= 0 || len(points) <= maxPoints {
		return points
	}
	start := len(points) - maxPoints
	if start < 0 {
		start = 0
	}
	return points[start:]
}

func normalizeCategory(category string) string {
	category = strings.TrimSpace(category)
	if category == "" {
		return "other"
	}
	return category
}

func normalizeChain(chain, fallback string) string {
	chain = strings.TrimSpace(chain)
	if chain != "" {
		return chain
	}
	return strings.TrimSpace(fallback)
}

func auditStatus(audits string) string {
	audits = strings.TrimSpace(audits)
	if audits == "" {
		return "unknown"
	}
	if audits == "0" {
		return "unaudited"
	}
	return "audited"
}

func oraclesFromChains(_ []string) []string {
	// No oracle data exists in the public list endpoint; leave the column empty.
	return []string{}
}

func tierForIndex(index int) string {
	switch {
	case index < 50:
		return defaultTop50Tier
	case index < 300:
		return defaultTop300Tier
	default:
		return defaultOtherTier
	}
}

func floatPtr(value float64) *float64 {
	v := value
	return &v
}
