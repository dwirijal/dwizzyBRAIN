package authapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDiscordConfigValidate(t *testing.T) {
	t.Parallel()

	valid := DiscordConfig{
		ClientID:     "123456789012345678",
		ClientSecret: "secret",
		RedirectURI:  "https://example.com/v1/auth/discord/callback",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid config should pass: %v", err)
	}

	invalid := valid
	invalid.ClientID = "abc"
	if err := invalid.Validate(); err == nil {
		t.Fatal("expected numeric client id validation to fail")
	}
}

func TestJWTSignAndVerify(t *testing.T) {
	t.Parallel()

	secret := []byte("test-secret")
	claims := authClaims{
		Sub:   "user-1",
		Sid:   "session-1",
		Plan:  "free",
		Roles: []string{"user"},
		Iss:   defaultIssuer,
		Aud:   defaultAudience,
		Iat:   time.Now().Unix(),
		Exp:   time.Now().Add(15 * time.Minute).Unix(),
	}

	token, err := signJWT(secret, claims)
	if err != nil {
		t.Fatalf("signJWT() returned error: %v", err)
	}

	parsed, err := verifyJWT(secret, token, defaultIssuer, defaultAudience)
	if err != nil {
		t.Fatalf("verifyJWT() returned error: %v", err)
	}
	if parsed.Sub != claims.Sub || parsed.Sid != claims.Sid {
		t.Fatalf("unexpected parsed claims: %#v", parsed)
	}
}

func TestServiceDiscordAuthFlowIntegration(t *testing.T) {
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

	profile := discordProfile{
		ID:         uuid.NewString(),
		Username:   "dwizzy",
		GlobalName: "D Wizzy",
		Avatar:     "avatar-hash",
		Locale:     "id-ID",
		Email:      strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", "")) + "@example.com",
	}

	discordServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if r.Method != http.MethodPost {
				t.Fatalf("expected POST token exchange, got %s", r.Method)
			}
			_ = json.NewEncoder(w).Encode(discordTokenResponse{
				AccessToken:  "discord-access-token",
				TokenType:    "Bearer",
				ExpiresIn:    3600,
				RefreshToken: "discord-refresh-token",
				Scope:        "identify email",
			})
		case "/me":
			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer discord-access-token" {
				t.Fatalf("unexpected auth header: %q", authHeader)
			}
			_ = json.NewEncoder(w).Encode(profile)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer discordServer.Close()

	cfg := Config{
		Discord: DiscordConfig{
			ClientID:     "123456789012345678",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/v1/auth/discord/callback",
			AuthorizeURL: discordServer.URL + "/authorize",
			TokenURL:     discordServer.URL + "/token",
			ProfileURL:   discordServer.URL + "/me",
		},
		DiscordConfigured: true,
		JWTSecret:         "super-secret-for-tests",
	}
	service := NewService(pool, cfg)
	service.httpClient = discordServer.Client()
	fixedNow := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixedNow }

	start, err := service.StartDiscordAuth(ctx)
	if err != nil {
		t.Fatalf("StartDiscordAuth() returned error: %v", err)
	}
	if !strings.Contains(start.AuthorizationURL, "client_id=123456789012345678") {
		t.Fatalf("authorization url missing client id: %s", start.AuthorizationURL)
	}

	result, err := service.CompleteDiscordAuth(ctx, "code-123", "state-abc", "state-abc", RequestMeta{
		RemoteAddr: "203.0.113.10:12345",
		UserAgent:  "dwizzy-test/1.0",
	})
	if err != nil {
		t.Fatalf("CompleteDiscordAuth() returned error: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected issued tokens, got %+v", result)
	}
	if result.User.ID == "" || result.Session.ID == "" {
		t.Fatalf("expected user/session in response, got %+v", result)
	}

	me, err := service.Me(ctx, result.AccessToken)
	if err != nil {
		t.Fatalf("Me() returned error: %v", err)
	}
	if me.User.ID != result.User.ID {
		t.Fatalf("expected me user %s, got %s", result.User.ID, me.User.ID)
	}

	wallet := "0x" + strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", "")) + "abcd1234"
	if _, err := pool.Exec(ctx, `
INSERT INTO auth_identities (id, user_id, provider, provider_user_id, metadata_json, created_at, updated_at)
VALUES ($1, $2, 'evm', $3, '{}'::jsonb, NOW(), NOW())`,
		uuid.NewString(), result.User.ID, wallet,
	); err != nil {
		t.Fatalf("insert wallet identity: %v", err)
	}
	service.SetPlanResolver(NewSubscriptionResolver(pool, fakeSubscriptionChecker{
		active: map[string]bool{
			wallet: true,
		},
	}, time.Minute))

	entitlement, err := service.Entitlement(ctx, result.AccessToken)
	if err != nil {
		t.Fatalf("Entitlement() returned error: %v", err)
	}
	if entitlement.User.ID != result.User.ID {
		t.Fatalf("expected entitlement user %s, got %s", result.User.ID, entitlement.User.ID)
	}
	if entitlement.Plan != "premium" {
		t.Fatalf("expected premium entitlement, got %q", entitlement.Plan)
	}
	if entitlement.Source != "onchain" {
		t.Fatalf("expected onchain entitlement source, got %q", entitlement.Source)
	}
	if entitlement.WalletsFound != 1 {
		t.Fatalf("expected 1 wallet, got %d", entitlement.WalletsFound)
	}

	rotated, err := service.Refresh(ctx, result.RefreshToken, RequestMeta{
		RemoteAddr: "203.0.113.10:12345",
		UserAgent:  "dwizzy-test/1.0",
	})
	if err != nil {
		t.Fatalf("Refresh() returned error: %v", err)
	}
	if rotated.RefreshToken == result.RefreshToken {
		t.Fatal("expected refresh token rotation")
	}

	if _, err := service.Refresh(ctx, result.RefreshToken, RequestMeta{}); !errors.Is(err, ErrTokenReuse) {
		t.Fatalf("expected ErrTokenReuse on reused token, got %v", err)
	}

	if err := service.Logout(ctx, rotated.RefreshToken); err != nil {
		t.Fatalf("Logout() returned error: %v", err)
	}
	if _, err := service.Me(ctx, rotated.AccessToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized after logout, got %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from users where id = $1", result.User.ID)
	})
}

func TestServiceWeb3AuthFlowIntegration(t *testing.T) {
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

	privKey, err := ethcrypto.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey() returned error: %v", err)
	}
	walletAddress := strings.ToLower(ethcrypto.PubkeyToAddress(privKey.PublicKey).Hex())

	cfg := Config{
		DiscordConfigured: false,
		JWTSecret:         "super-secret-for-tests",
	}
	service := NewService(pool, cfg)
	fixedNow := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return fixedNow }

	nonceResp, err := service.RequestWeb3Nonce(ctx, walletAddress, "")
	if err != nil {
		t.Fatalf("RequestWeb3Nonce() returned error: %v", err)
	}
	if nonceResp.WalletAddress != walletAddress {
		t.Fatalf("expected wallet %s, got %s", walletAddress, nonceResp.WalletAddress)
	}

	sig, err := ethcrypto.Sign(accounts.TextHash([]byte(nonceResp.Challenge)), privKey)
	if err != nil {
		t.Fatalf("Sign() returned error: %v", err)
	}

	result, err := service.VerifyWeb3Signature(ctx, walletAddress, "", nonceResp.Nonce, "0x"+common.Bytes2Hex(sig), RequestMeta{
		RemoteAddr: "203.0.113.11:23456",
		UserAgent:  "dwizzy-wallet-test/1.0",
	})
	if err != nil {
		t.Fatalf("VerifyWeb3Signature() returned error: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatalf("expected issued tokens, got %+v", result)
	}

	me, err := service.Me(ctx, result.AccessToken)
	if err != nil {
		t.Fatalf("Me() returned error: %v", err)
	}
	if me.User.ID != result.User.ID {
		t.Fatalf("expected me user %s, got %s", result.User.ID, me.User.ID)
	}

	if _, err := service.VerifyWeb3Signature(ctx, walletAddress, "", nonceResp.Nonce, "0x"+common.Bytes2Hex(sig), RequestMeta{}); !errors.Is(err, ErrNonceConsumed) {
		t.Fatalf("expected ErrNonceConsumed on reused nonce, got %v", err)
	}

	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "delete from users where id = $1", result.User.ID)
	})
}

func ensureAuthTables(ctx context.Context, pool *pgxpool.Pool) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id UUID PRIMARY KEY,
			username TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			avatar_url TEXT,
			timezone TEXT NOT NULL DEFAULT 'UTC',
			locale TEXT NOT NULL DEFAULT 'id-ID',
			plan_override TEXT,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS auth_identities (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			provider TEXT NOT NULL,
			provider_user_id TEXT NOT NULL,
			metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (provider, provider_user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			id UUID PRIMARY KEY,
			user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status TEXT NOT NULL DEFAULT 'active',
			session_family_id UUID NOT NULL,
			ip_hash TEXT NOT NULL DEFAULT '',
			user_agent_hash TEXT NOT NULL DEFAULT '',
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			revoked_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS auth_refresh_tokens (
			id UUID PRIMARY KEY,
			session_id UUID NOT NULL REFERENCES auth_sessions(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			rotated_from_token_id UUID REFERENCES auth_refresh_tokens(id) ON DELETE SET NULL,
			consumed_at TIMESTAMPTZ,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS auth_nonces (
			id UUID PRIMARY KEY,
			wallet_address TEXT NOT NULL,
			nonce TEXT NOT NULL,
			purpose TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			used_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, stmt := range statements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
