package authapi

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultSubscriptionMethod = "isSubscribed"

const subscriptionABIJSON = `[
	{
		"inputs":[{"internalType":"address","name":"wallet","type":"address"}],
		"name":"isSubscribed",
		"outputs":[{"internalType":"bool","name":"","type":"bool"}],
		"stateMutability":"view",
		"type":"function"
	}
]`

type PlanResolver interface {
	ResolvePlan(ctx context.Context, userID, fallback string) (string, error)
}

type EntitlementResolver interface {
	ResolveEntitlement(ctx context.Context, userID, fallback string) (Entitlement, error)
}

type SubscriptionChecker interface {
	HasActiveSubscription(ctx context.Context, walletAddress string) (bool, error)
}

type sourceAwareSubscriptionChecker interface {
	HasActiveSubscriptionWithSource(ctx context.Context, walletAddress string) (bool, string, error)
}

type Entitlement struct {
	Plan       string
	Source     string
	Cached     bool
	ResolvedAt time.Time
}

type SubscriptionResolver struct {
	db        *pgxpool.Pool
	checker   SubscriptionChecker
	cacheTTL  time.Duration
	mu        sync.RWMutex
	planCache map[string]cachedPlan
}

type cachedPlan struct {
	plan      string
	source    string
	expiresAt time.Time
}

type EthSubscriptionChecker struct {
	name     string
	chainID  int64
	client   *ethclient.Client
	contract common.Address
	method   string
	abi      abi.ABI
}

type multiEthSubscriptionChecker struct {
	checkers []*EthSubscriptionChecker
}

type subscriptionNetworkConfig struct {
	Name            string
	ChainID         int64
	RPCURL          string
	ContractAddress string
	Method          string
}

func NewSubscriptionResolver(db *pgxpool.Pool, checker SubscriptionChecker, cacheTTL time.Duration) *SubscriptionResolver {
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &SubscriptionResolver{
		db:        db,
		checker:   checker,
		cacheTTL:  cacheTTL,
		planCache: make(map[string]cachedPlan),
	}
}

func NewSubscriptionResolverFromEnv(ctx context.Context, db *pgxpool.Pool) (*SubscriptionResolver, error) {
	cacheTTL := 5 * time.Minute
	if raw := strings.TrimSpace(os.Getenv("SUBSCRIPTION_CACHE_TTL")); raw != "" {
		ttl, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("parse SUBSCRIPTION_CACHE_TTL: %w", err)
		}
		cacheTTL = ttl
	}
	if rawNetworks := strings.TrimSpace(os.Getenv("SUBSCRIPTION_NETWORKS")); rawNetworks != "" {
		entries, err := parseSubscriptionNetworks(rawNetworks)
		if err != nil {
			return nil, err
		}
		checkers := make([]*EthSubscriptionChecker, 0, len(entries))
		for _, entry := range entries {
			checker, err := NewEthSubscriptionChecker(ctx, entry.Name, entry.ChainID, entry.RPCURL, entry.ContractAddress, entry.Method)
			if err != nil {
				return nil, fmt.Errorf("create subscription checker for %s: %w", entry.Name, err)
			}
			checkers = append(checkers, checker)
		}
		if len(checkers) == 0 {
			return nil, nil
		}
		return NewSubscriptionResolver(db, &multiEthSubscriptionChecker{checkers: checkers}, cacheTTL), nil
	}
	rpcURL := strings.TrimSpace(os.Getenv("SUBSCRIPTION_RPC_URL"))
	contractAddress := strings.TrimSpace(os.Getenv("SUBSCRIPTION_MANAGER_ADDRESS"))
	if rpcURL == "" || contractAddress == "" {
		return nil, nil
	}
	method := strings.TrimSpace(os.Getenv("SUBSCRIPTION_METHOD"))
	if method == "" {
		method = defaultSubscriptionMethod
	}
	checker, err := NewEthSubscriptionChecker(ctx, "", 0, rpcURL, contractAddress, method)
	if err != nil {
		return nil, err
	}
	return NewSubscriptionResolver(db, checker, cacheTTL), nil
}

func NewEthSubscriptionChecker(ctx context.Context, name string, chainID int64, rpcURL, contractAddress, method string) (*EthSubscriptionChecker, error) {
	if strings.TrimSpace(rpcURL) == "" {
		return nil, fmt.Errorf("SUBSCRIPTION_RPC_URL is required")
	}
	if strings.TrimSpace(contractAddress) == "" {
		return nil, fmt.Errorf("SUBSCRIPTION_MANAGER_ADDRESS is required")
	}
	if !common.IsHexAddress(contractAddress) {
		return nil, fmt.Errorf("SUBSCRIPTION_MANAGER_ADDRESS must be a valid hex address")
	}
	client, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("dial subscription rpc: %w", err)
	}
	parsedABI, err := abi.JSON(strings.NewReader(subscriptionABIJSON))
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("parse subscription abi: %w", err)
	}
	if method == "" {
		method = defaultSubscriptionMethod
	}
	if _, ok := parsedABI.Methods[method]; !ok {
		client.Close()
		return nil, fmt.Errorf("subscription method %q is not defined in the default abi", method)
	}
	return &EthSubscriptionChecker{
		name:     strings.TrimSpace(name),
		chainID:  chainID,
		client:   client,
		contract: common.HexToAddress(contractAddress),
		method:   method,
		abi:      parsedABI,
	}, nil
}

func (r *SubscriptionResolver) ResolvePlan(ctx context.Context, userID, fallback string) (string, error) {
	entitlement, err := r.ResolveEntitlement(ctx, userID, fallback)
	return entitlement.Plan, err
}

func (r *SubscriptionResolver) ResolveEntitlement(ctx context.Context, userID, fallback string) (Entitlement, error) {
	if r == nil || r.db == nil {
		return Entitlement{Plan: normalizePlan(fallback), Source: "fallback"}, nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Entitlement{Plan: normalizePlan(fallback), Source: "fallback"}, fmt.Errorf("user id is required")
	}

	now := time.Now().UTC()
	if plan, source, ok := r.cachedPlan(userID, now); ok {
		return Entitlement{
			Plan:       plan,
			Source:     source,
			Cached:     true,
			ResolvedAt: now,
		}, nil
	}

	planOverride, err := r.planOverride(ctx, userID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		ent := Entitlement{Plan: normalizePlan(fallback), Source: "fallback", ResolvedAt: now}
		return ent, err
	}

	active, source, chainErr := r.hasActiveSubscription(ctx, userID)
	if active {
		if source == "" {
			source = "onchain"
		}
		r.storePlan(userID, "premium", source, now)
		return Entitlement{Plan: "premium", Source: source, ResolvedAt: now}, nil
	}

	if planOverride != "" {
		plan := normalizePlan(planOverride)
		r.storePlan(userID, plan, "override", now)
		return Entitlement{Plan: plan, Source: "override", ResolvedAt: now}, nil
	}

	plan := normalizePlan(fallback)
	r.storePlan(userID, plan, "fallback", now)
	if chainErr != nil {
		return Entitlement{Plan: plan, Source: "fallback", ResolvedAt: now}, chainErr
	}
	return Entitlement{Plan: plan, Source: "fallback", ResolvedAt: now}, nil
}

func (r *SubscriptionResolver) planOverride(ctx context.Context, userID string) (string, error) {
	row := r.db.QueryRow(ctx, `SELECT COALESCE(NULLIF(TRIM(plan_override), ''), '') FROM users WHERE id = $1`, userID)
	var planOverride string
	if err := row.Scan(&planOverride); err != nil {
		return "", err
	}
	return strings.TrimSpace(planOverride), nil
}

func (r *SubscriptionResolver) hasActiveSubscription(ctx context.Context, userID string) (bool, string, error) {
	rows, err := r.db.Query(ctx, `
SELECT provider_user_id
FROM auth_identities
WHERE user_id = $1 AND provider = 'evm'
ORDER BY created_at ASC`, userID)
	if err != nil {
		return false, "", fmt.Errorf("query wallet identities: %w", err)
	}
	defer rows.Close()

	var lastErr error
	for rows.Next() {
		var wallet string
		if err := rows.Scan(&wallet); err != nil {
			return false, "", fmt.Errorf("scan wallet identity: %w", err)
		}
		if r.checker == nil {
			continue
		}
		active, source, err := r.checkerHasActiveSubscription(ctx, wallet)
		if err != nil {
			lastErr = err
			continue
		}
		if active {
			return true, source, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", fmt.Errorf("iterate wallet identities: %w", err)
	}
	if lastErr != nil {
		return false, "", lastErr
	}
	return false, "", nil
}

func (r *SubscriptionResolver) checkerHasActiveSubscription(ctx context.Context, walletAddress string) (bool, string, error) {
	if r == nil || r.checker == nil {
		return false, "", nil
	}
	if sourceChecker, ok := r.checker.(sourceAwareSubscriptionChecker); ok {
		return sourceChecker.HasActiveSubscriptionWithSource(ctx, walletAddress)
	}
	active, err := r.checker.HasActiveSubscription(ctx, walletAddress)
	if !active || err != nil {
		return active, "", err
	}
	return true, "onchain", nil
}

func (r *SubscriptionResolver) cachedPlan(userID string, now time.Time) (string, string, bool) {
	r.mu.RLock()
	entry, ok := r.planCache[userID]
	r.mu.RUnlock()
	if !ok || now.After(entry.expiresAt) {
		return "", "", false
	}
	return entry.plan, entry.source, true
}

func (r *SubscriptionResolver) storePlan(userID, plan, source string, now time.Time) {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.planCache[userID] = cachedPlan{
		plan:      normalizePlan(plan),
		source:    strings.TrimSpace(source),
		expiresAt: now.Add(r.cacheTTL),
	}
	r.mu.Unlock()
}

func (c *EthSubscriptionChecker) HasActiveSubscription(ctx context.Context, walletAddress string) (bool, error) {
	active, _, err := c.HasActiveSubscriptionWithSource(ctx, walletAddress)
	return active, err
}

func (c *EthSubscriptionChecker) HasActiveSubscriptionWithSource(ctx context.Context, walletAddress string) (bool, string, error) {
	if c == nil || c.client == nil {
		return false, "", nil
	}
	if !common.IsHexAddress(walletAddress) {
		return false, "", fmt.Errorf("invalid wallet address")
	}
	wallet := common.HexToAddress(strings.ToLower(strings.TrimSpace(walletAddress)))
	data, err := c.abi.Pack(c.method, wallet)
	if err != nil {
		return false, "", fmt.Errorf("pack subscription call: %w", err)
	}
	msg := ethereum.CallMsg{
		To:   &c.contract,
		Data: data,
	}
	out, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return false, "", fmt.Errorf("call subscription contract: %w", err)
	}
	values, err := c.abi.Unpack(c.method, out)
	if err != nil {
		return false, "", fmt.Errorf("unpack subscription response: %w", err)
	}
	if len(values) == 0 {
		return false, "", fmt.Errorf("subscription response was empty")
	}
	active, ok := values[0].(bool)
	if !ok {
		return false, "", fmt.Errorf("subscription response had unexpected type %T", values[0])
	}
	if !active {
		return false, "", nil
	}
	return true, c.sourceLabel(), nil
}

func (c *EthSubscriptionChecker) sourceLabel() string {
	if c == nil {
		return "onchain"
	}
	if name := strings.TrimSpace(c.name); name != "" {
		return "onchain:" + name
	}
	if c.chainID > 0 {
		return fmt.Sprintf("onchain:%d", c.chainID)
	}
	return "onchain"
}

func parseSubscriptionNetworks(raw string) ([]subscriptionNetworkConfig, error) {
	segments := strings.Split(raw, ";")
	networks := make([]subscriptionNetworkConfig, 0, len(segments))
	for _, segment := range segments {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		parts := strings.Split(segment, "|")
		if len(parts) < 4 || len(parts) > 5 {
			return nil, fmt.Errorf("invalid SUBSCRIPTION_NETWORKS entry %q: expected name|chain_id|rpc_url|contract_address[|method]", segment)
		}
		chainID, err := parseSubscriptionChainID(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("parse subscription chain id for %q: %w", segment, err)
		}
		name := strings.TrimSpace(parts[0])
		if name == "" {
			name = fmt.Sprintf("chain-%d", chainID)
		}
		rpcURL := strings.TrimSpace(parts[2])
		contractAddress := strings.TrimSpace(parts[3])
		if rpcURL == "" {
			return nil, fmt.Errorf("rpc url is required for SUBSCRIPTION_NETWORKS entry %q", segment)
		}
		if contractAddress == "" {
			return nil, fmt.Errorf("contract address is required for SUBSCRIPTION_NETWORKS entry %q", segment)
		}
		method := defaultSubscriptionMethod
		if len(parts) == 5 && strings.TrimSpace(parts[4]) != "" {
			method = strings.TrimSpace(parts[4])
		}
		networks = append(networks, subscriptionNetworkConfig{
			Name:            name,
			ChainID:         chainID,
			RPCURL:          rpcURL,
			ContractAddress: contractAddress,
			Method:          method,
		})
	}
	return networks, nil
}

func parseSubscriptionChainID(raw string) (int64, error) {
	if raw == "" {
		return 0, fmt.Errorf("chain id is required")
	}
	chainID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid chain id: %w", err)
	}
	if chainID <= 0 {
		return 0, fmt.Errorf("chain id must be positive")
	}
	return chainID, nil
}

func (m *multiEthSubscriptionChecker) HasActiveSubscription(ctx context.Context, walletAddress string) (bool, error) {
	active, _, err := m.HasActiveSubscriptionWithSource(ctx, walletAddress)
	return active, err
}

func (m *multiEthSubscriptionChecker) HasActiveSubscriptionWithSource(ctx context.Context, walletAddress string) (bool, string, error) {
	if m == nil || len(m.checkers) == 0 {
		return false, "", nil
	}
	var lastErr error
	for _, checker := range m.checkers {
		active, source, err := checker.HasActiveSubscriptionWithSource(ctx, walletAddress)
		if err != nil {
			lastErr = err
			continue
		}
		if active {
			if source == "" {
				source = checker.sourceLabel()
			}
			return true, source, nil
		}
	}
	if lastErr != nil {
		return false, "", lastErr
	}
	return false, "", nil
}

func normalizePlan(plan string) string {
	switch strings.ToLower(strings.TrimSpace(plan)) {
	case "premium":
		return "premium"
	default:
		return "free"
	}
}
