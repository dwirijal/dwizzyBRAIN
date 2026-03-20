package marketapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"dwizzyBRAIN/engine/market/mapping"
	"dwizzyBRAIN/engine/market/ohlcv"
	engticker "dwizzyBRAIN/engine/market/ticker"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

const (
	defaultListLimit      = 20
	maxListLimit          = 200
	defaultDetailLimit    = 5
	defaultOHLVCLimit     = 200
	defaultTickerLimit    = 20
	defaultTickerStaleAge = 15 * time.Second
)

type Service struct {
	db           *pgxpool.Pool
	ohlcvReader  ohlcvReader
	spreadReader spreadReader
	cache        redis.Cmdable
	now          func() time.Time
}

type Availability struct {
	Tier              string     `json:"tier"`
	OnBinance         bool       `json:"on_binance"`
	OnBybit           bool       `json:"on_bybit"`
	OnOKX             bool       `json:"on_okx"`
	OnKucoin          bool       `json:"on_kucoin"`
	OnGate            bool       `json:"on_gate"`
	OnKraken          bool       `json:"on_kraken"`
	OnMexc            bool       `json:"on_mexc"`
	OnHtx             bool       `json:"on_htx"`
	OnCoinpaprika     bool       `json:"on_coinpaprika"`
	IsDexOnly         bool       `json:"is_dex_only"`
	BinanceVerifiedAt *time.Time `json:"binance_verified_at,omitempty"`
	BybitVerifiedAt   *time.Time `json:"bybit_verified_at,omitempty"`
	AssignedAt        time.Time  `json:"assigned_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ColdData struct {
	Description   string         `json:"description,omitempty"`
	Links         map[string]any `json:"links,omitempty"`
	ATHUSD        *float64       `json:"ath_usd,omitempty"`
	ATLUSD        *float64       `json:"atl_usd,omitempty"`
	ATHDate       *time.Time     `json:"ath_date,omitempty"`
	ATLDate       *time.Time     `json:"atl_date,omitempty"`
	MarketCapRank *int           `json:"market_cap_rank,omitempty"`
	UpdatedAt     *time.Time     `json:"updated_at,omitempty"`
}

type PriceSource struct {
	Exchange string `json:"exchange"`
	Symbol   string `json:"symbol"`
}

type ExchangePrice struct {
	Exchange string   `json:"exchange"`
	Symbol   string   `json:"symbol"`
	Price    *float64 `json:"price,omitempty"`
}

type SnapshotSummary struct {
	CoinID          string       `json:"coin_id"`
	Symbol          string       `json:"symbol"`
	Name            string       `json:"name"`
	ImageURL        string       `json:"image_url,omitempty"`
	Rank            int          `json:"rank"`
	UpdatedAt       time.Time    `json:"updated_at"`
	Availability    Availability `json:"availability"`
	ExchangeCount   int          `json:"exchange_count"`
	CurrentPriceUSD *float64     `json:"current_price_usd,omitempty"`
	MarketCapUSD    *float64     `json:"market_cap_usd,omitempty"`
	PriceSource     *PriceSource `json:"price_source,omitempty"`
}

type MappingSnapshot struct {
	CoinID         string    `json:"coin_id"`
	Exchange       string    `json:"exchange"`
	ExchangeSymbol string    `json:"exchange_symbol"`
	BaseAsset      string    `json:"base_asset"`
	QuoteAsset     string    `json:"quote_asset"`
	IsPrimary      bool      `json:"is_primary"`
	VerifiedAt     time.Time `json:"verified_at"`
}

type ArbitrageSignal struct {
	ID             int64     `json:"id"`
	DetectedAt     time.Time `json:"detected_at"`
	CoinID         string    `json:"coin_id"`
	Symbol         string    `json:"symbol"`
	BuyExchange    string    `json:"buy_exchange"`
	SellExchange   string    `json:"sell_exchange"`
	BuyPrice       float64   `json:"buy_price"`
	SellPrice      float64   `json:"sell_price"`
	GrossSpreadPct float64   `json:"gross_spread_pct"`
	IsProfitable   bool      `json:"is_profitable"`
	BuyDepthUSD    float64   `json:"buy_depth_usd"`
	SellDepthUSD   float64   `json:"sell_depth_usd"`
	Alerted        bool      `json:"alerted"`
}

type SnapshotDetail struct {
	SnapshotSummary
	ColdData        ColdData          `json:"cold_data"`
	Mappings        []MappingSnapshot `json:"mappings"`
	ExchangePrices  []ExchangePrice   `json:"exchange_prices"`
	RecentArbitrage []ArbitrageSignal `json:"recent_arbitrage,omitempty"`
}

type OHLCVPoint = ohlcv.Candle

type TickerExchange struct {
	Exchange  string    `json:"exchange"`
	Symbol    string    `json:"symbol"`
	Price     *float64  `json:"price,omitempty"`
	Bid       *float64  `json:"bid,omitempty"`
	Ask       *float64  `json:"ask,omitempty"`
	Volume24h *float64  `json:"volume_24h,omitempty"`
	SpreadPct *float64  `json:"spread_pct,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	IsStale   bool      `json:"is_stale"`
}

type TickerSnapshot struct {
	CoinID                 string           `json:"coin_id"`
	BestBid                *float64         `json:"best_bid,omitempty"`
	BestBidExchange        string           `json:"best_bid_exchange,omitempty"`
	BestAsk                *float64         `json:"best_ask,omitempty"`
	BestAskExchange        string           `json:"best_ask_exchange,omitempty"`
	CrossExchangeSpreadPct *float64         `json:"cross_exchange_spread_pct,omitempty"`
	ExchangeCount          int              `json:"exchange_count"`
	Exchanges              []TickerExchange `json:"exchanges"`
}

type OrderBookSnapshot = TickerSnapshot

type ohlcvReader interface {
	GetCandles(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]ohlcv.Candle, error)
}

type spreadReader interface {
	LatestByExchange(ctx context.Context, coinID string) ([]engticker.SpreadRecord, error)
}

type listRow struct {
	CoinID        string
	Symbol        string
	Name          string
	ImageURL      string
	Rank          int
	UpdatedAt     time.Time
	Availability  Availability
	ExchangeCount int
	CurrentPriceUSD *float64
	MarketCapUSD    *float64
}

type detailRow struct {
	listRow
	ColdData ColdData
}

type resolvedPrice struct {
	price  *float64
	source *PriceSource
	ok     bool
}

type mappingRow struct {
	CoinID         string
	Exchange       string
	ExchangeSymbol string
	BaseAsset      string
	QuoteAsset     string
	IsPrimary      bool
	VerifiedAt     time.Time
}

type priceCandidate struct {
	key      string
	exchange string
	symbol   string
}

func NewService(db *pgxpool.Pool, ohlcvReader ohlcvReader, spreadReader spreadReader, cache redis.Cmdable) *Service {
	return &Service{
		db:           db,
		ohlcvReader:  ohlcvReader,
		spreadReader: spreadReader,
		cache:        cache,
		now:          time.Now,
	}
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]SnapshotSummary, int, error) {
	if s.db == nil {
		return nil, 0, fmt.Errorf("postgres pool is required")
	}

	limit = clampLimit(limit)
	if offset < 0 {
		offset = 0
	}

	rows, total, err := s.querySummaries(ctx, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	coinIDs := make([]string, 0, len(rows))
	for _, row := range rows {
		coinIDs = append(coinIDs, row.CoinID)
	}

	mappings, err := s.loadMappings(ctx, coinIDs)
	if err != nil {
		return nil, 0, err
	}

	prices := s.resolveBestPrices(ctx, mappings)
	items := make([]SnapshotSummary, 0, len(rows))
	for _, row := range rows {
		item := SnapshotSummary{
			CoinID:        row.CoinID,
			Symbol:        row.Symbol,
			Name:          row.Name,
			ImageURL:      row.ImageURL,
			Rank:          row.Rank,
			UpdatedAt:     row.UpdatedAt,
			Availability:  row.Availability,
			ExchangeCount: len(mappings[row.CoinID]),
		}
		if row.CurrentPriceUSD != nil {
			item.CurrentPriceUSD = row.CurrentPriceUSD
		}
		if row.MarketCapUSD != nil {
			item.MarketCapUSD = row.MarketCapUSD
		}
		if resolved, ok := prices[row.CoinID]; ok {
			item.CurrentPriceUSD = resolved.price
			item.PriceSource = resolved.source
		}
		items = append(items, item)
	}

	return items, total, nil
}

func (s *Service) Detail(ctx context.Context, coinID string) (SnapshotDetail, error) {
	if s.db == nil {
		return SnapshotDetail{}, fmt.Errorf("postgres pool is required")
	}

	coinID = strings.TrimSpace(coinID)
	if coinID == "" {
		return SnapshotDetail{}, fmt.Errorf("coin_id is required")
	}

	row, err := s.queryDetail(ctx, coinID)
	if err != nil {
		return SnapshotDetail{}, err
	}

	mappings, err := s.loadMappings(ctx, []string{row.CoinID})
	if err != nil {
		return SnapshotDetail{}, err
	}
	priceMap := s.resolvePricesByExchange(ctx, mappings[row.CoinID])
	bestPrice, bestSource, ok := pickBestPrice(priceMap, mappings[row.CoinID])

	signalLimit := defaultDetailLimit
	signals, err := s.loadRecentSignals(ctx, row.CoinID, signalLimit)
	if err != nil {
		return SnapshotDetail{}, err
	}

	detail := SnapshotDetail{
		SnapshotSummary: SnapshotSummary{
			CoinID:        row.CoinID,
			Symbol:        row.Symbol,
			Name:          row.Name,
			ImageURL:      row.ImageURL,
			Rank:          row.Rank,
			UpdatedAt:     row.UpdatedAt,
			Availability:  row.Availability,
			ExchangeCount: len(mappings[row.CoinID]),
		},
		ColdData:        row.ColdData,
		Mappings:        convertMappings(mappings[row.CoinID]),
		ExchangePrices:  buildExchangePrices(mappings[row.CoinID], priceMap),
		RecentArbitrage: signals,
	}
	if ok {
		detail.CurrentPriceUSD = bestPrice
		detail.PriceSource = bestSource
	}

	return detail, nil
}

func (s *Service) OHLCV(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]OHLCVPoint, error) {
	if s.ohlcvReader == nil {
		return nil, fmt.Errorf("ohlcv reader is required")
	}

	coinID = strings.TrimSpace(coinID)
	if coinID == "" {
		return nil, fmt.Errorf("coin_id is required")
	}

	exchange = normalizeExchange(exchange)
	timeframe = normalizeTimeframe(timeframe)
	if exchange == "" {
		mappings, err := s.loadMappings(ctx, []string{coinID})
		if err != nil {
			return nil, err
		}
		exchange = primaryExchangeFromMappings(mappings[coinID])
	}
	if exchange == "" {
		return nil, fmt.Errorf("exchange is required")
	}
	if timeframe == "" {
		timeframe = "1m"
	}
	if limit <= 0 {
		limit = defaultOHLVCLimit
	}

	return s.ohlcvReader.GetCandles(ctx, coinID, exchange, timeframe, limit)
}

func (s *Service) Tickers(ctx context.Context, coinID string) (TickerSnapshot, error) {
	coinID = strings.TrimSpace(coinID)
	if coinID == "" {
		return TickerSnapshot{}, fmt.Errorf("coin_id is required")
	}

	mappings, err := s.loadMappings(ctx, []string{coinID})
	if err != nil {
		return TickerSnapshot{}, err
	}
	items := dedupeMappingsByExchange(mappings[coinID])
	if len(items) == 0 {
		return TickerSnapshot{}, fmt.Errorf("coin %q not found", coinID)
	}

	var latest []engticker.SpreadRecord
	if s.spreadReader != nil {
		var spreadErr error
		latest, spreadErr = s.spreadReader.LatestByExchange(ctx, coinID)
		if spreadErr != nil {
			return TickerSnapshot{}, spreadErr
		}
	}
	spreadByExchange := make(map[string]engticker.SpreadRecord, len(latest))
	for _, record := range latest {
		spreadByExchange[normalizeExchange(record.Exchange)] = record
	}
	priceMap := s.resolvePricesByExchange(ctx, items)
	now := s.now().UTC()

	exchanges := make([]TickerExchange, 0, len(items))
	for _, item := range items {
		record, ok := spreadByExchange[normalizeExchange(item.Exchange)]
		var price *float64
		if value := priceMap[priceKey(item.CoinID, item.Exchange)]; value != nil {
			price = value
		} else if value := priceMap[priceKey(item.ExchangeSymbol, item.Exchange)]; value != nil {
			price = value
		}
		var bid, ask, spreadPct, volume *float64
		var ts time.Time
		if ok {
			bid = float64Ptr(record.Bid)
			ask = float64Ptr(record.Ask)
			spreadPct = float64Ptr(record.SpreadPct)
			volume = float64Ptr(record.Volume24h)
			ts = record.Timestamp.UTC()
		}

		exchangeItem := TickerExchange{
			Exchange:  item.Exchange,
			Symbol:    item.ExchangeSymbol,
			Price:     price,
			Bid:       bid,
			Ask:       ask,
			Volume24h: volume,
			SpreadPct: spreadPct,
			Timestamp: ts,
			IsStale:   !ts.IsZero() && ts.Before(now.Add(-defaultTickerStaleAge)),
		}
		exchanges = append(exchanges, exchangeItem)
	}

	sort.Slice(exchanges, func(i, j int) bool {
		return exchangePriority(exchanges[i].Exchange) < exchangePriority(exchanges[j].Exchange)
	})

	return buildTickerSnapshot(coinID, exchanges), nil
}

func (s *Service) OrderBook(ctx context.Context, coinID string) (OrderBookSnapshot, error) {
	return s.Tickers(ctx, coinID)
}

func (s *Service) Arbitrage(ctx context.Context, coinID string, limit int) ([]ArbitrageSignal, error) {
	if s.db == nil {
		return nil, fmt.Errorf("postgres pool is required")
	}
	coinID = strings.TrimSpace(coinID)
	if coinID == "" {
		return nil, fmt.Errorf("coin_id is required")
	}
	if limit <= 0 {
		limit = defaultTickerLimit
	}

	return s.loadRecentSignals(ctx, coinID, limit)
}

func (s *Service) querySummaries(ctx context.Context, limit, offset int) ([]listRow, int, error) {
const countQuery = `
SELECT count(*)
FROM coins
WHERE TRUE`

	var total int
	if err := s.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count coins: %w", err)
	}

const query = `
SELECT
    c.id AS coin_id,
    COALESCE(NULLIF(c.symbol, ''), '') AS symbol,
    COALESCE(NULLIF(c.name, ''), '') AS name,
    COALESCE(NULLIF(c.image_url, ''), '') AS image_url,
    COALESCE(c.rank, 0) AS rank,
    COALESCE(c.updated_at, NOW()) AS updated_at,
    cd.current_price_usd,
    cd.market_cap_usd,
    COALESCE(cc.tier, 'D') AS tier,
    COALESCE(cc.on_binance, FALSE) AS on_binance,
    COALESCE(cc.on_bybit, FALSE) AS on_bybit,
    COALESCE(cc.is_dex_only, FALSE) AS is_dex_only,
    FALSE AS on_okx,
    FALSE AS on_kucoin,
    FALSE AS on_gate,
    FALSE AS on_kraken,
    FALSE AS on_mexc,
    FALSE AS on_htx,
    FALSE AS on_coinpaprika,
    NULL::timestamptz AS binance_verified_at,
    NULL::timestamptz AS bybit_verified_at,
    COALESCE(cc.updated_at, NOW()) AS assigned_at,
    COALESCE(cc.updated_at, NOW()) AS coverage_updated_at
FROM coins c
LEFT JOIN coin_coverage cc ON cc.coin_id = c.id
LEFT JOIN cold_coin_data cd ON cd.coin_id = c.id
WHERE TRUE
ORDER BY COALESCE(c.rank, 2147483647), c.id
LIMIT $1 OFFSET $2`

	rows, err := s.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("query market summaries: %w", err)
	}
	defer rows.Close()

	items := make([]listRow, 0)
	for rows.Next() {
		var row listRow
		var tier string
		var onBinance, onBybit, onOKX, onKucoin, onGate, onKraken, onMexc, onHtx, onCoinpaprika, isDexOnly bool
		var binanceVerifiedAt, bybitVerifiedAt sql.NullTime
		var assignedAt, updatedAt time.Time
		var currentPriceUSD, marketCapUSD sql.NullFloat64

		if err := rows.Scan(
			&row.CoinID,
			&row.Symbol,
			&row.Name,
			&row.ImageURL,
			&row.Rank,
			&row.UpdatedAt,
			&currentPriceUSD,
			&marketCapUSD,
			&tier,
			&onBinance,
			&onBybit,
			&onOKX,
			&onKucoin,
			&onGate,
			&onKraken,
			&onMexc,
			&onHtx,
			&onCoinpaprika,
			&isDexOnly,
			&binanceVerifiedAt,
			&bybitVerifiedAt,
			&assignedAt,
			&updatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan market summary: %w", err)
		}

		row.Availability = Availability{
			Tier:          strings.TrimSpace(tier),
			OnBinance:     onBinance,
			OnBybit:       onBybit,
			OnOKX:         onOKX,
			OnKucoin:      onKucoin,
			OnGate:        onGate,
			OnKraken:      onKraken,
			OnMexc:        onMexc,
			OnHtx:         onHtx,
			OnCoinpaprika: onCoinpaprika,
			IsDexOnly:     isDexOnly,
			AssignedAt:    assignedAt.UTC(),
			UpdatedAt:     updatedAt.UTC(),
		}
		if binanceVerifiedAt.Valid {
			value := binanceVerifiedAt.Time.UTC()
			row.Availability.BinanceVerifiedAt = &value
		}
		if bybitVerifiedAt.Valid {
			value := bybitVerifiedAt.Time.UTC()
			row.Availability.BybitVerifiedAt = &value
		}
		if currentPriceUSD.Valid {
			value := currentPriceUSD.Float64
			row.CurrentPriceUSD = &value
		}
		if marketCapUSD.Valid {
			value := marketCapUSD.Float64
			row.MarketCapUSD = &value
		}

		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate market summaries: %w", err)
	}

	return items, total, nil
}

func (s *Service) queryDetail(ctx context.Context, coinID string) (detailRow, error) {
	const query = `
SELECT
    c.id AS coin_id,
    COALESCE(NULLIF(c.symbol, ''), '') AS symbol,
    COALESCE(NULLIF(c.name, ''), '') AS name,
    COALESCE(NULLIF(c.image_url, ''), '') AS image_url,
    COALESCE(c.rank, 0) AS rank,
    COALESCE(c.updated_at, NOW()) AS updated_at,
    cd.current_price_usd,
    cd.market_cap_usd,
    COALESCE(cc.tier, 'D') AS tier,
    COALESCE(cc.on_binance, FALSE) AS on_binance,
    COALESCE(cc.on_bybit, FALSE) AS on_bybit,
    COALESCE(cc.is_dex_only, FALSE) AS is_dex_only,
    FALSE AS on_okx,
    FALSE AS on_kucoin,
    FALSE AS on_gate,
    FALSE AS on_kraken,
    FALSE AS on_mexc,
    FALSE AS on_htx,
    FALSE AS on_coinpaprika,
    NULL::timestamptz AS binance_verified_at,
    NULL::timestamptz AS bybit_verified_at,
    COALESCE(cc.updated_at, NOW()) AS assigned_at,
    COALESCE(cc.updated_at, NOW()) AS coverage_updated_at,
    COALESCE(cd.description, '') AS description,
    COALESCE(cd.links, '{}'::jsonb) AS links,
    cd.ath,
    cd.atl,
    cd.ath_date,
    cd.atl_date,
    cd.market_cap_rank,
    cd.updated_at AS cold_updated_at
FROM coins c
LEFT JOIN coin_coverage cc ON cc.coin_id = c.id
LEFT JOIN cold_coin_data cd ON cd.coin_id = c.id
WHERE c.id = $1
LIMIT 1`

	var row detailRow
	var tier string
	var onBinance, onBybit, onOKX, onKucoin, onGate, onKraken, onMexc, onHtx, onCoinpaprika, isDexOnly bool
	var binanceVerifiedAt, bybitVerifiedAt sql.NullTime
	var assignedAt, updatedAt time.Time
	var description string
	var linksRaw []byte
	var ath, atl sql.NullFloat64
	var athDate, atlDate, coldUpdatedAt sql.NullTime
	var marketCapRank sql.NullInt64
	var currentPriceUSD, marketCapUSD sql.NullFloat64

	err := s.db.QueryRow(ctx, query, coinID).Scan(
		&row.CoinID,
		&row.Symbol,
		&row.Name,
		&row.ImageURL,
		&row.Rank,
		&row.UpdatedAt,
		&currentPriceUSD,
		&marketCapUSD,
		&tier,
		&onBinance,
		&onBybit,
		&onOKX,
		&onKucoin,
		&onGate,
		&onKraken,
		&onMexc,
		&onHtx,
		&onCoinpaprika,
		&isDexOnly,
		&binanceVerifiedAt,
		&bybitVerifiedAt,
		&assignedAt,
		&updatedAt,
		&description,
		&linksRaw,
		&ath,
		&atl,
		&athDate,
		&atlDate,
		&marketCapRank,
		&coldUpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return detailRow{}, fmt.Errorf("coin %q not found", coinID)
		}
		return detailRow{}, fmt.Errorf("query market detail: %w", err)
	}

	row.Availability = Availability{
		Tier:          strings.TrimSpace(tier),
		OnBinance:     onBinance,
		OnBybit:       onBybit,
		OnOKX:         onOKX,
		OnKucoin:      onKucoin,
		OnGate:        onGate,
		OnKraken:      onKraken,
		OnMexc:        onMexc,
		OnHtx:         onHtx,
		OnCoinpaprika: onCoinpaprika,
		IsDexOnly:     isDexOnly,
		AssignedAt:    assignedAt.UTC(),
		UpdatedAt:     updatedAt.UTC(),
	}
	if binanceVerifiedAt.Valid {
		value := binanceVerifiedAt.Time.UTC()
		row.Availability.BinanceVerifiedAt = &value
	}
	if bybitVerifiedAt.Valid {
		value := bybitVerifiedAt.Time.UTC()
		row.Availability.BybitVerifiedAt = &value
	}
	if currentPriceUSD.Valid {
		value := currentPriceUSD.Float64
		row.CurrentPriceUSD = &value
	}
	if marketCapUSD.Valid {
		value := marketCapUSD.Float64
		row.MarketCapUSD = &value
	}

	row.ColdData = ColdData{
		Description: description,
	}
	if len(linksRaw) > 0 {
		var links map[string]any
		if err := json.Unmarshal(linksRaw, &links); err == nil {
			row.ColdData.Links = links
		}
	}
	if ath.Valid {
		value := ath.Float64
		row.ColdData.ATHUSD = &value
	}
	if atl.Valid {
		value := atl.Float64
		row.ColdData.ATLUSD = &value
	}
	if athDate.Valid {
		value := athDate.Time.UTC()
		row.ColdData.ATHDate = &value
	}
	if atlDate.Valid {
		value := atlDate.Time.UTC()
		row.ColdData.ATLDate = &value
	}
	if marketCapRank.Valid {
		value := int(marketCapRank.Int64)
		row.ColdData.MarketCapRank = &value
	}
	if coldUpdatedAt.Valid {
		value := coldUpdatedAt.Time.UTC()
		row.ColdData.UpdatedAt = &value
	}

	return row, nil
}

func (s *Service) loadMappings(ctx context.Context, coinIDs []string) (map[string][]mapping.Mapping, error) {
	out := make(map[string][]mapping.Mapping)
	if len(coinIDs) == 0 {
		return out, nil
	}

	const query = `
SELECT coin_id, exchange, exchange_symbol, base_asset, quote_asset, is_primary, COALESCE(verified_at, TIMESTAMPTZ 'epoch')
FROM coin_exchange_mappings
WHERE status = 'active'
  AND coin_id = ANY($1::text[])
ORDER BY coin_id, is_primary DESC, exchange ASC, exchange_symbol ASC`

	rows, err := s.db.Query(ctx, query, coinIDs)
	if err != nil {
		return nil, fmt.Errorf("query mappings: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var row mappingRow
		if err := rows.Scan(
			&row.CoinID,
			&row.Exchange,
			&row.ExchangeSymbol,
			&row.BaseAsset,
			&row.QuoteAsset,
			&row.IsPrimary,
			&row.VerifiedAt,
		); err != nil {
			return nil, fmt.Errorf("scan mapping row: %w", err)
		}

		out[row.CoinID] = append(out[row.CoinID], mapping.Mapping{
			CoinID:         row.CoinID,
			Exchange:       row.Exchange,
			ExchangeSymbol: row.ExchangeSymbol,
			BaseAsset:      row.BaseAsset,
			QuoteAsset:     row.QuoteAsset,
			IsPrimary:      row.IsPrimary,
			VerifiedAt:     row.VerifiedAt.UTC(),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate mappings: %w", err)
	}

	for _, coinID := range coinIDs {
		if out[coinID] == nil {
			out[coinID] = []mapping.Mapping{}
		}
		sortMappings(out[coinID])
	}

	return out, nil
}

func (s *Service) loadRecentSignals(ctx context.Context, coinID string, limit int) ([]ArbitrageSignal, error) {
	if limit <= 0 {
		limit = defaultDetailLimit
	}

	const query = `
SELECT id, detected_at, coin_id, symbol, buy_exchange, sell_exchange, buy_price, sell_price,
       gross_spread_pct, is_profitable, buy_depth_usd, sell_depth_usd, alerted
FROM arbitrage_signals
WHERE coin_id = $1
ORDER BY detected_at DESC
LIMIT $2`

	rows, err := s.db.Query(ctx, query, coinID, limit)
	if err != nil {
		return nil, fmt.Errorf("query arbitrage signals: %w", err)
	}
	defer rows.Close()

	signals := make([]ArbitrageSignal, 0)
	for rows.Next() {
		var signal ArbitrageSignal
		if err := rows.Scan(
			&signal.ID,
			&signal.DetectedAt,
			&signal.CoinID,
			&signal.Symbol,
			&signal.BuyExchange,
			&signal.SellExchange,
			&signal.BuyPrice,
			&signal.SellPrice,
			&signal.GrossSpreadPct,
			&signal.IsProfitable,
			&signal.BuyDepthUSD,
			&signal.SellDepthUSD,
			&signal.Alerted,
		); err != nil {
			return nil, fmt.Errorf("scan arbitrage signal: %w", err)
		}
		signals = append(signals, signal)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate arbitrage signals: %w", err)
	}

	return signals, nil
}

func (s *Service) resolveBestPrices(ctx context.Context, mappings map[string][]mapping.Mapping) map[string]resolvedPrice {
	result := make(map[string]resolvedPrice)
	for coinID, items := range mappings {
		prices := s.resolvePricesByExchange(ctx, items)
		if price, source, ok := pickBestPrice(prices, items); ok {
			result[coinID] = resolvedPrice{
				price:  price,
				source: source,
				ok:     true,
			}
		}
	}
	return result
}

func (s *Service) resolvePricesByExchange(ctx context.Context, items []mapping.Mapping) map[string]*float64 {
	prices := make(map[string]*float64)
	if len(items) == 0 || s.cache == nil {
		return prices
	}

	keys := make([]string, 0, len(items)*2)
	seen := make(map[string]struct{}, len(items)*2)
	for _, item := range items {
		for _, key := range []string{
			priceKey(item.CoinID, item.Exchange),
			priceKey(item.ExchangeSymbol, item.Exchange),
		} {
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}

	values, err := s.cache.MGet(ctx, keys...).Result()
	if err != nil {
		return prices
	}
	for i, value := range values {
		if value == nil {
			continue
		}
		str, ok := value.(string)
		if !ok {
			continue
		}
		if price, err := strconv.ParseFloat(strings.TrimSpace(str), 64); err == nil {
			value := price
			prices[keys[i]] = &value
		}
	}

	return prices
}

func pickBestPrice(priceMap map[string]*float64, items []mapping.Mapping) (*float64, *PriceSource, bool) {
	if len(items) == 0 {
		return nil, nil, false
	}

	sorted := append([]mapping.Mapping(nil), items...)
	sortMappings(sorted)

	for _, item := range sorted {
		if price := priceMap[priceKey(item.CoinID, item.Exchange)]; price != nil {
			return price, &PriceSource{Exchange: item.Exchange, Symbol: item.ExchangeSymbol}, true
		}
		if price := priceMap[priceKey(item.ExchangeSymbol, item.Exchange)]; price != nil {
			return price, &PriceSource{Exchange: item.Exchange, Symbol: item.ExchangeSymbol}, true
		}
	}

	return nil, nil, false
}

func buildExchangePrices(items []mapping.Mapping, priceMap map[string]*float64) []ExchangePrice {
	if len(items) == 0 {
		return nil
	}

	sorted := append([]mapping.Mapping(nil), items...)
	sortMappings(sorted)
	out := make([]ExchangePrice, 0, len(sorted))
	for _, item := range sorted {
		var price *float64
		if value := priceMap[priceKey(item.CoinID, item.Exchange)]; value != nil {
			price = value
		} else if value := priceMap[priceKey(item.ExchangeSymbol, item.Exchange)]; value != nil {
			price = value
		}
		out = append(out, ExchangePrice{
			Exchange: item.Exchange,
			Symbol:   item.ExchangeSymbol,
			Price:    price,
		})
	}

	return out
}

func buildTickerSnapshot(coinID string, exchanges []TickerExchange) TickerSnapshot {
	snapshot := TickerSnapshot{
		CoinID:    coinID,
		Exchanges: exchanges,
	}

	for _, exchange := range exchanges {
		if exchange.Bid != nil {
			if snapshot.BestBid == nil || *exchange.Bid > *snapshot.BestBid {
				snapshot.BestBid = exchange.Bid
				snapshot.BestBidExchange = exchange.Exchange
			}
		}
		if exchange.Ask != nil {
			if snapshot.BestAsk == nil || *exchange.Ask < *snapshot.BestAsk {
				snapshot.BestAsk = exchange.Ask
				snapshot.BestAskExchange = exchange.Exchange
			}
		}
	}

	snapshot.ExchangeCount = len(exchanges)
	if snapshot.BestBid != nil && snapshot.BestAsk != nil && *snapshot.BestAsk > 0 {
		value := ((*snapshot.BestBid - *snapshot.BestAsk) / *snapshot.BestAsk) * 100
		snapshot.CrossExchangeSpreadPct = &value
	}

	return snapshot
}

func dedupeMappingsByExchange(items []mapping.Mapping) []mapping.Mapping {
	if len(items) == 0 {
		return nil
	}

	unique := make(map[string]mapping.Mapping, len(items))
	order := make([]string, 0, len(items))
	for _, item := range items {
		key := normalizeExchange(item.Exchange)
		current, ok := unique[key]
		if !ok {
			unique[key] = item
			order = append(order, key)
			continue
		}
		if !current.IsPrimary && item.IsPrimary {
			unique[key] = item
		}
	}

	out := make([]mapping.Mapping, 0, len(unique))
	for _, key := range order {
		out = append(out, unique[key])
	}

	sortMappings(out)
	return out
}

func convertMappings(items []mapping.Mapping) []MappingSnapshot {
	if len(items) == 0 {
		return nil
	}

	out := make([]MappingSnapshot, 0, len(items))
	for _, item := range items {
		out = append(out, MappingSnapshot{
			CoinID:         item.CoinID,
			Exchange:       item.Exchange,
			ExchangeSymbol: item.ExchangeSymbol,
			BaseAsset:      item.BaseAsset,
			QuoteAsset:     item.QuoteAsset,
			IsPrimary:      item.IsPrimary,
			VerifiedAt:     item.VerifiedAt.UTC(),
		})
	}

	return out
}

func sortMappings(items []mapping.Mapping) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsPrimary != items[j].IsPrimary {
			return items[i].IsPrimary
		}
		if exchangePriority(items[i].Exchange) != exchangePriority(items[j].Exchange) {
			return exchangePriority(items[i].Exchange) < exchangePriority(items[j].Exchange)
		}
		if items[i].Exchange != items[j].Exchange {
			return items[i].Exchange < items[j].Exchange
		}
		return strings.ToUpper(strings.TrimSpace(items[i].ExchangeSymbol)) < strings.ToUpper(strings.TrimSpace(items[j].ExchangeSymbol))
	})
}

func exchangePriority(exchange string) int {
	switch strings.ToLower(strings.TrimSpace(exchange)) {
	case "binance":
		return 0
	case "bybit":
		return 1
	case "okx":
		return 2
	case "kucoin":
		return 3
	case "gateio":
		return 4
	case "kraken":
		return 5
	case "mexc":
		return 6
	case "htx":
		return 7
	default:
		return 100
	}
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return defaultListLimit
	case limit > maxListLimit:
		return maxListLimit
	default:
		return limit
	}
}

func priceKey(symbol, exchange string) string {
	symbol = strings.TrimSpace(symbol)
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	if symbol == "" || exchange == "" {
		return ""
	}
	return "price:" + symbol + ":" + exchange
}

func primaryExchangeFromMappings(items []mapping.Mapping) string {
	for _, item := range items {
		if item.IsPrimary {
			return normalizeExchange(item.Exchange)
		}
	}
	if len(items) > 0 {
		return normalizeExchange(items[0].Exchange)
	}
	return ""
}

func float64Ptr(value float64) *float64 {
	v := value
	return &v
}

func normalizeExchange(exchange string) string {
	return strings.ToLower(strings.TrimSpace(exchange))
}

func normalizeTimeframe(timeframe string) string {
	return strings.ToLower(strings.TrimSpace(timeframe))
}
