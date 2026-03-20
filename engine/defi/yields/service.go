package yields

import (
	"context"
	"fmt"
	"sort"
	"time"
)

const (
	defaultPoolLimit     = 100
	defaultHistoryLimit  = 30
	defaultHistoryPoints = 365
)

type client interface {
	Pools(ctx context.Context) ([]PoolSnapshot, error)
	PoolChart(ctx context.Context, pool string) ([]ChartPoint, error)
}

type store interface {
	LookupProtocolSlugByProject(ctx context.Context, project string) (string, error)
	UpsertLatest(ctx context.Context, items []LatestRecord) error
	InsertHistory(ctx context.Context, records []HistoryRecord) error
}

type Service struct {
	client        client
	store         store
	poolLimit     int
	historyLimit  int
	historyPoints int
	now           func() time.Time
}

func NewService(client client, store store, poolLimit, historyLimit, historyPoints int) *Service {
	if poolLimit <= 0 {
		poolLimit = defaultPoolLimit
	}
	if historyLimit <= 0 {
		historyLimit = defaultHistoryLimit
	}
	if historyPoints <= 0 {
		historyPoints = defaultHistoryPoints
	}
	return &Service{
		client:        client,
		store:         store,
		poolLimit:     poolLimit,
		historyLimit:  historyLimit,
		historyPoints: historyPoints,
		now:           time.Now,
	}
}

func (s *Service) RunOnce(ctx context.Context) (Result, error) {
	if s.client == nil {
		return Result{}, fmt.Errorf("yields client is required")
	}
	if s.store == nil {
		return Result{}, fmt.Errorf("yields store is required")
	}

	pools, err := s.client.Pools(ctx)
	if err != nil {
		return Result{}, err
	}
	sort.Slice(pools, func(i, j int) bool {
		if pools[i].TVLUSD == pools[j].TVLUSD {
			return pools[i].Pool < pools[j].Pool
		}
		return pools[i].TVLUSD > pools[j].TVLUSD
	})
	if len(pools) > s.poolLimit {
		pools = pools[:s.poolLimit]
	}

	now := s.now().UTC()
	latest := make([]LatestRecord, 0, len(pools))
	for _, item := range pools {
		protocolSlug, err := s.store.LookupProtocolSlugByProject(ctx, item.Project)
		if err != nil {
			return Result{}, err
		}
		latest = append(latest, LatestRecord{
			Pool:             item.Pool,
			Chain:            item.Chain,
			Project:          item.Project,
			Symbol:           item.Symbol,
			ProtocolSlug:     protocolSlug,
			TVLUSD:           item.TVLUSD,
			APY:              item.APY,
			APYBase:          item.APYBase,
			APYReward:        item.APYReward,
			APYPct1D:         item.APYPct1D,
			APYPct7D:         item.APYPct7D,
			APYPct30D:        item.APYPct30D,
			APYMean30D:       item.APYMean30D,
			VolumeUsd1D:      item.VolumeUsd1D,
			VolumeUsd7D:      item.VolumeUsd7D,
			Stablecoin:       item.Stablecoin,
			ILRisk:           item.ILRisk,
			Exposure:         item.Exposure,
			RewardTokens:     item.RewardTokens,
			UnderlyingTokens: item.UnderlyingTokens,
			Predictions:      item.Predictions,
			PoolMeta:         toMap(item.PoolMeta),
			Outlier:          item.Outlier,
			Count:            item.Count,
			UpdatedAt:        now,
			SyncedAt:         now,
		})
	}

	if err := s.store.UpsertLatest(ctx, latest); err != nil {
		return Result{}, err
	}

	backfillCount := s.historyLimit
	if backfillCount > len(pools) {
		backfillCount = len(pools)
	}
	backfilled, err := s.backfillHistory(ctx, pools[:backfillCount])
	if err != nil {
		return Result{}, err
	}

	return Result{
		PoolsFetched:    len(pools),
		PoolsUpserted:   len(latest),
		PoolsBackfilled: backfilled,
	}, nil
}

func (s *Service) backfillHistory(ctx context.Context, pools []PoolSnapshot) (int, error) {
	total := 0
	for _, item := range pools {
		points, err := s.client.PoolChart(ctx, item.Pool)
		if err != nil {
			return total, fmt.Errorf("fetch pool chart %s: %w", item.Pool, err)
		}
		records := buildHistoryRecords(item, points, s.historyPoints, s.now())
		if len(records) == 0 {
			continue
		}
		if err := s.store.InsertHistory(ctx, records); err != nil {
			return total, err
		}
		total++
	}
	return total, nil
}

func buildHistoryRecords(item PoolSnapshot, points []ChartPoint, maxPoints int, now time.Time) []HistoryRecord {
	if len(points) == 0 {
		return nil
	}
	points = trimChartPoints(points, maxPoints)
	records := make([]HistoryRecord, 0, len(points))
	for _, point := range points {
		if point.Timestamp.IsZero() {
			continue
		}
		records = append(records, HistoryRecord{
			Time:      point.Timestamp.UTC(),
			Pool:      item.Pool,
			Chain:     item.Chain,
			Project:   item.Project,
			Symbol:    item.Symbol,
			TVLUSD:    point.TVLUSD,
			APY:       point.APY,
			APYBase:   point.APYBase,
			APYReward: point.APYReward,
			Metadata: map[string]any{
				"apyBase7d": point.APYBase7D,
				"il7d":      point.IL7D,
				"synced_at": now.UTC().Format(time.RFC3339),
			},
		})
	}
	return records
}

func trimChartPoints(points []ChartPoint, maxPoints int) []ChartPoint {
	if maxPoints <= 0 || len(points) <= maxPoints {
		return points
	}
	start := len(points) - maxPoints
	if start < 0 {
		start = 0
	}
	return points[start:]
}

func toMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{"value": value}
}
