package quantapi

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultPatternLimit     = 20
	maxPatternLimit         = 50
	defaultPatternMinMatches = 30
)

type Service struct {
	db *pgxpool.Pool
}

type PatternQuery struct {
	Symbol          string    `json:"symbol"`
	Timeframe       string    `json:"timeframe"`
	Time            time.Time `json:"time"`
	Fingerprint     []float64 `json:"fingerprint"`
	MacroEnvironment string    `json:"macro_environment,omitempty"`
}

type PatternMatch struct {
	Time              time.Time  `json:"time"`
	Symbol            string     `json:"symbol"`
	Timeframe         string     `json:"timeframe"`
	Close             *float64   `json:"close,omitempty"`
	MacroEnvironment  string     `json:"macro_environment,omitempty"`
	ProximityLabel    string     `json:"proximity_label,omitempty"`
	RateDirection     string     `json:"rate_direction,omitempty"`
	CpiTrend          string     `json:"cpi_trend,omitempty"`
	LastSurpriseLabel string     `json:"last_surprise_label,omitempty"`
	LastSurpriseValue *float64   `json:"last_surprise_value,omitempty"`
	SimilarityScore   float64    `json:"similarity_score"`
	Close1hLater      *float64   `json:"close_1h_later,omitempty"`
	Close4hLater      *float64   `json:"close_4h_later,omitempty"`
	Close1dLater      *float64   `json:"close_1d_later,omitempty"`
	Close1wLater      *float64   `json:"close_1w_later,omitempty"`
}

type OutcomeStats struct {
	Count   int       `json:"count"`
	Median  *float64  `json:"median,omitempty"`
	WinRate *float64  `json:"win_rate,omitempty"`
	AvgWin  *float64  `json:"avg_win,omitempty"`
	AvgLoss *float64  `json:"avg_loss,omitempty"`
}

type PatternResponse struct {
	LowConfidence bool                    `json:"low_confidence"`
	Query         PatternQuery            `json:"query"`
	Matches       []PatternMatch          `json:"matches"`
	Outcomes      map[string]OutcomeStats  `json:"outcomes"`
}

type NotFoundError struct {
	message string
}

func (e NotFoundError) Error() string {
	return e.message
}

func (e NotFoundError) NotFound() bool {
	return true
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

func (s *Service) Pattern(ctx context.Context, symbol, timeframe string, limit, minMatches int) (PatternResponse, error) {
	if s.db == nil {
		return PatternResponse{}, fmt.Errorf("postgres pool is required")
	}

	symbol = strings.TrimSpace(symbol)
	timeframe = strings.ToLower(strings.TrimSpace(timeframe))
	if symbol == "" {
		return PatternResponse{}, fmt.Errorf("symbol is required")
	}
	if timeframe == "" {
		return PatternResponse{}, fmt.Errorf("timeframe is required")
	}

	limit = clampPatternLimit(limit)
	if minMatches <= 0 {
		minMatches = defaultPatternMinMatches
	}

	source, err := s.loadQuerySource(ctx, symbol, timeframe)
	if err != nil {
		return PatternResponse{}, err
	}

	matches, err := s.findMatches(ctx, symbol, timeframe, source.EmbeddingText, limit)
	if err != nil {
		return PatternResponse{}, err
	}

	return PatternResponse{
		LowConfidence: len(matches) < minMatches,
		Query: PatternQuery{
			Symbol:          symbol,
			Timeframe:       timeframe,
			Time:            source.Time,
			Fingerprint:     decodeVectorLiteral(source.EmbeddingText),
			MacroEnvironment: source.MacroEnvironment,
		},
		Matches:  matches,
		Outcomes: summarizeOutcomes(matches),
	}, nil
}

type querySource struct {
	Time             time.Time
	EmbeddingText    string
	MacroEnvironment string
}

func (s *Service) loadQuerySource(ctx context.Context, symbol, timeframe string) (querySource, error) {
	const query = `
SELECT
    emb.time,
    emb.embedding::text AS embedding_text,
    COALESCE(cel.macro_environment, '') AS macro_environment
FROM candle_embeddings emb
LEFT JOIN candle_event_labels cel
    ON cel.time = emb.time
   AND cel.symbol = emb.symbol
   AND cel.timeframe = emb.timeframe
WHERE emb.symbol = $1
  AND emb.timeframe = $2
ORDER BY emb.time DESC
LIMIT 1
`
	var out querySource
	err := s.db.QueryRow(ctx, query, symbol, timeframe).Scan(&out.Time, &out.EmbeddingText, &out.MacroEnvironment)
	if err != nil {
		if err == pgx.ErrNoRows {
			return querySource{}, NotFoundError{message: fmt.Sprintf("pattern source not found for %s %s", symbol, timeframe)}
		}
		return querySource{}, fmt.Errorf("load pattern source: %w", err)
	}
	if strings.TrimSpace(out.EmbeddingText) == "" {
		return querySource{}, NotFoundError{message: fmt.Sprintf("pattern source not found for %s %s", symbol, timeframe)}
	}
	return out, nil
}

func (s *Service) findMatches(ctx context.Context, symbol, timeframe, embeddingText string, limit int) ([]PatternMatch, error) {
	const query = `
WITH ranked AS (
    SELECT
        emb.time,
        emb.symbol,
        emb.timeframe,
        ohlcv.close,
        COALESCE(cel.macro_environment, '') AS macro_environment,
        COALESCE(cel.proximity_label, '') AS proximity_label,
        COALESCE(cel.rate_direction, '') AS rate_direction,
        COALESCE(cel.cpi_trend, '') AS cpi_trend,
        COALESCE(cel.last_surprise_label, '') AS last_surprise_label,
        cel.last_surprise_value,
        1 - (emb.embedding <=> $3::vector) AS similarity_score
    FROM candle_embeddings emb
    JOIN ohlcv
        ON ohlcv.time = emb.time
       AND ohlcv.symbol = emb.symbol
       AND ohlcv.timeframe = emb.timeframe
    LEFT JOIN candle_event_labels cel
        ON cel.time = emb.time
       AND cel.symbol = emb.symbol
       AND cel.timeframe = emb.timeframe
    WHERE emb.symbol = $1
      AND emb.timeframe = $2
    ORDER BY emb.embedding <=> $3::vector
    LIMIT $4
)
SELECT
    ranked.time,
    ranked.symbol,
    ranked.timeframe,
    ranked.close,
    ranked.macro_environment,
    ranked.proximity_label,
    ranked.rate_direction,
    ranked.cpi_trend,
    ranked.last_surprise_label,
    ranked.last_surprise_value,
    ranked.similarity_score,
    o1.close AS close_1h_later,
    o4.close AS close_4h_later,
    o1d.close AS close_1d_later,
    o1w.close AS close_1w_later
FROM ranked
LEFT JOIN ohlcv o1
    ON o1.symbol = ranked.symbol
   AND o1.timeframe = ranked.timeframe
   AND o1.time = ranked.time + INTERVAL '1 hour'
LEFT JOIN ohlcv o4
    ON o4.symbol = ranked.symbol
   AND o4.timeframe = ranked.timeframe
   AND o4.time = ranked.time + INTERVAL '4 hours'
LEFT JOIN ohlcv o1d
    ON o1d.symbol = ranked.symbol
   AND o1d.timeframe = ranked.timeframe
   AND o1d.time = ranked.time + INTERVAL '1 day'
LEFT JOIN ohlcv o1w
    ON o1w.symbol = ranked.symbol
   AND o1w.timeframe = ranked.timeframe
   AND o1w.time = ranked.time + INTERVAL '1 week'
ORDER BY ranked.similarity_score DESC, ranked.time DESC
`
	rows, err := s.db.Query(ctx, query, symbol, timeframe, embeddingText, limit)
	if err != nil {
		return nil, fmt.Errorf("query pattern matches: %w", err)
	}
	defer rows.Close()

	matches := make([]PatternMatch, 0)
	for rows.Next() {
		var row PatternMatch
		var closeValue float64
		var lastSurpriseValue sql.NullFloat64
		var close1h sql.NullFloat64
		var close4h sql.NullFloat64
		var close1d sql.NullFloat64
		var close1w sql.NullFloat64
		if err := rows.Scan(
			&row.Time,
			&row.Symbol,
			&row.Timeframe,
			&closeValue,
			&row.MacroEnvironment,
			&row.ProximityLabel,
			&row.RateDirection,
			&row.CpiTrend,
			&row.LastSurpriseLabel,
			&lastSurpriseValue,
			&row.SimilarityScore,
			&close1h,
			&close4h,
			&close1d,
			&close1w,
		); err != nil {
			return nil, fmt.Errorf("scan pattern match: %w", err)
		}
		row.Close = &closeValue
		if lastSurpriseValue.Valid {
			value := lastSurpriseValue.Float64
			row.LastSurpriseValue = &value
		}
		if close1h.Valid {
			value := close1h.Float64
			row.Close1hLater = &value
		}
		if close4h.Valid {
			value := close4h.Float64
			row.Close4hLater = &value
		}
		if close1d.Valid {
			value := close1d.Float64
			row.Close1dLater = &value
		}
		if close1w.Valid {
			value := close1w.Float64
			row.Close1wLater = &value
		}
		matches = append(matches, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pattern matches: %w", err)
	}
	return matches, nil
}

func summarizeOutcomes(matches []PatternMatch) map[string]OutcomeStats {
	return map[string]OutcomeStats{
		"1h": summarizeHorizon(matches, func(m PatternMatch) *float64 { return m.Close1hLater }),
		"4h": summarizeHorizon(matches, func(m PatternMatch) *float64 { return m.Close4hLater }),
		"1d": summarizeHorizon(matches, func(m PatternMatch) *float64 { return m.Close1dLater }),
		"1w": summarizeHorizon(matches, func(m PatternMatch) *float64 { return m.Close1wLater }),
	}
}

func summarizeHorizon(matches []PatternMatch, selector func(PatternMatch) *float64) OutcomeStats {
	return summarizeReturns(matches, selector)
}

func summarizeReturns(matches []PatternMatch, selector func(PatternMatch) *float64) OutcomeStats {
	values := make([]float64, 0)
	for _, match := range matches {
		if match.Close == nil {
			continue
		}
		future := selector(match)
		if future == nil {
			continue
		}
		closeValue := *match.Close
		if closeValue == 0 {
			continue
		}
		values = append(values, ((*future-closeValue)/closeValue)*100.0)
	}
	if len(values) == 0 {
		return OutcomeStats{Count: 0}
	}

	sumWins := 0.0
	sumLosses := 0.0
	wins := 0
	losses := 0
	for _, value := range values {
		if value > 0 {
			sumWins += value
			wins++
		} else {
			sumLosses += value
			losses++
		}
	}
	median := median(values)
	winRate := float64(wins) / float64(len(values))

	stats := OutcomeStats{
		Count:   len(values),
		Median:  &median,
		WinRate: &winRate,
	}
	if wins > 0 {
		avgWin := sumWins / float64(wins)
		stats.AvgWin = &avgWin
	}
	if losses > 0 {
		avgLoss := sumLosses / float64(losses)
		stats.AvgLoss = &avgLoss
	}
	return stats
}

func clampPatternLimit(limit int) int {
	if limit <= 0 {
		return defaultPatternLimit
	}
	if limit > maxPatternLimit {
		return maxPatternLimit
	}
	return limit
}

func decodeVectorLiteral(text string) []float64 {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "[")
	text = strings.TrimSuffix(text, "]")
	if text == "" {
		return nil
	}
	parts := strings.Split(text, ",")
	values := make([]float64, 0, len(parts))
	for _, part := range parts {
		parsed, err := strconv.ParseFloat(strings.TrimSpace(part), 64)
		if err != nil {
			continue
		}
		if math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			continue
		}
		values = append(values, parsed)
	}
	return values
}

func median(values []float64) float64 {
	sorted := append([]float64(nil), values...)
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j-1] > sorted[j]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2.0
}
