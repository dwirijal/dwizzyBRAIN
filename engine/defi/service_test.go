package defi

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	resolved     map[string]string
	protocols    []ProtocolUpsert
	coverage     []ProtocolCoverage
	chains       []ChainUpsert
	latest       []ProtocolLatest
	protocolHist []ProtocolHistoryRecord
	chainHist    []ChainHistoryRecord
}

func (f *fakeStore) LookupCoinIDBySymbol(ctx context.Context, symbol string) (string, error) {
	if f.resolved == nil {
		return "", nil
	}
	return f.resolved[symbol], nil
}

func (f *fakeStore) UpsertProtocols(ctx context.Context, items []ProtocolUpsert) error {
	f.protocols = append(f.protocols, items...)
	return nil
}

func (f *fakeStore) UpsertProtocolCoverage(ctx context.Context, items []ProtocolCoverage) error {
	f.coverage = append(f.coverage, items...)
	return nil
}

func (f *fakeStore) UpsertChains(ctx context.Context, items []ChainUpsert) error {
	f.chains = append(f.chains, items...)
	return nil
}

func (f *fakeStore) UpsertProtocolLatest(ctx context.Context, items []ProtocolLatest) error {
	f.latest = append(f.latest, items...)
	return nil
}

func (f *fakeStore) InsertProtocolHistory(ctx context.Context, records []ProtocolHistoryRecord) error {
	f.protocolHist = append(f.protocolHist, records...)
	return nil
}

func (f *fakeStore) InsertChainHistory(ctx context.Context, records []ChainHistoryRecord) error {
	f.chainHist = append(f.chainHist, records...)
	return nil
}

type fakeClient struct {
	protocols    []ProtocolListItem
	chains       []ChainListItem
	details      map[string]ProtocolDetail
	chainHistory map[string][]ChainTVLPoint
}

func (f *fakeClient) Protocols(ctx context.Context) ([]ProtocolListItem, error) {
	return f.protocols, nil
}
func (f *fakeClient) Chains(ctx context.Context) ([]ChainListItem, error) { return f.chains, nil }
func (f *fakeClient) Protocol(ctx context.Context, slug string) (ProtocolDetail, error) {
	if f.details == nil {
		return ProtocolDetail{}, nil
	}
	return f.details[slug], nil
}
func (f *fakeClient) ChainHistory(ctx context.Context, chain string) ([]ChainTVLPoint, error) {
	if f.chainHistory == nil {
		return nil, nil
	}
	return f.chainHistory[chain], nil
}

func TestServiceRunOnce(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	client := &fakeClient{
		protocols: []ProtocolListItem{
			{Slug: "aave-v3", Name: "Aave V3", Symbol: "AAVE", Category: "Lending", Chains: []string{"Ethereum"}, TVL: 1000, Change1D: 1, Change7D: 2, Audits: "2"},
			{Slug: "lido", Name: "Lido", Symbol: "LDO", Category: "Liquid Staking", Chains: []string{"Ethereum"}, TVL: 900, Change1D: 1, Change7D: 2, Audits: "1"},
		},
		chains: []ChainListItem{
			{Chain: "Ethereum", Name: "Ethereum", TokenSymbol: "ETH", TVL: 1234, Change1D: 3, Change7D: 4},
		},
		details: map[string]ProtocolDetail{
			"aave-v3": {
				Slug: "aave-v3",
				TVL:  []TVLPoint{{Date: now.Unix(), TotalLiquidityUSD: 1000}},
				ChainTvls: map[string]ProtocolChainHistory{
					"Ethereum": {TVL: []TVLPoint{{Date: now.Unix(), TotalLiquidityUSD: 900}}},
				},
			},
			"lido": {
				Slug: "lido",
				TVL:  []TVLPoint{{Date: now.Unix(), TotalLiquidityUSD: 900}},
			},
		},
		chainHistory: map[string][]ChainTVLPoint{
			"Ethereum": {{Date: now.Unix(), TVL: 1234}},
		},
	}
	store := &fakeStore{resolved: map[string]string{"AAVE": "aave", "LDO": "lido"}}
	svc := NewService(client, store, 10, 1, 1, 10)
	svc.now = func() time.Time { return now }

	result, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("run once: %v", err)
	}
	if result.ProtocolsFetched != 2 || result.ProtocolsUpserted != 2 {
		t.Fatalf("unexpected protocol result: %#v", result)
	}
	if result.ProtocolsBackfilled != 1 {
		t.Fatalf("unexpected protocol backfill count: %#v", result)
	}
	if result.ChainsFetched != 1 || result.ChainsUpserted != 1 {
		t.Fatalf("unexpected chain result: %#v", result)
	}
	if result.ChainsBackfilled != 1 {
		t.Fatalf("unexpected chain backfill count: %#v", result)
	}
	if len(store.protocolHist) == 0 || len(store.chainHist) == 0 {
		t.Fatalf("expected history inserts, got protocol=%d chain=%d", len(store.protocolHist), len(store.chainHist))
	}
	if got := store.protocols[0].CoinID; got != "aave" {
		t.Fatalf("expected resolved coin id, got %q", got)
	}
	if got := store.latest[0].TVLUSD; got != 1000 {
		t.Fatalf("unexpected latest TVL: %#v", store.latest[0])
	}
}
