package authapi

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type fakeScanRow struct {
	scanFn func(dest ...any) error
}

func (f fakeScanRow) Scan(dest ...any) error {
	return f.scanFn(dest...)
}

func TestConfigFromEnv(t *testing.T) {
	t.Setenv("DISCORD_CLIENT_ID", "")
	t.Setenv("DISCORD_CLIENT_SECRET", "")
	t.Setenv("DISCORD_REDIRECT_URI", "")
	t.Setenv("JWT_SECRET", "")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("expected error when JWT_SECRET is missing")
	}

	t.Setenv("DISCORD_CLIENT_ID", "123456789012345678")
	t.Setenv("DISCORD_CLIENT_SECRET", "")
	t.Setenv("DISCORD_REDIRECT_URI", "")
	t.Setenv("JWT_SECRET", "secret")
	if _, err := ConfigFromEnv(); err == nil {
		t.Fatal("expected discord config validation error")
	}

	t.Setenv("DISCORD_CLIENT_ID", "123456789012345678")
	t.Setenv("DISCORD_CLIENT_SECRET", "secret")
	t.Setenv("DISCORD_REDIRECT_URI", "https://example.com/callback")
	t.Setenv("JWT_SECRET", "secret")
	cfg, err := ConfigFromEnv()
	if err != nil {
		t.Fatalf("ConfigFromEnv error: %v", err)
	}
	if !cfg.DiscordConfigured {
		t.Fatal("expected discord configured")
	}
}

func TestServiceFlagsAndStartDiscordAuth(t *testing.T) {
	t.Parallel()

	svc := NewService(nil, Config{
		Discord: DiscordConfig{
			ClientID:     "123456789012345678",
			ClientSecret: "secret",
			RedirectURI:  "https://example.com/callback",
			AuthorizeURL: "https://discord.com/oauth2/authorize",
		},
		DiscordConfigured: true,
		JWTSecret:         "jwt",
	})
	if svc.Enabled() {
		t.Fatal("expected service with nil db to be disabled")
	}
	if svc.DiscordEnabled() {
		t.Fatal("expected discord to be disabled when service disabled")
	}
	if !svc.CookieSecure() {
		t.Fatal("expected secure cookie for https redirect URI")
	}

	enabled := &Service{
		db:                &pgxpool.Pool{},
		discord:           svc.discord,
		discordConfigured: true,
		jwtSecret:         []byte("jwt"),
		now:               func() time.Time { return time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC) },
		stateTTL:          10 * time.Minute,
	}
	if !enabled.Enabled() {
		t.Fatal("expected enabled service")
	}
	if !enabled.DiscordEnabled() {
		t.Fatal("expected discord enabled")
	}

	start, err := enabled.StartDiscordAuth(context.Background())
	if err != nil {
		t.Fatalf("StartDiscordAuth error: %v", err)
	}
	if start.State == "" {
		t.Fatal("expected non-empty state")
	}
	if !strings.Contains(start.AuthorizationURL, "client_id=123456789012345678") {
		t.Fatalf("unexpected authorization URL: %s", start.AuthorizationURL)
	}

	disabled := &Service{}
	if _, err := disabled.StartDiscordAuth(context.Background()); !errors.Is(err, ErrMissingConfig) {
		t.Fatalf("expected ErrMissingConfig, got %v", err)
	}
}

func TestServiceHelperFunctions(t *testing.T) {
	t.Parallel()

	if got := normalizeRemoteAddr("203.0.113.1:12345"); got != "203.0.113.1" {
		t.Fatalf("normalizeRemoteAddr host: %q", got)
	}
	if got := normalizeRemoteAddr("203.0.113.1"); got != "203.0.113.1" {
		t.Fatalf("normalizeRemoteAddr raw: %q", got)
	}

	if got := firstNonEmpty(" ", "", "hello", "world"); got != "hello" {
		t.Fatalf("firstNonEmpty=%q want=hello", got)
	}
	if got := sanitizeUsername("  A*B C___D  "); got != "a_b_c_d" {
		t.Fatalf("sanitizeUsername=%q", got)
	}
	if got := sanitizeUsername(".."); got != "" {
		t.Fatalf("sanitizeUsername short=%q want empty", got)
	}
	if got := usernameCandidate("abcdefghijklmnopqrstuvwxyz", 12); len(got) > 24 {
		t.Fatalf("usernameCandidate too long: %q", got)
	}
	if got := trimUsername("__ab__"); got != "discord_user" {
		t.Fatalf("trimUsername short=%q", got)
	}

	avatar := discordAvatarURL(discordProfile{ID: "123", Avatar: "hash"})
	if avatar == "" || !strings.Contains(avatar, "/123/hash.png") {
		t.Fatalf("discordAvatarURL=%q", avatar)
	}
	if got := discordAvatarURL(discordProfile{}); got != "" {
		t.Fatalf("discordAvatarURL empty=%q", got)
	}

	tokenHash := hashToken("  token ")
	if tokenHash != hashToken("token") {
		t.Fatal("hashToken should trim input")
	}
	if tokenHash == "" {
		t.Fatal("hashToken should not be empty")
	}

	if got := hashWithPepper([]byte("pepper"), "value"); got == "" {
		t.Fatal("hashWithPepper should not be empty")
	}
}

func TestTokenAndSignatureHelpers(t *testing.T) {
	t.Parallel()

	token, err := generateToken(0)
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("decode token error: %v", err)
	}
	if len(raw) != 32 {
		t.Fatalf("generateToken bytes=%d want=32", len(raw))
	}

	if _, _, err := netSplitHostPort("127.0.0.1:8080"); err != nil {
		t.Fatalf("netSplitHostPort valid error: %v", err)
	}

	addr, err := normalizeWalletAddress("0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
	if err != nil {
		t.Fatalf("normalizeWalletAddress error: %v", err)
	}
	if addr != "0x742d35cc6634c0532925a3b844bc454e4438f44e" {
		t.Fatalf("normalizeWalletAddress=%q", addr)
	}
	if _, err := normalizeWalletAddress("invalid"); !errors.Is(err, ErrInvalidWalletAddress) {
		t.Fatalf("expected ErrInvalidWalletAddress, got %v", err)
	}

	if got := normalizeNoncePurpose("custom"); got != "login" {
		t.Fatalf("normalizeNoncePurpose=%q", got)
	}
	if got := walletDisplayName("0x742d35cc6634c0532925a3b844bc454e4438f44e"); !strings.Contains(got, "…") {
		t.Fatalf("walletDisplayName=%q", got)
	}

	if _, err := decodeSignature("0x1234"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
	if _, err := recoverWalletFromSignature("hello", "0x1234"); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("expected ErrInvalidSignature, got %v", err)
	}
}

func TestMetadataAndChallengeHelpers(t *testing.T) {
	t.Parallel()

	challenge := buildNonceChallenge("0xabc", "login", "nonce123", time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC))
	if !strings.Contains(challenge, "Wallet: 0xabc") || !strings.Contains(challenge, "Nonce: nonce123") {
		t.Fatalf("unexpected challenge: %q", challenge)
	}

	meta := RequestMeta{RemoteAddr: "203.0.113.1", UserAgent: "agent"}
	discord := discordMetadataJSON(discordProfile{ID: "1", Username: "u"}, meta)
	if discord["request"] == nil || discord["discord"] == nil {
		t.Fatalf("discord metadata malformed: %#v", discord)
	}

	wallet := walletMetadataJSON("0xabc", meta)
	if wallet["request"] == nil || wallet["wallet"] == nil {
		t.Fatalf("wallet metadata malformed: %#v", wallet)
	}
}

func TestScanAndMappingHelpers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 19, 12, 0, 0, 0, time.UTC)
	revoked := now.Add(2 * time.Hour)
	consumed := now.Add(1 * time.Hour)

	userRow := fakeScanRow{scanFn: func(dest ...any) error {
		*(dest[0].(*string)) = "u1"
		*(dest[1].(*string)) = "user"
		*(dest[2].(*string)) = "User"
		*(dest[3].(*string)) = "avatar"
		*(dest[4].(*string)) = "UTC"
		*(dest[5].(*string)) = "id-ID"
		*(dest[6].(*string)) = "free"
		*(dest[7].(*time.Time)) = now
		*(dest[8].(*time.Time)) = now
		return nil
	}}
	user, err := scanUserRow(userRow)
	if err != nil || user.ID != "u1" {
		t.Fatalf("scanUserRow result=%+v err=%v", user, err)
	}

	sessionRow := fakeScanRow{scanFn: func(dest ...any) error {
		*(dest[0].(*string)) = "s1"
		*(dest[1].(*string)) = "u1"
		*(dest[2].(*string)) = "active"
		*(dest[3].(*string)) = "fam1"
		*(dest[4].(*time.Time)) = now
		*(dest[5].(*time.Time)) = now.Add(24 * time.Hour)
		*(dest[6].(*time.Time)) = now
		*(dest[7].(**time.Time)) = &revoked
		return nil
	}}
	session, err := scanSessionRow(sessionRow)
	if err != nil || session.ID != "s1" {
		t.Fatalf("scanSessionRow result=%+v err=%v", session, err)
	}

	refreshRow := fakeScanRow{scanFn: func(dest ...any) error {
		*(dest[0].(*string)) = "r1"
		*(dest[1].(*string)) = "s1"
		*(dest[2].(*string)) = "hash"
		*(dest[3].(**time.Time)) = &consumed
		*(dest[4].(*time.Time)) = now.Add(24 * time.Hour)
		*(dest[5].(*time.Time)) = now
		return nil
	}}
	refresh, err := scanRefreshRow(refreshRow)
	if err != nil || refresh.ID != "r1" {
		t.Fatalf("scanRefreshRow result=%+v err=%v", refresh, err)
	}

	profile := user.toProfile()
	if profile.Username != "user" {
		t.Fatalf("toProfile mismatch: %+v", profile)
	}
	info := session.toInfo()
	if info.ID != "s1" || info.RevokedAt == nil {
		t.Fatalf("toInfo mismatch: %+v", info)
	}
	if got := session.withLastSeen(now.Add(5 * time.Minute)); !got.LastSeenAt.Equal(now.Add(5*time.Minute)) {
		t.Fatalf("withLastSeen mismatch: %+v", got)
	}
}
