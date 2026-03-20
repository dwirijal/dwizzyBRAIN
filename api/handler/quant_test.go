package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	quantapi "dwizzyBRAIN/api/quant"
)

type fakeQuantReader struct{}

func (f *fakeQuantReader) Pattern(ctx context.Context, symbol, timeframe string, limit, minMatches int) (quantapi.PatternResponse, error) {
	return quantapi.PatternResponse{
		LowConfidence: true,
		Query: quantapi.PatternQuery{
			Symbol:    symbol,
			Timeframe: timeframe,
			Time:      time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
		},
		Matches: []quantapi.PatternMatch{
			{
				Time:            time.Date(2026, 3, 19, 0, 0, 0, 0, time.UTC),
				Symbol:          symbol,
				Timeframe:       timeframe,
				SimilarityScore: 1.0,
			},
		},
		Outcomes: map[string]quantapi.OutcomeStats{
			"1h": {Count: 0},
		},
	}, nil
}

func (f *fakeQuantReader) SignalLatest(ctx context.Context, symbol, timeframe, exchange string) (quantapi.SignalLatestResponse, error) {
	return quantapi.SignalLatestResponse{
		Query: quantapi.SignalQuery{Symbol: symbol, Timeframe: timeframe, Exchange: exchange},
		Signal: quantapi.SignalRecord{
			ID:          1,
			Symbol:      symbol,
			Timeframe:   timeframe,
			Exchange:    exchange,
			SignalType:  "buy",
			QuantScore:  float64PtrForTest(72.5),
			VolumeSpike: true,
		},
	}, nil
}

func (f *fakeQuantReader) SignalHistory(ctx context.Context, symbol, timeframe, exchange string, limit int) (quantapi.SignalHistoryResponse, error) {
	return quantapi.SignalHistoryResponse{
		Query: quantapi.SignalQuery{Symbol: symbol, Timeframe: timeframe, Exchange: exchange},
		Limit: limit,
		Items: []quantapi.SignalRecord{
			{ID: 1, Symbol: symbol, Timeframe: timeframe, Exchange: exchange, SignalType: "buy"},
		},
	}, nil
}

func (f *fakeQuantReader) SignalSummary(ctx context.Context, symbol, timeframe, exchange string, limit int) (quantapi.SignalSummaryResponse, error) {
	return quantapi.SignalSummaryResponse{
		Query:            quantapi.SignalQuery{Symbol: symbol, Timeframe: timeframe, Exchange: exchange},
		Limit:            limit,
		Count:            1,
		AvgQuantScore:    float64PtrForTest(72.5),
		SignalTypeCounts: map[string]int{"buy": 1},
		Latest: &quantapi.SignalRecord{
			ID:         1,
			Symbol:     symbol,
			Timeframe:  timeframe,
			Exchange:   exchange,
			SignalType: "buy",
		},
	}, nil
}

func float64PtrForTest(v float64) *float64 {
	return &v
}

func TestQuantPatternRoute(t *testing.T) {
	mux := http.NewServeMux()
	NewQuantHandler(&fakeQuantReader{}).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/quant/pattern?symbol=BTC/USDT&timeframe=1m&limit=5&min_matches=1", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected json content type, got %q", ct)
	}
	if want := `"low_confidence":true`; !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("expected body to contain %q, got %s", want, rec.Body.String())
	}
	if want := `"symbol":"BTC/USDT"`; !strings.Contains(rec.Body.String(), want) {
		t.Fatalf("expected body to contain %q, got %s", want, rec.Body.String())
	}
}

func TestQuantSignalRoutes(t *testing.T) {
	mux := http.NewServeMux()
	NewQuantHandler(&fakeQuantReader{}).Register(mux)

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "latest", path: "/v1/quant/signals/latest?symbol=BTC/USDT&timeframe=1m&exchange=binance", want: `"signal_type":"buy"`},
		{name: "history", path: "/v1/quant/signals?symbol=BTC/USDT&timeframe=1m&exchange=binance&limit=5", want: `"items"`},
		{name: "summary", path: "/v1/quant/signals/summary?symbol=BTC/USDT&timeframe=1m&exchange=binance&limit=5", want: `"avg_quant_score"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d body=%s", rec.Code, rec.Body.String())
			}
			if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
				t.Fatalf("expected json content type, got %q", ct)
			}
			if !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("expected body to contain %q, got %s", tc.want, rec.Body.String())
			}
		})
	}
}
