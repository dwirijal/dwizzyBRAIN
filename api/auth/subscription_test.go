package authapi

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeSubscriptionChecker struct {
	active map[string]bool
}

func (f fakeSubscriptionChecker) HasActiveSubscription(ctx context.Context, walletAddress string) (bool, error) {
	if f.active == nil {
		return false, nil
	}
	return f.active[strings.ToLower(walletAddress)], nil
}

func TestSubscriptionResolverResolvePlan(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New() returned error: %v", err)
	}
	defer pool.Close()

	if err := ensureAuthTables(ctx, pool); err != nil {
		t.Fatalf("ensureAuthTables() returned error: %v", err)
	}

	userWithWallet := uuid.NewString()
	wallet := testWalletAddress()
	checker := fakeSubscriptionChecker{
		active: map[string]bool{
			wallet: true,
		},
	}
	resolver := NewSubscriptionResolver(pool, checker, time.Minute)

	userWithWalletName := "wallet_user_" + strings.ToLower(strings.ReplaceAll(userWithWallet, "-", ""))[:8]
	if _, err := pool.Exec(ctx, `
INSERT INTO users (id, email, name, picture, username, display_name, timezone, locale, plan_override, created_at, updated_at)
VALUES ($1, $2, $3, NULL, $4, $5, 'UTC', 'id-ID', NULL, NOW(), NOW())`,
		userWithWallet, userWithWalletName+"@example.com", "Wallet User", userWithWalletName, "Wallet User",
	); err != nil {
		t.Fatalf("insert userWithWallet: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO auth_identities (id, user_id, provider, provider_user_id, metadata_json, created_at, updated_at)
VALUES ($1, $2, 'evm', $3, '{}'::jsonb, NOW(), NOW())`,
		uuid.NewString(), userWithWallet, wallet,
	); err != nil {
		t.Fatalf("insert wallet identity: %v", err)
	}

	userOverride := uuid.NewString()
	userOverrideName := "override_user_" + strings.ToLower(strings.ReplaceAll(userOverride, "-", ""))[:8]
	if _, err := pool.Exec(ctx, `
INSERT INTO users (id, email, name, picture, username, display_name, timezone, locale, plan_override, created_at, updated_at)
VALUES ($1, $2, $3, NULL, $4, $5, 'UTC', 'id-ID', 'premium', NOW(), NOW())`,
		userOverride, userOverrideName+"@example.com", "Override User", userOverrideName, "Override User",
	); err != nil {
		t.Fatalf("insert userOverride: %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from auth_identities where user_id = $1", userWithWallet)
		_, _ = pool.Exec(context.Background(), "delete from auth_identities where user_id = $1", userOverride)
		_, _ = pool.Exec(context.Background(), "delete from users where id = $1", userWithWallet)
		_, _ = pool.Exec(context.Background(), "delete from users where id = $1", userOverride)
	})

	plan, err := resolver.ResolvePlan(ctx, userWithWallet, "free")
	if err != nil {
		t.Fatalf("ResolvePlan(wallet) returned error: %v", err)
	}
	if plan != "premium" {
		t.Fatalf("expected premium for active wallet, got %q", plan)
	}

	plan, err = resolver.ResolvePlan(ctx, userOverride, "free")
	if err != nil {
		t.Fatalf("ResolvePlan(override) returned error: %v", err)
	}
	if plan != "premium" {
		t.Fatalf("expected premium from override, got %q", plan)
	}
}

func TestSubscriptionResolverResolveEntitlement(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New() returned error: %v", err)
	}
	defer pool.Close()

	if err := ensureAuthTables(ctx, pool); err != nil {
		t.Fatalf("ensureAuthTables() returned error: %v", err)
	}

	userID := uuid.NewString()
	userName := "entitlement_" + strings.ToLower(strings.ReplaceAll(userID, "-", ""))[:8]
	wallet := testWalletAddress()
	if _, err := pool.Exec(ctx, `
INSERT INTO users (id, email, name, picture, username, display_name, timezone, locale, plan_override, created_at, updated_at)
VALUES ($1, $2, $3, NULL, $4, $5, 'UTC', 'id-ID', 'free', NOW(), NOW())`,
		userID, userName+"@example.com", "Entitlement User", userName, "Entitlement User",
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO auth_identities (id, user_id, provider, provider_user_id, metadata_json, created_at, updated_at)
VALUES ($1, $2, 'evm', $3, '{}'::jsonb, NOW(), NOW())`,
		uuid.NewString(), userID, wallet,
	); err != nil {
		t.Fatalf("insert wallet identity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from auth_identities where user_id = $1", userID)
		_, _ = pool.Exec(context.Background(), "delete from users where id = $1", userID)
	})

	checker := fakeSubscriptionChecker{
		active: map[string]bool{
			wallet: true,
		},
	}
	resolver := NewSubscriptionResolver(pool, checker, time.Minute)

	entitlement, err := resolver.ResolveEntitlement(ctx, userID, "free")
	if err != nil {
		t.Fatalf("ResolveEntitlement() returned error: %v", err)
	}
	if entitlement.Plan != "premium" || entitlement.Source != "onchain" || entitlement.Cached {
		t.Fatalf("unexpected entitlement: %+v", entitlement)
	}

	cached, err := resolver.ResolveEntitlement(ctx, userID, "free")
	if err != nil {
		t.Fatalf("ResolveEntitlement(cached) returned error: %v", err)
	}
	if !cached.Cached || cached.Source != "onchain" || cached.Plan != "premium" {
		t.Fatalf("unexpected cached entitlement: %+v", cached)
	}
}

func TestSubscriptionResolverFallsBackOnCheckerError(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("pgxpool.New() returned error: %v", err)
	}
	defer pool.Close()

	if err := ensureAuthTables(ctx, pool); err != nil {
		t.Fatalf("ensureAuthTables() returned error: %v", err)
	}

	userID := uuid.NewString()
	userName := "fallback_" + strings.ToLower(strings.ReplaceAll(userID, "-", ""))[:8]
	wallet := testWalletAddress()
	if _, err := pool.Exec(ctx, `
INSERT INTO users (id, email, name, picture, username, display_name, timezone, locale, plan_override, created_at, updated_at)
VALUES ($1, $2, $3, NULL, $4, $5, 'UTC', 'id-ID', 'free', NOW(), NOW())`,
		userID, userName+"@example.com", "Fallback User", userName, "Fallback User",
	); err != nil {
		t.Fatalf("insert user: %v", err)
	}
	if _, err := pool.Exec(ctx, `
INSERT INTO auth_identities (id, user_id, provider, provider_user_id, metadata_json, created_at, updated_at)
VALUES ($1, $2, 'evm', $3, '{}'::jsonb, NOW(), NOW())`,
		uuid.NewString(), userID, wallet,
	); err != nil {
		t.Fatalf("insert wallet identity: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from auth_identities where user_id = $1", userID)
		_, _ = pool.Exec(context.Background(), "delete from users where id = $1", userID)
	})

	resolver := NewSubscriptionResolver(pool, failingSubscriptionChecker{}, time.Minute)
	plan, err := resolver.ResolvePlan(ctx, userID, "free")
	if plan != "free" {
		t.Fatalf("expected free fallback, got %q", plan)
	}
	_ = err
}

func TestParseSubscriptionNetworks(t *testing.T) {
	networks, err := parseSubscriptionNetworks("base|8453|https://mainnet.base.org|0x0000000000000000000000000000000000000001;bnb|56|https://bsc-dataseed.binance.org|0x0000000000000000000000000000000000000002|checkSubscribed")
	if err != nil {
		t.Fatalf("parseSubscriptionNetworks() returned error: %v", err)
	}
	if len(networks) != 2 {
		t.Fatalf("expected 2 networks, got %d", len(networks))
	}
	if networks[0].Name != "base" || networks[0].ChainID != 8453 || networks[0].Method != defaultSubscriptionMethod {
		t.Fatalf("unexpected first network: %+v", networks[0])
	}
	if networks[1].Name != "bnb" || networks[1].ChainID != 56 || networks[1].Method != "checkSubscribed" {
		t.Fatalf("unexpected second network: %+v", networks[1])
	}
}

func TestEthSubscriptionCheckerSourceLabel(t *testing.T) {
	checker := &EthSubscriptionChecker{name: "base", chainID: 8453}
	if got := checker.sourceLabel(); got != "onchain:base" {
		t.Fatalf("expected onchain:base, got %q", got)
	}
	checker = &EthSubscriptionChecker{chainID: 56}
	if got := checker.sourceLabel(); got != "onchain:56" {
		t.Fatalf("expected onchain:56, got %q", got)
	}
}

func testWalletAddress() string {
	raw := strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", ""))
	return "0x" + raw + "abcd1234"
}

type failingSubscriptionChecker struct{}

func (failingSubscriptionChecker) HasActiveSubscription(ctx context.Context, walletAddress string) (bool, error) {
	return false, errors.New("rpc unavailable")
}
