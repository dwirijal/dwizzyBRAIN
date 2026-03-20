package authapi

import "time"

const (
	defaultIssuer   = "dwizzyBRAIN"
	defaultAudience = "dwizzyBRAIN-api"
)

type Config struct {
	Discord           DiscordConfig
	DiscordConfigured bool
	JWTSecret         string
	CookieDomain      string
	FrontendOrigin    string
}

type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	AuthorizeURL string
	TokenURL     string
	ProfileURL   string
}

type RequestMeta struct {
	RemoteAddr string
	UserAgent  string
}

type Web3NonceRequest struct {
	WalletAddress string `json:"wallet_address"`
	Purpose       string `json:"purpose,omitempty"`
}

type Web3NonceResponse struct {
	WalletAddress string    `json:"wallet_address"`
	Purpose       string    `json:"purpose"`
	Nonce         string    `json:"nonce"`
	Challenge     string    `json:"challenge"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type Web3VerifyRequest struct {
	WalletAddress string `json:"wallet_address"`
	Purpose       string `json:"purpose,omitempty"`
	Nonce         string `json:"nonce"`
	Signature     string `json:"signature"`
}

type StartResponse struct {
	AuthorizationURL string    `json:"authorization_url"`
	State            string    `json:"state"`
	StateExpiresAt   time.Time `json:"state_expires_at"`
}

type UserProfile struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Timezone    string `json:"timezone"`
	Locale      string `json:"locale"`
	Plan        string `json:"plan"`
}

type SessionInfo struct {
	ID         string     `json:"id"`
	Status     string     `json:"status"`
	FamilyID   string     `json:"session_family_id"`
	LastSeenAt time.Time  `json:"last_seen_at"`
	ExpiresAt  time.Time  `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
}

type AuthResponse struct {
	User                  UserProfile `json:"user"`
	Session               SessionInfo `json:"session"`
	AccessToken           string      `json:"access_token"`
	AccessTokenExpiresAt  time.Time   `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time   `json:"refresh_token_expires_at"`
	RefreshToken          string      `json:"-"`
	TokenType             string      `json:"token_type"`
}

type MeResponse struct {
	User    UserProfile `json:"user"`
	Session SessionInfo `json:"session"`
}

type EntitlementResponse struct {
	User         UserProfile `json:"user"`
	Session      SessionInfo `json:"session"`
	Plan         string      `json:"plan"`
	Source       string      `json:"source"`
	Cached       bool        `json:"cached"`
	ResolvedAt   time.Time   `json:"resolved_at"`
	WalletsFound int         `json:"wallets_found,omitempty"`
}
