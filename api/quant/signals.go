package quantapi

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	defaultSignalHistoryLimit = 20
	maxSignalHistoryLimit     = 200
	defaultSignalSummaryLimit = 50
	maxSignalSummaryLimit     = 200
)

type SignalQuery struct {
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`
	Exchange  string `json:"exchange,omitempty"`
}

type SignalRecord struct {
	ID               int64     `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	CoinID           string    `json:"coin_id"`
	Exchange         string    `json:"exchange"`
	Symbol           string    `json:"symbol"`
	Timeframe        string    `json:"timeframe"`
	QuantScore       *float64  `json:"quant_score,omitempty"`
	SignalType       string    `json:"signal_type"`
	Strength         string    `json:"strength,omitempty"`
	RSI14            *float64  `json:"rsi_14,omitempty"`
	MACDLine         *float64  `json:"macd_line,omitempty"`
	MACDSignal       *float64  `json:"macd_signal,omitempty"`
	MACDHist         *float64  `json:"macd_hist,omitempty"`
	BBUpper          *float64  `json:"bb_upper,omitempty"`
	BBLower          *float64  `json:"bb_lower,omitempty"`
	BBMid            *float64  `json:"bb_mid,omitempty"`
	EMA9             *float64  `json:"ema_9,omitempty"`
	EMA21            *float64  `json:"ema_21,omitempty"`
	EMA200           *float64  `json:"ema_200,omitempty"`
	FundingRate      *float64  `json:"funding_rate,omitempty"`
	FundingSentiment string    `json:"funding_sentiment,omitempty"`
	VolumeSpike      bool      `json:"volume_spike"`
	PriceDeviation   bool      `json:"price_deviation"`
	AnomalyScore     *float64  `json:"anomaly_score,omitempty"`
	PriceAtSignal    *float64  `json:"price_at_signal,omitempty"`
}

type SignalLatestResponse struct {
	Query  SignalQuery  `json:"query"`
	Signal SignalRecord `json:"signal"`
}

type SignalHistoryResponse struct {
	Query SignalQuery    `json:"query"`
	Limit int            `json:"limit"`
	Items []SignalRecord `json:"items"`
}

type SignalSummaryResponse struct {
	Query              SignalQuery    `json:"query"`
	Limit              int            `json:"limit"`
	Count              int            `json:"count"`
	Latest             *SignalRecord  `json:"latest,omitempty"`
	AvgQuantScore      *float64       `json:"avg_quant_score,omitempty"`
	AvgFundingRate     *float64       `json:"avg_funding_rate,omitempty"`
	AvgAnomalyScore    *float64       `json:"avg_anomaly_score,omitempty"`
	VolumeSpikeRate    *float64       `json:"volume_spike_rate,omitempty"`
	PriceDeviationRate *float64       `json:"price_deviation_rate,omitempty"`
	SignalTypeCounts   map[string]int `json:"signal_type_counts"`
	StrengthCounts     map[string]int `json:"strength_counts,omitempty"`
}

type signalRow struct {
	ID               int64
	CreatedAt        time.Time
	CoinID           string
	Exchange         string
	Symbol           string
	Timeframe        string
	QuantScore       sql.NullFloat64
	SignalType       string
	Strength         sql.NullString
	RSI14            sql.NullFloat64
	MACDLine         sql.NullFloat64
	MACDSignal       sql.NullFloat64
	MACDHist         sql.NullFloat64
	BBUpper          sql.NullFloat64
	BBLower          sql.NullFloat64
	BBMid            sql.NullFloat64
	EMA9             sql.NullFloat64
	EMA21            sql.NullFloat64
	EMA200           sql.NullFloat64
	FundingRate      sql.NullFloat64
	FundingSentiment sql.NullString
	VolumeSpike      bool
	PriceDeviation   bool
	AnomalyScore     sql.NullFloat64
	PriceAtSignal    sql.NullFloat64
}

func (s *Service) SignalLatest(ctx context.Context, symbol, timeframe, exchange string) (SignalLatestResponse, error) {
	records, err := s.loadSignals(ctx, symbol, timeframe, exchange, 1)
	if err != nil {
		return SignalLatestResponse{}, err
	}
	if len(records) == 0 {
		return SignalLatestResponse{}, NotFoundError{message: fmt.Sprintf("signal not found for %s %s", symbol, timeframe)}
	}
	return SignalLatestResponse{
		Query:  SignalQuery{Symbol: symbol, Timeframe: timeframe, Exchange: exchange},
		Signal: records[0],
	}, nil
}

func (s *Service) SignalHistory(ctx context.Context, symbol, timeframe, exchange string, limit int) (SignalHistoryResponse, error) {
	limit = clampSignalHistoryLimit(limit)
	records, err := s.loadSignals(ctx, symbol, timeframe, exchange, limit)
	if err != nil {
		return SignalHistoryResponse{}, err
	}
	if len(records) == 0 {
		return SignalHistoryResponse{}, NotFoundError{message: fmt.Sprintf("signal history not found for %s %s", symbol, timeframe)}
	}
	return SignalHistoryResponse{
		Query: SignalQuery{Symbol: symbol, Timeframe: timeframe, Exchange: exchange},
		Limit: limit,
		Items: records,
	}, nil
}

func (s *Service) SignalSummary(ctx context.Context, symbol, timeframe, exchange string, limit int) (SignalSummaryResponse, error) {
	limit = clampSignalSummaryLimit(limit)
	records, err := s.loadSignals(ctx, symbol, timeframe, exchange, limit)
	if err != nil {
		return SignalSummaryResponse{}, err
	}
	if len(records) == 0 {
		return SignalSummaryResponse{}, NotFoundError{message: fmt.Sprintf("signal summary not found for %s %s", symbol, timeframe)}
	}
	return summarizeSignalHistory(records, symbol, timeframe, exchange, limit), nil
}

func (s *Service) loadSignals(ctx context.Context, symbol, timeframe, exchange string, limit int) ([]SignalRecord, error) {
	if s.db == nil {
		return nil, fmt.Errorf("postgres pool is required")
	}

	symbol = strings.TrimSpace(symbol)
	timeframe = strings.ToLower(strings.TrimSpace(timeframe))
	exchange = strings.ToLower(strings.TrimSpace(exchange))
	if symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if timeframe == "" {
		return nil, fmt.Errorf("timeframe is required")
	}
	limit = max(1, limit)

	where := []string{"symbol = $1", "timeframe = $2"}
	args := []any{symbol, timeframe}
	if exchange != "" {
		where = append(where, fmt.Sprintf("LOWER(exchange) = LOWER($%d)", len(args)+1))
		args = append(args, exchange)
	}
	args = append(args, limit)

	const baseQuery = `
SELECT
    id,
    created_at,
    coin_id,
    exchange,
    symbol,
    timeframe,
    quant_score,
    signal_type,
    strength,
    rsi_14,
    macd_line,
    macd_signal,
    macd_hist,
    bb_upper,
    bb_lower,
    bb_mid,
    ema_9,
    ema_21,
    ema_200,
    funding_rate,
    funding_sentiment,
    volume_spike,
    price_deviation,
    anomaly_score,
    price_at_signal
FROM signals
WHERE %s
ORDER BY created_at DESC, id DESC
LIMIT $%d
`
	query := fmt.Sprintf(baseQuery, strings.Join(where, " AND "), len(args))

	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query signals: %w", err)
	}
	defer rows.Close()

	records := make([]SignalRecord, 0)
	for rows.Next() {
		var row signalRow
		if err := rows.Scan(
			&row.ID,
			&row.CreatedAt,
			&row.CoinID,
			&row.Exchange,
			&row.Symbol,
			&row.Timeframe,
			&row.QuantScore,
			&row.SignalType,
			&row.Strength,
			&row.RSI14,
			&row.MACDLine,
			&row.MACDSignal,
			&row.MACDHist,
			&row.BBUpper,
			&row.BBLower,
			&row.BBMid,
			&row.EMA9,
			&row.EMA21,
			&row.EMA200,
			&row.FundingRate,
			&row.FundingSentiment,
			&row.VolumeSpike,
			&row.PriceDeviation,
			&row.AnomalyScore,
			&row.PriceAtSignal,
		); err != nil {
			return nil, fmt.Errorf("scan signals: %w", err)
		}
		records = append(records, row.toRecord())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate signals: %w", err)
	}
	return records, nil
}

func summarizeSignalHistory(records []SignalRecord, symbol, timeframe, exchange string, limit int) SignalSummaryResponse {
	quantScores := make([]float64, 0, len(records))
	fundingRates := make([]float64, 0, len(records))
	anomalyScores := make([]float64, 0, len(records))
	volumeSpikeCount := 0
	priceDeviationCount := 0
	signalTypeCounts := make(map[string]int)
	strengthCounts := make(map[string]int)

	for _, record := range records {
		signalTypeCounts[record.SignalType]++
		if strings.TrimSpace(record.Strength) != "" {
			strengthCounts[record.Strength]++
		}
		if record.VolumeSpike {
			volumeSpikeCount++
		}
		if record.PriceDeviation {
			priceDeviationCount++
		}
		if record.QuantScore != nil {
			quantScores = append(quantScores, *record.QuantScore)
		}
		if record.FundingRate != nil {
			fundingRates = append(fundingRates, *record.FundingRate)
		}
		if record.AnomalyScore != nil {
			anomalyScores = append(anomalyScores, *record.AnomalyScore)
		}
	}

	summary := SignalSummaryResponse{
		Query:            SignalQuery{Symbol: symbol, Timeframe: timeframe, Exchange: exchange},
		Limit:            limit,
		Count:            len(records),
		Latest:           &records[0],
		SignalTypeCounts: signalTypeCounts,
	}
	summary.AvgQuantScore = avgFloat64(quantScores)
	summary.AvgFundingRate = avgFloat64(fundingRates)
	summary.AvgAnomalyScore = avgFloat64(anomalyScores)
	summary.VolumeSpikeRate = ratioFloat64(volumeSpikeCount, len(records))
	summary.PriceDeviationRate = ratioFloat64(priceDeviationCount, len(records))
	if len(strengthCounts) > 0 {
		summary.StrengthCounts = strengthCounts
	}
	return summary
}

func clampSignalHistoryLimit(limit int) int {
	if limit <= 0 {
		return defaultSignalHistoryLimit
	}
	if limit > maxSignalHistoryLimit {
		return maxSignalHistoryLimit
	}
	return limit
}

func clampSignalSummaryLimit(limit int) int {
	if limit <= 0 {
		return defaultSignalSummaryLimit
	}
	if limit > maxSignalSummaryLimit {
		return maxSignalSummaryLimit
	}
	return limit
}

func (r signalRow) toRecord() SignalRecord {
	record := SignalRecord{
		ID:               r.ID,
		CreatedAt:        r.CreatedAt,
		CoinID:           r.CoinID,
		Exchange:         r.Exchange,
		Symbol:           r.Symbol,
		Timeframe:        r.Timeframe,
		SignalType:       r.SignalType,
		Strength:         r.Strength.String,
		FundingSentiment: r.FundingSentiment.String,
		VolumeSpike:      r.VolumeSpike,
		PriceDeviation:   r.PriceDeviation,
	}
	if r.QuantScore.Valid {
		record.QuantScore = float64Ptr(r.QuantScore.Float64)
	}
	if r.RSI14.Valid {
		record.RSI14 = float64Ptr(r.RSI14.Float64)
	}
	if r.MACDLine.Valid {
		record.MACDLine = float64Ptr(r.MACDLine.Float64)
	}
	if r.MACDSignal.Valid {
		record.MACDSignal = float64Ptr(r.MACDSignal.Float64)
	}
	if r.MACDHist.Valid {
		record.MACDHist = float64Ptr(r.MACDHist.Float64)
	}
	if r.BBUpper.Valid {
		record.BBUpper = float64Ptr(r.BBUpper.Float64)
	}
	if r.BBLower.Valid {
		record.BBLower = float64Ptr(r.BBLower.Float64)
	}
	if r.BBMid.Valid {
		record.BBMid = float64Ptr(r.BBMid.Float64)
	}
	if r.EMA9.Valid {
		record.EMA9 = float64Ptr(r.EMA9.Float64)
	}
	if r.EMA21.Valid {
		record.EMA21 = float64Ptr(r.EMA21.Float64)
	}
	if r.EMA200.Valid {
		record.EMA200 = float64Ptr(r.EMA200.Float64)
	}
	if r.FundingRate.Valid {
		record.FundingRate = float64Ptr(r.FundingRate.Float64)
	}
	if r.AnomalyScore.Valid {
		record.AnomalyScore = float64Ptr(r.AnomalyScore.Float64)
	}
	if r.PriceAtSignal.Valid {
		record.PriceAtSignal = float64Ptr(r.PriceAtSignal.Float64)
	}
	return record
}

func float64Ptr(v float64) *float64 {
	out := v
	return &out
}

func avgFloat64(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	total := 0.0
	count := 0
	for _, value := range values {
		if math.IsNaN(value) || math.IsInf(value, 0) {
			continue
		}
		total += value
		count++
	}
	if count == 0 {
		return nil
	}
	avg := total / float64(count)
	return &avg
}

func ratioFloat64(numerator, denominator int) *float64 {
	if denominator <= 0 {
		return nil
	}
	ratio := float64(numerator) / float64(denominator)
	return &ratio
}
