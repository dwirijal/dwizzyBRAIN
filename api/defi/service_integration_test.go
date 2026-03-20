package defiapi

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"dwizzyBRAIN/engine/storage"
)

func TestServiceLiveDefiRead(t *testing.T) {
	if os.Getenv("POSTGRES_URL") == "" {
		t.Skip("POSTGRES_URL is required for integration test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := storage.NewPostgresPoolFromEnv(ctx)
	if err != nil {
		t.Fatalf("new postgres pool: %v", err)
	}
	defer pool.Close()

	service := NewService(pool)
	slug := fmt.Sprintf("test-defi-%d", time.Now().UnixNano())

	_, err = pool.Exec(ctx, `
INSERT INTO defi_protocols (slug, name, description, logo, category, url, twitter, coin_id, audit_status, oracles)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		slug,
		"Test DeFi Protocol",
		"integration fixture",
		"https://example.com/logo.png",
		"dex",
		"https://example.com",
		"@test",
		"bitcoin",
		"audited",
		[]string{"chainlink"},
	)
	if err != nil {
		t.Fatalf("seed defi protocol: %v", err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM defi_dex_latest WHERE slug = $1`, slug)
		_, _ = pool.Exec(context.Background(), `DELETE FROM defi_chain_tvl_latest WHERE chain = $1`, "test-chain")
		_, _ = pool.Exec(context.Background(), `DELETE FROM defi_protocol_tvl_latest WHERE slug = $1`, slug)
		_, _ = pool.Exec(context.Background(), `DELETE FROM defi_protocol_coverage WHERE slug = $1`, slug)
		_, _ = pool.Exec(context.Background(), `DELETE FROM defi_protocols WHERE slug = $1`, slug)
	}()

	_, err = pool.Exec(ctx, `
INSERT INTO defi_protocol_tvl_latest (slug, tvl, change_1d, change_7d)
VALUES ($1, $2, $3, $4)
ON CONFLICT (slug) DO UPDATE SET tvl = EXCLUDED.tvl, change_1d = EXCLUDED.change_1d, change_7d = EXCLUDED.change_7d, updated_at = NOW()`,
		slug,
		1234567.89,
		1.23,
		4.56,
	)
	if err != nil {
		t.Fatalf("seed defi tvl: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO defi_protocol_coverage (slug, tier)
VALUES ($1, $2)
ON CONFLICT (slug) DO UPDATE SET tier = EXCLUDED.tier, updated_at = NOW()`,
		slug,
		"top50",
	)
	if err != nil {
		t.Fatalf("seed defi coverage: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO defi_chain_tvl_latest (chain, tvl, change_1d, change_7d)
VALUES ($1, $2, $3, $4)
ON CONFLICT (chain) DO UPDATE SET tvl = EXCLUDED.tvl, change_1d = EXCLUDED.change_1d, change_7d = EXCLUDED.change_7d, updated_at = NOW()`,
		"test-chain",
		9876543.21,
		-1.11,
		2.22,
	)
	if err != nil {
		t.Fatalf("seed defi chain: %v", err)
	}

	_, err = pool.Exec(ctx, `
INSERT INTO defi_dex_latest (slug, name, volume_24h, volume_7d, volume_30d, change_1d)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name, volume_24h = EXCLUDED.volume_24h, volume_7d = EXCLUDED.volume_7d, volume_30d = EXCLUDED.volume_30d, change_1d = EXCLUDED.change_1d, updated_at = NOW()`,
		slug,
		"Test DEX",
		654321.00,
		7654321.00,
		87654321.00,
		0.42,
	)
	if err != nil {
		t.Fatalf("seed defi dex: %v", err)
	}

	protocols, err := service.ListProtocols(ctx, 10, 0, "")
	if err != nil {
		t.Fatalf("list protocols: %v", err)
	}
	if protocols.Total <= 0 {
		t.Fatalf("expected positive protocol total")
	}

	protocol, err := service.Protocol(ctx, slug)
	if err != nil {
		t.Fatalf("protocol detail: %v", err)
	}
	if protocol.Slug != slug {
		t.Fatalf("expected protocol slug %s, got %s", slug, protocol.Slug)
	}

	chains, err := service.ListChains(ctx, 10)
	if err != nil {
		t.Fatalf("list chains: %v", err)
	}
	if len(chains) == 0 {
		t.Fatalf("expected chains")
	}

	dexes, err := service.ListDexes(ctx, 10)
	if err != nil {
		t.Fatalf("list dexes: %v", err)
	}
	if len(dexes) == 0 {
		t.Fatalf("expected dexes")
	}

	overview, err := service.Overview(ctx, 5)
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if len(overview.Protocols) == 0 || len(overview.Chains) == 0 || len(overview.Dexes) == 0 {
		t.Fatalf("expected overview data")
	}
}
