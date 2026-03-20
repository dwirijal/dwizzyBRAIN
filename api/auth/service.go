package authapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	sharedconfig "dwizzyBRAIN/shared/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrMissingConfig        = errors.New("auth is not configured")
	ErrInvalidState         = errors.New("invalid oauth state")
	ErrInvalidToken         = errors.New("invalid auth token")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrTokenReuse           = errors.New("refresh token reuse detected")
	ErrTokenExpired         = errors.New("refresh token expired")
	ErrSessionRevoked       = errors.New("session revoked")
	ErrInvalidNonce         = errors.New("invalid nonce")
	ErrNonceExpired         = errors.New("nonce expired")
	ErrNonceConsumed        = errors.New("nonce already used")
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrUnsupportedPurpose   = errors.New("unsupported nonce purpose")
	ErrInvalidWalletAddress = errors.New("invalid wallet address")
)

const (
	stateCookieName   = "dwizzy_discord_oauth_state"
	accessCookieName  = "dwizzy_access_token"
	refreshCookieName = "dwizzy_refresh_token"

	defaultAccessTTL    = 15 * time.Minute
	defaultRefreshTTL   = 30 * 24 * time.Hour
	defaultRefreshIdle  = 7 * 24 * time.Hour
	defaultStateTTL     = 10 * time.Minute
	tokenTypeBearer     = "Bearer"
	discordProviderName = "discord"
)

type Service struct {
	db                *pgxpool.Pool
	discord           DiscordConfig
	discordConfigured bool
	planResolver      PlanResolver
	httpClient        *http.Client
	jwtSecret         []byte
	now               func() time.Time
	accessTTL         time.Duration
	refreshTTL        time.Duration
	refreshIdle       time.Duration
	stateTTL          time.Duration
	cookieSecure      bool
	cookieDomain      string
	frontendOrigin    string
	issuer            string
	audience          string
}

type discordTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type discordProfile struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
	Avatar     string `json:"avatar"`
	Locale     string `json:"locale"`
	Email      string `json:"email"`
}

type userRecord struct {
	ID          string
	Username    string
	DisplayName string
	AvatarURL   string
	Timezone    string
	Locale      string
	Plan        string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type sessionRecord struct {
	ID         string
	UserID     string
	Status     string
	FamilyID   string
	LastSeenAt time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
	RevokedAt  *time.Time
}

type refreshRecord struct {
	ID         string
	SessionID  string
	TokenHash  string
	ConsumedAt *time.Time
	ExpiresAt  time.Time
	CreatedAt  time.Time
}

type authClaims struct {
	Sub   string   `json:"sub"`
	Sid   string   `json:"sid"`
	Plan  string   `json:"plan"`
	Roles []string `json:"roles"`
	Iss   string   `json:"iss"`
	Aud   string   `json:"aud"`
	Iat   int64    `json:"iat"`
	Exp   int64    `json:"exp"`
}

func ConfigFromEnv() (Config, error) {
	discord := DiscordConfig{
		ClientID:     strings.TrimSpace(os.Getenv("DISCORD_CLIENT_ID")),
		RedirectURI:  strings.TrimSpace(os.Getenv("DISCORD_REDIRECT_URI")),
		AuthorizeURL: "https://discord.com/oauth2/authorize",
		TokenURL:     "https://discord.com/api/oauth2/token",
		ProfileURL:   "https://discord.com/api/users/@me",
	}
	cookieDomain := strings.TrimSpace(os.Getenv("AUTH_COOKIE_DOMAIN"))
	frontendOrigin := strings.TrimSpace(os.Getenv("AUTH_FRONTEND_ORIGIN"))
	discordConfigured := false
	if secret, err := sharedconfig.ReadOptional("DISCORD_CLIENT_SECRET"); err != nil {
		return Config{}, err
	} else {
		discord.ClientSecret = secret
	}
	if discord.ClientID != "" || discord.ClientSecret != "" || discord.RedirectURI != "" {
		if err := discord.Validate(); err != nil {
			return Config{}, err
		}
		discordConfigured = true
	}
	jwtSecret, err := sharedconfig.ReadRequired("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}
	if jwtSecret == "" {
		return Config{}, fmt.Errorf("JWT_SECRET is required for auth")
	}
	if frontendOrigin != "" {
		parsed, err := url.Parse(frontendOrigin)
		if err != nil {
			return Config{}, fmt.Errorf("parse AUTH_FRONTEND_ORIGIN: %w", err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return Config{}, fmt.Errorf("AUTH_FRONTEND_ORIGIN must include scheme and host")
		}
	}
	return Config{
		Discord:           discord,
		DiscordConfigured: discordConfigured,
		JWTSecret:         jwtSecret,
		CookieDomain:      cookieDomain,
		FrontendOrigin:    frontendOrigin,
	}, nil
}

func (c DiscordConfig) Validate() error {
	if strings.TrimSpace(c.ClientID) == "" {
		return fmt.Errorf("DISCORD_CLIENT_ID is required")
	}
	for _, r := range c.ClientID {
		if r < '0' || r > '9' {
			return fmt.Errorf("DISCORD_CLIENT_ID must be a numeric discord application id")
		}
	}
	if strings.TrimSpace(c.ClientSecret) == "" {
		return fmt.Errorf("DISCORD_CLIENT_SECRET is required")
	}
	if strings.TrimSpace(c.RedirectURI) == "" {
		return fmt.Errorf("DISCORD_REDIRECT_URI is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(c.RedirectURI))
	if err != nil {
		return fmt.Errorf("parse DISCORD_REDIRECT_URI: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" || parsed.Path == "" {
		return fmt.Errorf("DISCORD_REDIRECT_URI must include scheme, host, and path")
	}
	return nil
}

func NewService(db *pgxpool.Pool, cfg Config) *Service {
	secureCookie := false
	if parsed, err := url.Parse(cfg.Discord.RedirectURI); err == nil && strings.EqualFold(parsed.Scheme, "https") {
		secureCookie = true
	}
	return &Service{
		db:                db,
		discord:           cfg.Discord,
		discordConfigured: cfg.DiscordConfigured,
		httpClient:        &http.Client{Timeout: 15 * time.Second},
		jwtSecret:         []byte(cfg.JWTSecret),
		now:               time.Now,
		accessTTL:         defaultAccessTTL,
		refreshTTL:        defaultRefreshTTL,
		refreshIdle:       defaultRefreshIdle,
		stateTTL:          defaultStateTTL,
		cookieSecure:      secureCookie,
		cookieDomain:      strings.TrimSpace(cfg.CookieDomain),
		frontendOrigin:    strings.TrimSpace(cfg.FrontendOrigin),
		issuer:            defaultIssuer,
		audience:          defaultAudience,
	}
}

func (s *Service) SetPlanResolver(resolver PlanResolver) {
	if s == nil {
		return
	}
	s.planResolver = resolver
}

func (s *Service) Enabled() bool {
	return s != nil && s.db != nil && len(s.jwtSecret) > 0
}

func (s *Service) DiscordEnabled() bool {
	return s != nil && s.Enabled() && s.discordConfigured
}

func (s *Service) CookieSecure() bool {
	if s == nil {
		return false
	}
	return s.cookieSecure
}

func (s *Service) CookieDomain() string {
	if s == nil {
		return ""
	}
	return s.cookieDomain
}

func (s *Service) FrontendOrigin() string {
	if s == nil {
		return ""
	}
	return s.frontendOrigin
}

func (s *Service) StartDiscordAuth(ctx context.Context) (StartResponse, error) {
	if !s.DiscordEnabled() {
		return StartResponse{}, ErrMissingConfig
	}
	state, err := generateToken(32)
	if err != nil {
		return StartResponse{}, fmt.Errorf("generate oauth state: %w", err)
	}
	return StartResponse{
		AuthorizationURL: s.discordAuthURL(state),
		State:            state,
		StateExpiresAt:   s.now().Add(s.stateTTL),
	}, nil
}

func (s *Service) CompleteDiscordAuth(ctx context.Context, code, state, cookieState string, meta RequestMeta) (AuthResponse, error) {
	if !s.DiscordEnabled() {
		return AuthResponse{}, ErrMissingConfig
	}
	if strings.TrimSpace(code) == "" {
		return AuthResponse{}, fmt.Errorf("code is required")
	}
	if strings.TrimSpace(state) == "" || strings.TrimSpace(cookieState) == "" || state != cookieState {
		return AuthResponse{}, ErrInvalidState
	}

	discordToken, err := s.exchangeDiscordCode(ctx, code)
	if err != nil {
		return AuthResponse{}, err
	}
	profile, err := s.fetchDiscordProfile(ctx, discordToken.AccessToken)
	if err != nil {
		return AuthResponse{}, err
	}

	now := s.now().UTC()
	user, err := s.upsertDiscordUser(ctx, profile, meta, now)
	if err != nil {
		return AuthResponse{}, err
	}

	session, refreshToken, refreshTokenValue, err := s.createSession(ctx, user, meta, now)
	if err != nil {
		return AuthResponse{}, err
	}
	effectivePlan, err := s.resolvePlan(ctx, user.ID, user.Plan)
	if err != nil {
		effectivePlan = user.Plan
	}
	user.Plan = effectivePlan
	accessToken, accessExpiresAt, err := s.issueAccessToken(user.ID, session.ID, effectivePlan, now)
	if err != nil {
		return AuthResponse{}, err
	}

	return AuthResponse{
		User:                  user.toProfile(),
		Session:               session.toInfo(),
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessExpiresAt,
		RefreshTokenExpiresAt: refreshToken.ExpiresAt,
		RefreshToken:          refreshTokenValue,
		TokenType:             tokenTypeBearer,
	}, nil
}

func (s *Service) Refresh(ctx context.Context, token string, meta RequestMeta) (AuthResponse, error) {
	if !s.Enabled() {
		return AuthResponse{}, ErrMissingConfig
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return AuthResponse{}, ErrUnauthorized
	}

	now := s.now().UTC()
	ref, err := s.refreshRecordByToken(ctx, token)
	if err != nil {
		return AuthResponse{}, err
	}
	if ref.ConsumedAt != nil {
		_ = s.revokeSession(ctx, ref.SessionID, now)
		return AuthResponse{}, ErrTokenReuse
	}
	if now.After(ref.ExpiresAt) {
		return AuthResponse{}, ErrTokenExpired
	}

	session, err := s.sessionByID(ctx, ref.SessionID)
	if err != nil {
		return AuthResponse{}, err
	}
	if strings.ToLower(session.Status) != "active" {
		return AuthResponse{}, ErrSessionRevoked
	}
	if now.After(session.ExpiresAt) || now.Sub(session.LastSeenAt) > s.refreshIdle {
		_ = s.revokeSession(ctx, session.ID, now)
		return AuthResponse{}, ErrUnauthorized
	}

	user, err := s.userByID(ctx, session.UserID)
	if err != nil {
		return AuthResponse{}, err
	}

	var result AuthResponse
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE auth_refresh_tokens SET consumed_at = $2 WHERE id = $1 AND consumed_at IS NULL`, ref.ID, now); err != nil {
			return fmt.Errorf("consume refresh token: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE auth_sessions SET last_seen_at = $2 WHERE id = $1`, session.ID, now); err != nil {
			return fmt.Errorf("touch session: %w", err)
		}

		nextToken, nextTokenRecord, err := s.insertRefreshTokenTx(ctx, tx, session.ID, now, ref.ID, now.Add(s.refreshTTL))
		if err != nil {
			return err
		}
		effectivePlan, err := s.resolvePlan(ctx, user.ID, user.Plan)
		if err != nil {
			effectivePlan = user.Plan
		}
		user.Plan = effectivePlan
		accessToken, accessExpiresAt, err := s.issueAccessToken(user.ID, session.ID, effectivePlan, now)
		if err != nil {
			return err
		}

		result = AuthResponse{
			User:                  user.toProfile(),
			Session:               session.withLastSeen(now).toInfo(),
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshTokenExpiresAt: nextTokenRecord.ExpiresAt,
			RefreshToken:          nextToken,
			TokenType:             tokenTypeBearer,
		}
		return nil
	})
	if err != nil {
		return AuthResponse{}, err
	}

	return result, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if !s.Enabled() {
		return ErrMissingConfig
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}

	if claims, err := s.parseAccessToken(token); err == nil {
		return s.revokeSession(ctx, claims.Sid, s.now().UTC())
	}
	ref, err := s.refreshRecordByToken(ctx, token)
	if err != nil {
		return err
	}
	return s.revokeSession(ctx, ref.SessionID, s.now().UTC())
}

func (s *Service) Me(ctx context.Context, token string) (MeResponse, error) {
	if !s.Enabled() {
		return MeResponse{}, ErrMissingConfig
	}
	claims, err := s.parseAccessToken(token)
	if err != nil {
		return MeResponse{}, err
	}
	session, err := s.sessionByID(ctx, claims.Sid)
	if err != nil {
		return MeResponse{}, err
	}
	if strings.ToLower(session.Status) != "active" || s.now().UTC().After(session.ExpiresAt) {
		return MeResponse{}, ErrUnauthorized
	}
	user, err := s.userByID(ctx, session.UserID)
	if err != nil {
		return MeResponse{}, err
	}
	if effectivePlan, err := s.resolvePlan(ctx, user.ID, user.Plan); err == nil {
		user.Plan = effectivePlan
	}
	return MeResponse{User: user.toProfile(), Session: session.toInfo()}, nil
}

func (s *Service) Entitlement(ctx context.Context, token string) (EntitlementResponse, error) {
	if !s.Enabled() {
		return EntitlementResponse{}, ErrMissingConfig
	}
	claims, err := s.parseAccessToken(token)
	if err != nil {
		return EntitlementResponse{}, err
	}
	session, err := s.sessionByID(ctx, claims.Sid)
	if err != nil {
		return EntitlementResponse{}, err
	}
	if strings.ToLower(session.Status) != "active" || s.now().UTC().After(session.ExpiresAt) {
		return EntitlementResponse{}, ErrUnauthorized
	}
	user, err := s.userByID(ctx, session.UserID)
	if err != nil {
		return EntitlementResponse{}, err
	}

	entitlement := Entitlement{Plan: normalizePlan(user.Plan), Source: "fallback", ResolvedAt: s.now().UTC()}
	if resolver, ok := s.planResolver.(interface {
		ResolveEntitlement(context.Context, string, string) (Entitlement, error)
	}); ok {
		if resolved, err := resolver.ResolveEntitlement(ctx, user.ID, user.Plan); err == nil {
			entitlement = resolved
		}
	} else if effectivePlan, err := s.resolvePlan(ctx, user.ID, user.Plan); err == nil {
		entitlement.Plan = effectivePlan
	}
	user.Plan = entitlement.Plan

	wallets, err := s.walletCount(ctx, user.ID)
	if err != nil {
		return EntitlementResponse{}, err
	}

	return EntitlementResponse{
		User:         user.toProfile(),
		Session:      session.toInfo(),
		Plan:         entitlement.Plan,
		Source:       entitlement.Source,
		Cached:       entitlement.Cached,
		ResolvedAt:   entitlement.ResolvedAt,
		WalletsFound: wallets,
	}, nil
}

func (s *Service) RequestWeb3Nonce(ctx context.Context, walletAddress, purpose string) (Web3NonceResponse, error) {
	if !s.Enabled() {
		return Web3NonceResponse{}, ErrMissingConfig
	}
	wallet, err := normalizeWalletAddress(walletAddress)
	if err != nil {
		return Web3NonceResponse{}, err
	}
	purpose = normalizeNoncePurpose(purpose)
	now := s.now().UTC()
	nonce, err := generateToken(24)
	if err != nil {
		return Web3NonceResponse{}, fmt.Errorf("generate nonce: %w", err)
	}
	if err := s.insertNonce(ctx, wallet, nonce, purpose, now.Add(5*time.Minute), now); err != nil {
		return Web3NonceResponse{}, err
	}
	return Web3NonceResponse{
		WalletAddress: wallet,
		Purpose:       purpose,
		Nonce:         nonce,
		Challenge:     buildNonceChallenge(wallet, purpose, nonce, now),
		ExpiresAt:     now.Add(5 * time.Minute),
	}, nil
}

func (s *Service) VerifyWeb3Signature(ctx context.Context, walletAddress, purpose, nonceValue, signature string, meta RequestMeta) (AuthResponse, error) {
	if !s.Enabled() {
		return AuthResponse{}, ErrMissingConfig
	}
	wallet, err := normalizeWalletAddress(walletAddress)
	if err != nil {
		return AuthResponse{}, err
	}
	purpose = normalizeNoncePurpose(purpose)
	nonceValue = strings.TrimSpace(nonceValue)
	signature = strings.TrimSpace(signature)
	if nonceValue == "" {
		return AuthResponse{}, ErrInvalidNonce
	}
	if signature == "" {
		return AuthResponse{}, ErrInvalidSignature
	}

	now := s.now().UTC()
	rec, err := s.nonceRecord(ctx, wallet, nonceValue, purpose)
	if err != nil {
		return AuthResponse{}, err
	}
	if rec.UsedAt != nil {
		return AuthResponse{}, ErrNonceConsumed
	}
	if now.After(rec.ExpiresAt) {
		return AuthResponse{}, ErrNonceExpired
	}

	expectedChallenge := buildNonceChallenge(rec.WalletAddress, rec.Purpose, rec.Nonce, rec.CreatedAt)
	recovered, err := recoverWalletFromSignature(expectedChallenge, signature)
	if err != nil {
		return AuthResponse{}, err
	}
	if !strings.EqualFold(recovered, rec.WalletAddress) {
		return AuthResponse{}, ErrInvalidSignature
	}

	var result AuthResponse
	err = s.withTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `UPDATE auth_nonces SET used_at = $2 WHERE id = $1 AND used_at IS NULL`, rec.ID, now); err != nil {
			return fmt.Errorf("consume nonce: %w", err)
		}

		user, err := s.upsertWalletUserTx(ctx, tx, rec.WalletAddress, meta, now)
		if err != nil {
			return err
		}
		session, refreshToken, refreshTokenValue, err := s.createSessionTx(ctx, tx, user.ID, meta, now)
		if err != nil {
			return err
		}
		effectivePlan, err := s.resolvePlan(ctx, user.ID, user.Plan)
		if err != nil {
			effectivePlan = user.Plan
		}
		user.Plan = effectivePlan
		accessToken, accessExpiresAt, err := s.issueAccessToken(user.ID, session.ID, effectivePlan, now)
		if err != nil {
			return err
		}
		result = AuthResponse{
			User:                  user.toProfile(),
			Session:               session.toInfo(),
			AccessToken:           accessToken,
			AccessTokenExpiresAt:  accessExpiresAt,
			RefreshTokenExpiresAt: refreshToken.ExpiresAt,
			RefreshToken:          refreshTokenValue,
			TokenType:             tokenTypeBearer,
		}
		return nil
	})
	if err != nil {
		return AuthResponse{}, err
	}
	return result, nil
}

func (s *Service) discordAuthURL(state string) string {
	values := url.Values{}
	values.Set("client_id", s.discord.ClientID)
	values.Set("redirect_uri", s.discord.RedirectURI)
	values.Set("response_type", "code")
	values.Set("scope", "identify email")
	values.Set("state", state)
	return s.discord.AuthorizeURL + "?" + values.Encode()
}

func (s *Service) exchangeDiscordCode(ctx context.Context, code string) (discordTokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", s.discord.ClientID)
	form.Set("client_secret", s.discord.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", s.discord.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.discord.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return discordTokenResponse{}, fmt.Errorf("create discord token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return discordTokenResponse{}, fmt.Errorf("exchange discord code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return discordTokenResponse{}, fmt.Errorf("discord token exchange failed: %s", resp.Status)
	}

	var tokenResp discordTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return discordTokenResponse{}, fmt.Errorf("decode discord token response: %w", err)
	}
	if strings.TrimSpace(tokenResp.AccessToken) == "" {
		return discordTokenResponse{}, fmt.Errorf("discord token response missing access_token")
	}
	return tokenResp, nil
}

func (s *Service) fetchDiscordProfile(ctx context.Context, accessToken string) (discordProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.discord.ProfileURL, nil)
	if err != nil {
		return discordProfile{}, fmt.Errorf("create discord profile request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return discordProfile{}, fmt.Errorf("fetch discord profile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return discordProfile{}, fmt.Errorf("discord profile request failed: %s", resp.Status)
	}

	var profile discordProfile
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return discordProfile{}, fmt.Errorf("decode discord profile: %w", err)
	}
	if strings.TrimSpace(profile.ID) == "" {
		return discordProfile{}, fmt.Errorf("discord profile missing id")
	}
	return profile, nil
}

func (s *Service) upsertDiscordUser(ctx context.Context, profile discordProfile, meta RequestMeta, now time.Time) (userRecord, error) {
	if s.db == nil {
		return userRecord{}, fmt.Errorf("postgres pool is required")
	}

	var out userRecord
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		existing, err := s.userByDiscordIDTx(ctx, tx, profile.ID)
		if err == nil {
			updated, err := s.updateUserFromProfileTx(ctx, tx, existing.ID, profile, now)
			if err != nil {
				return err
			}
			out = updated
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		created, err := s.insertUserTx(ctx, tx, profile, now)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `
INSERT INTO auth_identities (
    id, user_id, provider, provider_user_id, metadata_json, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			uuid.NewString(), created.ID, discordProviderName, profile.ID, discordMetadataJSON(profile, meta), now,
		); err != nil {
			return fmt.Errorf("insert auth identity: %w", err)
		}
		out = created
		return nil
	})
	if err != nil {
		return userRecord{}, err
	}
	return out, nil
}

func (s *Service) userByDiscordIDTx(ctx context.Context, tx pgx.Tx, providerUserID string) (userRecord, error) {
	row := tx.QueryRow(ctx, `
SELECT u.id, u.username, u.display_name, COALESCE(u.avatar_url, ''), COALESCE(u.timezone, 'UTC'), COALESCE(u.locale, 'id-ID'), COALESCE(u.plan_override, 'free'), u.created_at, u.updated_at
FROM auth_identities i
JOIN users u ON u.id = i.user_id
WHERE i.provider = $1 AND i.provider_user_id = $2`, discordProviderName, providerUserID)
	rec, err := scanUserRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userRecord{}, pgx.ErrNoRows
	}
	return rec, err
}

func (s *Service) userByID(ctx context.Context, userID string) (userRecord, error) {
	if s.db == nil {
		return userRecord{}, fmt.Errorf("postgres pool is required")
	}
	row := s.db.QueryRow(ctx, `
SELECT id, username, display_name, COALESCE(avatar_url, ''), COALESCE(timezone, 'UTC'), COALESCE(locale, 'id-ID'), COALESCE(plan_override, 'free'), created_at, updated_at
FROM users WHERE id = $1`, userID)
	rec, err := scanUserRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return userRecord{}, ErrUnauthorized
	}
	return rec, err
}

func (s *Service) sessionByID(ctx context.Context, sessionID string) (sessionRecord, error) {
	if s.db == nil {
		return sessionRecord{}, fmt.Errorf("postgres pool is required")
	}
	row := s.db.QueryRow(ctx, `
SELECT id, user_id, status, session_family_id, last_seen_at, expires_at, created_at, revoked_at
FROM auth_sessions WHERE id = $1`, sessionID)
	rec, err := scanSessionRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return sessionRecord{}, ErrUnauthorized
	}
	return rec, err
}

func (s *Service) refreshRecordByToken(ctx context.Context, token string) (refreshRecord, error) {
	if s.db == nil {
		return refreshRecord{}, fmt.Errorf("postgres pool is required")
	}
	hash := hashToken(token)
	row := s.db.QueryRow(ctx, `
SELECT id, session_id, token_hash, consumed_at, expires_at, created_at
FROM auth_refresh_tokens WHERE token_hash = $1`, hash)
	rec, err := scanRefreshRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return refreshRecord{}, ErrUnauthorized
	}
	return rec, err
}

func (s *Service) createSession(ctx context.Context, user userRecord, meta RequestMeta, now time.Time) (sessionRecord, refreshRecord, string, error) {
	if s.db == nil {
		return sessionRecord{}, refreshRecord{}, "", fmt.Errorf("postgres pool is required")
	}

	var session sessionRecord
	var refresh refreshRecord
	var refreshToken string
	err := s.withTx(ctx, func(tx pgx.Tx) error {
		var err error
		session, refresh, refreshToken, err = s.createSessionTx(ctx, tx, user.ID, meta, now)
		return err
	})
	if err != nil {
		return sessionRecord{}, refreshRecord{}, "", err
	}
	return session, refresh, refreshToken, nil
}

func (s *Service) createSessionTx(ctx context.Context, tx pgx.Tx, userID string, meta RequestMeta, now time.Time) (sessionRecord, refreshRecord, string, error) {
	session, err := s.insertSessionTx(ctx, tx, userID, meta, now)
	if err != nil {
		return sessionRecord{}, refreshRecord{}, "", err
	}
	refreshToken, refresh, err := s.insertRefreshTokenTx(ctx, tx, session.ID, now, "", now.Add(s.refreshTTL))
	if err != nil {
		return sessionRecord{}, refreshRecord{}, "", err
	}
	return session, refresh, refreshToken, nil
}

func (s *Service) insertSessionTx(ctx context.Context, tx pgx.Tx, userID string, meta RequestMeta, now time.Time) (sessionRecord, error) {
	sessionID := uuid.NewString()
	familyID := uuid.NewString()
	ipHash := hashWithPepper(s.jwtSecret, normalizeRemoteAddr(meta.RemoteAddr))
	uaHash := hashWithPepper(s.jwtSecret, strings.TrimSpace(meta.UserAgent))
	expiresAt := now.Add(s.refreshTTL)
	row := tx.QueryRow(ctx, `
INSERT INTO auth_sessions (
    id, user_id, status, session_family_id, ip_hash, user_agent_hash, last_seen_at, expires_at, created_at
) VALUES ($1, $2, 'active', $3, $4, $5, $6, $7, $6)
RETURNING id, user_id, status, session_family_id, last_seen_at, expires_at, created_at, revoked_at`,
		sessionID, userID, familyID, ipHash, uaHash, now, expiresAt,
	)
	return scanSessionRow(row)
}

func (s *Service) insertRefreshTokenTx(ctx context.Context, tx pgx.Tx, sessionID string, now time.Time, rotatedFrom string, expiresAt time.Time) (string, refreshRecord, error) {
	token, err := generateToken(32)
	if err != nil {
		return "", refreshRecord{}, err
	}
	id := uuid.NewString()
	var rotatedFromValue any
	if strings.TrimSpace(rotatedFrom) != "" {
		rotatedFromValue = rotatedFrom
	}
	row := tx.QueryRow(ctx, `
INSERT INTO auth_refresh_tokens (
    id, session_id, token_hash, rotated_from_token_id, consumed_at, expires_at, created_at
) VALUES ($1, $2, $3, $4, NULL, $5, $6)
RETURNING id, session_id, token_hash, consumed_at, expires_at, created_at`,
		id, sessionID, hashToken(token), rotatedFromValue, expiresAt, now,
	)
	record, err := scanRefreshRow(row)
	if err != nil {
		return "", refreshRecord{}, err
	}
	return token, record, nil
}

func (s *Service) updateUserFromProfileTx(ctx context.Context, tx pgx.Tx, userID string, profile discordProfile, now time.Time) (userRecord, error) {
	displayName := strings.TrimSpace(firstNonEmpty(profile.GlobalName, profile.Username))
	if displayName == "" {
		displayName = "Discord User"
	}
	avatarURL := discordAvatarURL(profile)
	if _, err := tx.Exec(ctx, `
UPDATE users
SET display_name = $2,
    avatar_url = $3,
    locale = COALESCE(NULLIF($4, ''), locale),
    updated_at = $5
WHERE id = $1`,
		userID, displayName, avatarURL, strings.TrimSpace(profile.Locale), now,
	); err != nil {
		return userRecord{}, fmt.Errorf("update user: %w", err)
	}
	row := tx.QueryRow(ctx, `
SELECT id, username, display_name, COALESCE(avatar_url, ''), COALESCE(timezone, 'UTC'), COALESCE(locale, 'id-ID'), COALESCE(plan_override, 'free'), created_at, updated_at
FROM users WHERE id = $1`, userID)
	return scanUserRow(row)
}

func (s *Service) insertUserTx(ctx context.Context, tx pgx.Tx, profile discordProfile, now time.Time) (userRecord, error) {
	displayName := strings.TrimSpace(firstNonEmpty(profile.GlobalName, profile.Username))
	if displayName == "" {
		displayName = "Discord User"
	}
	avatarURL := discordAvatarURL(profile)
	email := strings.TrimSpace(profile.Email)
	if email == "" {
		email = sanitizeUsername(displayName)
		if email == "" {
			email = "discord_user"
		}
		email += "@discord.local"
	}
	locale := strings.TrimSpace(profile.Locale)
	if locale == "" {
		locale = "id-ID"
	}
	base := sanitizeUsername(firstNonEmpty(profile.GlobalName, profile.Username))
	if base == "" {
		base = "discord_user"
	}

	for attempt := 0; attempt < 8; attempt++ {
		username := usernameCandidate(base, attempt)
		id := uuid.NewString()
		row := tx.QueryRow(ctx, `
INSERT INTO users (
    id, email, name, picture, username, display_name, avatar_url, timezone, locale, plan_override, created_at, updated_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, 'UTC', $8, NULL, $9, $9)
ON CONFLICT (username) DO NOTHING
RETURNING id, username, display_name, COALESCE(avatar_url, ''), COALESCE(timezone, 'UTC'), COALESCE(locale, 'id-ID'), COALESCE(plan_override, 'free'), created_at, updated_at`,
			id, email, displayName, avatarURL, username, displayName, avatarURL, locale, now,
		)
		user, err := scanUserRow(row)
		if err == nil {
			return user, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return userRecord{}, err
		}
	}

	return userRecord{}, fmt.Errorf("unable to allocate username for discord user %s", profile.ID)
}

func (s *Service) revokeSession(ctx context.Context, sessionID string, now time.Time) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	_, err := s.db.Exec(ctx, `
UPDATE auth_sessions
SET status = 'revoked',
    revoked_at = COALESCE(revoked_at, $2)
WHERE id = $1`, sessionID, now)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Service) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if s.db == nil {
		return fmt.Errorf("postgres pool is required")
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin auth transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit auth transaction: %w", err)
	}
	return nil
}

func (s *Service) issueAccessToken(userID, sessionID, plan string, now time.Time) (string, time.Time, error) {
	claims := authClaims{
		Sub:   userID,
		Sid:   sessionID,
		Plan:  plan,
		Roles: []string{"user"},
		Iss:   s.issuer,
		Aud:   s.audience,
		Iat:   now.Unix(),
		Exp:   now.Add(s.accessTTL).Unix(),
	}
	token, err := signJWT(s.jwtSecret, claims)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, now.Add(s.accessTTL), nil
}

func (s *Service) parseAccessToken(token string) (authClaims, error) {
	return verifyJWT(s.jwtSecret, strings.TrimSpace(token), s.issuer, s.audience)
}

func (s *Service) resolvePlan(ctx context.Context, userID, fallback string) (string, error) {
	if s == nil || s.planResolver == nil {
		return normalizePlan(fallback), nil
	}
	plan, err := s.planResolver.ResolvePlan(ctx, userID, fallback)
	if err != nil {
		return normalizePlan(fallback), err
	}
	return normalizePlan(plan), nil
}

func (s *Service) walletCount(ctx context.Context, userID string) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("postgres pool is required")
	}
	var count int
	if err := s.db.QueryRow(ctx, `
SELECT COUNT(*)
FROM auth_identities
WHERE user_id = $1 AND provider = 'evm'`, userID).Scan(&count); err != nil {
		return 0, fmt.Errorf("count wallet identities: %w", err)
	}
	return count, nil
}

func normalizeRemoteAddr(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	host, _, err := netSplitHostPort(raw)
	if err == nil {
		return host
	}
	return raw
}

func discordMetadataJSON(profile discordProfile, meta RequestMeta) map[string]any {
	return map[string]any{
		"discord": map[string]any{
			"id":          profile.ID,
			"username":    profile.Username,
			"global_name": profile.GlobalName,
			"avatar":      profile.Avatar,
			"locale":      profile.Locale,
			"email":       profile.Email,
		},
		"request": map[string]any{
			"remote_addr": meta.RemoteAddr,
			"user_agent":  meta.UserAgent,
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func sanitizeUsername(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(raw))
	underscore := false
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			underscore = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			underscore = false
		default:
			if !underscore {
				b.WriteByte('_')
				underscore = true
			}
		}
	}
	out := strings.Trim(b.String(), "_")
	if len(out) > 24 {
		out = out[:24]
	}
	if len(out) < 3 {
		return ""
	}
	return out
}

func usernameCandidate(base string, attempt int) string {
	if attempt <= 0 {
		return trimUsername(base)
	}
	suffix := fmt.Sprintf("%d", attempt)
	trimmedBase := trimUsername(base)
	maxBase := 24 - 1 - len(suffix)
	if maxBase < 3 {
		maxBase = 3
	}
	if len(trimmedBase) > maxBase {
		trimmedBase = trimmedBase[:maxBase]
	}
	return trimUsername(trimmedBase + "_" + suffix)
}

func trimUsername(raw string) string {
	raw = strings.Trim(raw, "_")
	if len(raw) > 24 {
		raw = raw[:24]
	}
	if len(raw) < 3 {
		return "discord_user"
	}
	return raw
}

func discordAvatarURL(profile discordProfile) string {
	if strings.TrimSpace(profile.Avatar) == "" || strings.TrimSpace(profile.ID) == "" {
		return ""
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", profile.ID, profile.Avatar)
}

func generateToken(n int) (string, error) {
	if n <= 0 {
		n = 32
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return hex.EncodeToString(sum[:])
}

func hashWithPepper(secret []byte, value string) string {
	h := sha256.New()
	_, _ = h.Write(secret)
	_, _ = h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

func netSplitHostPort(value string) (string, string, error) {
	return net.SplitHostPort(value)
}
