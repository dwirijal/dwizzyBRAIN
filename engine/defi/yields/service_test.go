package yields

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	lookups []string
	latest  []LatestRecord
	history []HistoryRecord
}

func (f *fakeStore) LookupProtocolSlugByProject(ctx context.Context, project string) (string, error) {
	f.lookups = append(f.lookups, project)
	if project == "lido" {
		return "lido", nil
	}
	return "", nil
}

func (f *fakeStore) UpsertLatest(ctx context.Context, items []LatestRecord) error {
	f.latest = append(f.latest, items...)
	return nil
}

func (f *fakeStore) InsertHistory(ctx context.Context, records []HistoryRecord) error {
	f.history = append(f.history, records...)
	return nil
}

type fakeClient struct {
	pools []PoolSnapshot
	chart map[string][]ChartPoint
}

func (f *fakeClient) Pools(ctx context.Context) ([]PoolSnapshot, error) { return f.pools, nil }
func (f *fakeClient) PoolChart(ctx context.Context, pool string) ([]ChartPoint, error) {
	return f.chart[pool], nil
}

func TestServiceRunOnce(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	client := &fakeClient{
		pools: []PoolSnapshot{
			{Chain: "Ethereum", Project: "lido", Symbol: "STETH", Pool: "pool-1", TVLUSD: 123.45, APY: ptr(2.3)},
			{Chain: "Ethereum", Project: "aave-v3", Symbol: "USDC", Pool: "pool-2", TVLUSD: 100.00, APY: ptr(1.5)},
		},
		chart: map[string][]ChartPoint{
			"pool-1": {
				{Timestamp: now, TVLUSD: 123.45, APY: ptr(2.3), APYBase: ptr(2.1), APYReward: ptr(0.2)},
			},
		},
	}
	store := &fakeStore{}
	svc := NewService(client, store, 10, 1, 10)
	svc.now = func() time.Time { return now }

	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if result.PoolsFetched != 2 || result.PoolsUpserted != 2 || result.PoolsBackfilled != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if len(store.latest) != 2 || len(store.history) != 1 {
		t.Fatalf("unexpected storage counts: latest=%d history=%d", len(store.latest), len(store.history))
	}
	if store.latest[0].ProtocolSlug != "lido" {
		t.Fatalf("expected resolved protocol slug, got %#v", store.latest[0])
	}
}

func ptr(v float64) *float64 { return &v }
