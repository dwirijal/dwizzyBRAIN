package defiapi

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultProtocolsLimit = 20
	defaultChainsLimit    = 20
	defaultDexesLimit     = 20
)

type Service struct {
	db *pgxpool.Pool
}

type ProtocolSummary struct {
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Logo        string    `json:"logo,omitempty"`
	Category    string    `json:"category,omitempty"`
	URL         string    `json:"url,omitempty"`
	Twitter     string    `json:"twitter,omitempty"`
	CoinID      string    `json:"coin_id,omitempty"`
	AuditStatus string    `json:"audit_status,omitempty"`
	Oracles     []string  `json:"oracles,omitempty"`
	Tier        string    `json:"tier,omitempty"`
	TVLUSD      *float64  `json:"tvl_usd,omitempty"`
	TVLChange1D *float64  `json:"tvl_change_1d_pct,omitempty"`
	TVLChange7D *float64  `json:"tvl_change_7d_pct,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProtocolDetail = ProtocolSummary

type ChainSummary struct {
	Chain     string    `json:"chain"`
	TVLUSD    *float64  `json:"tvl_usd,omitempty"`
	Change1D  *float64  `json:"tvl_change_1d_pct,omitempty"`
	Change7D  *float64  `json:"tvl_change_7d_pct,omitempty"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DexSummary struct {
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Volume24HUSD *float64  `json:"volume_24h_usd,omitempty"`
	Volume7DUSD  *float64  `json:"volume_7d_usd,omitempty"`
	Volume30DUSD *float64  `json:"volume_30d_usd,omitempty"`
	Change1D     *float64  `json:"volume_change_1d_pct,omitempty"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Overview struct {
	Protocols []ProtocolSummary `json:"protocols"`
	Chains    []ChainSummary    `json:"chains"`
	Dexes     []DexSummary      `json:"dexes"`
}

type ProtocolList struct {
	Items []ProtocolSummary
	Total int
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Overview(ctx context.Context, limit int) (Overview, error) {
	if limit <= 0 {
		limit = 5
	}

	protocols, err := s.ListProtocols(ctx, limit, 0, "")
	if err != nil {
		return Overview{}, err
	}
	chains, err := s.ListChains(ctx, limit)
	if err != nil {
		return Overview{}, err
	}
	dexes, err := s.ListDexes(ctx, limit)
	if err != nil {
		return Overview{}, err
	}

	return Overview{
		Protocols: protocols.Items,
		Chains:    chains,
		Dexes:     dexes,
	}, nil
}

func (s *Service) ListProtocols(ctx context.Context, limit, offset int, category string) (ProtocolList, error) {
	if s.db == nil {
		return ProtocolList{}, fmt.Errorf("postgres pool is required")
	}
	limit = clampLimit(limit, defaultProtocolsLimit)
	if offset < 0 {
		offset = 0
	}

	args := []any{}
	where := []string{}
	if strings.TrimSpace(category) != "" {
		args = append(args, strings.TrimSpace(category))
		where = append(where, "LOWER(COALESCE(NULLIF(p.category, ''), 'other')) = LOWER($1)")
	}

	countQuery := `
SELECT count(*)
FROM defi_protocols p`
	if len(where) > 0 {
		countQuery += "\nWHERE " + strings.Join(where, " AND ")
	}

	var total int
	countArgs := append([]any(nil), args...)
	if err := s.db.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return ProtocolList{}, fmt.Errorf("count defi protocols: %w", err)
	}

	query := `
SELECT
    COALESCE(NULLIF(p.slug, ''), '') AS slug,
    COALESCE(NULLIF(p.name, ''), '') AS name,
    COALESCE(NULLIF(p.description, ''), '') AS description,
    COALESCE(NULLIF(p.logo, ''), '') AS logo,
    COALESCE(NULLIF(p.category, ''), 'other') AS category,
    COALESCE(NULLIF(p.url, ''), '') AS url,
    COALESCE(NULLIF(p.twitter, ''), '') AS twitter,
    COALESCE(NULLIF(p.coin_id, ''), '') AS coin_id,
    COALESCE(NULLIF(p.audit_status, ''), '') AS audit_status,
    COALESCE(p.oracles, '{}'::text[]) AS oracles,
    COALESCE(p.updated_at, NOW()) AS updated_at,
    COALESCE(c.tier, 'on_demand') AS tier,
    t.tvl,
    t.change_1d,
    t.change_7d
FROM defi_protocols p
LEFT JOIN defi_protocol_coverage c ON c.slug = p.slug
LEFT JOIN defi_protocol_tvl_latest t ON t.slug = p.slug`
	if len(where) > 0 {
		query += "\nWHERE " + strings.Join(where, " AND ")
	}
	query += "\nORDER BY COALESCE(t.tvl, 0) DESC, p.slug\nLIMIT $" + strconvForArg(len(args)+1) + " OFFSET $" + strconvForArg(len(args)+2)

	args = append(args, limit, offset)
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return ProtocolList{}, fmt.Errorf("query defi protocols: %w", err)
	}
	defer rows.Close()

	items := make([]ProtocolSummary, 0)
	for rows.Next() {
		item, err := scanProtocolRow(rows)
		if err != nil {
			return ProtocolList{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return ProtocolList{}, fmt.Errorf("iterate defi protocols: %w", err)
	}

	return ProtocolList{Items: items, Total: total}, nil
}

func (s *Service) Protocol(ctx context.Context, slug string) (ProtocolDetail, error) {
	if s.db == nil {
		return ProtocolDetail{}, fmt.Errorf("postgres pool is required")
	}
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return ProtocolDetail{}, fmt.Errorf("slug is required")
	}

	query := `
SELECT
    COALESCE(NULLIF(p.slug, ''), '') AS slug,
    COALESCE(NULLIF(p.name, ''), '') AS name,
    COALESCE(NULLIF(p.description, ''), '') AS description,
    COALESCE(NULLIF(p.logo, ''), '') AS logo,
    COALESCE(NULLIF(p.category, ''), 'other') AS category,
    COALESCE(NULLIF(p.url, ''), '') AS url,
    COALESCE(NULLIF(p.twitter, ''), '') AS twitter,
    COALESCE(NULLIF(p.coin_id, ''), '') AS coin_id,
    COALESCE(NULLIF(p.audit_status, ''), '') AS audit_status,
    COALESCE(p.oracles, '{}'::text[]) AS oracles,
    COALESCE(p.updated_at, NOW()) AS updated_at,
    COALESCE(c.tier, 'on_demand') AS tier,
    t.tvl,
    t.change_1d,
    t.change_7d
FROM defi_protocols p
LEFT JOIN defi_protocol_coverage c ON c.slug = p.slug
LEFT JOIN defi_protocol_tvl_latest t ON t.slug = p.slug
WHERE p.slug = $1
LIMIT 1`

	row, err := scanProtocolRow(s.db.QueryRow(ctx, query, slug))
	if err != nil {
		if err == pgx.ErrNoRows {
			return ProtocolDetail{}, fmt.Errorf("protocol %q not found", slug)
		}
		return ProtocolDetail{}, err
	}

	return row, nil
}

func (s *Service) ListChains(ctx context.Context, limit int) ([]ChainSummary, error) {
	if s.db == nil {
		return nil, fmt.Errorf("postgres pool is required")
	}
	limit = clampLimit(limit, defaultChainsLimit)

	const query = `
SELECT chain, tvl, change_1d, change_7d, COALESCE(updated_at, NOW())
FROM defi_chain_tvl_latest
ORDER BY tvl DESC, chain
LIMIT $1`

	rows, err := s.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query defi chains: %w", err)
	}
	defer rows.Close()

	items := make([]ChainSummary, 0)
	for rows.Next() {
		var item ChainSummary
		var tvl, change1D, change7D sql.NullFloat64
		if err := rows.Scan(&item.Chain, &tvl, &change1D, &change7D, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan defi chain: %w", err)
		}
		item.TVLUSD = float64Ptr(tvl)
		item.Change1D = float64Ptr(change1D)
		item.Change7D = float64Ptr(change7D)
		item.UpdatedAt = item.UpdatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate defi chains: %w", err)
	}

	return items, nil
}

func (s *Service) ListDexes(ctx context.Context, limit int) ([]DexSummary, error) {
	if s.db == nil {
		return nil, fmt.Errorf("postgres pool is required")
	}
	limit = clampLimit(limit, defaultDexesLimit)

	const query = `
SELECT slug, name, volume_24h, volume_7d, volume_30d, change_1d, COALESCE(updated_at, NOW())
FROM defi_dex_latest
ORDER BY volume_24h DESC, slug
LIMIT $1`

	rows, err := s.db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query defi dexes: %w", err)
	}
	defer rows.Close()

	items := make([]DexSummary, 0)
	for rows.Next() {
		var item DexSummary
		var volume24h, volume7d, volume30d, change1D sql.NullFloat64
		if err := rows.Scan(&item.Slug, &item.Name, &volume24h, &volume7d, &volume30d, &change1D, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan defi dex: %w", err)
		}
		item.Volume24HUSD = float64Ptr(volume24h)
		item.Volume7DUSD = float64Ptr(volume7d)
		item.Volume30DUSD = float64Ptr(volume30d)
		item.Change1D = float64Ptr(change1D)
		item.UpdatedAt = item.UpdatedAt.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate defi dexes: %w", err)
	}

	return items, nil
}

func scanProtocolRow(row interface {
	Scan(dest ...any) error
}) (ProtocolSummary, error) {
	var item ProtocolSummary
	var tvl, change1D, change7D sql.NullFloat64
	if err := row.Scan(
		&item.Slug,
		&item.Name,
		&item.Description,
		&item.Logo,
		&item.Category,
		&item.URL,
		&item.Twitter,
		&item.CoinID,
		&item.AuditStatus,
		&item.Oracles,
		&item.UpdatedAt,
		&item.Tier,
		&tvl,
		&change1D,
		&change7D,
	); err != nil {
		return ProtocolSummary{}, fmt.Errorf("scan defi protocol: %w", err)
	}
	item.UpdatedAt = item.UpdatedAt.UTC()
	item.TVLUSD = float64Ptr(tvl)
	item.TVLChange1D = float64Ptr(change1D)
	item.TVLChange7D = float64Ptr(change7D)
	return item, nil
}

func clampLimit(limit, fallback int) int {
	if limit <= 0 {
		return fallback
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func float64Ptr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	value := n.Float64
	return &value
}

func strconvForArg(n int) string {
	return fmt.Sprintf("%d", n)
}
