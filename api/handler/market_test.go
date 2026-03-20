package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	authapi "dwizzyBRAIN/api/auth"
	marketapi "dwizzyBRAIN/api/market"
	"dwizzyBRAIN/api/middleware"
)

type fakeMarketReader struct {
	items  []marketapi.SnapshotSummary
	total  int
	detail marketapi.SnapshotDetail
	err    error
}

func (f *fakeMarketReader) List(ctx context.Context, limit, offset int) ([]marketapi.SnapshotSummary, int, error) {
	return f.items, f.total, f.err
}

func (f *fakeMarketReader) Detail(ctx context.Context, coinID string) (marketapi.SnapshotDetail, error) {
	if f.err != nil {
		return marketapi.SnapshotDetail{}, f.err
	}
	return f.detail, nil
}

func (f *fakeMarketReader) OHLCV(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]marketapi.OHLCVPoint, error) {
	return []marketapi.OHLCVPoint{{CoinID: coinID, Exchange: exchange, Timeframe: timeframe}}, nil
}

func (f *fakeMarketReader) Tickers(ctx context.Context, coinID string) (marketapi.TickerSnapshot, error) {
	return marketapi.TickerSnapshot{CoinID: coinID}, nil
}

func (f *fakeMarketReader) OrderBook(ctx context.Context, coinID string) (marketapi.OrderBookSnapshot, error) {
	return marketapi.OrderBookSnapshot{CoinID: coinID}, nil
}

func (f *fakeMarketReader) Arbitrage(ctx context.Context, coinID string, limit int) ([]marketapi.ArbitrageSignal, error) {
	return []marketapi.ArbitrageSignal{{CoinID: coinID}}, nil
}

func TestMarketRoutes(t *testing.T) {
	now := time.Date(2026, 3, 18, 22, 0, 0, 0, time.UTC)
	reader := &fakeMarketReader{
		items: []marketapi.SnapshotSummary{
			{
				CoinID:    "bitcoin",
				Symbol:    "btc",
				Name:      "Bitcoin",
				Rank:      1,
				UpdatedAt: now,
				Availability: marketapi.Availability{
					Tier: "A",
				},
				ExchangeCount: 3,
			},
		},
		total: 1,
		detail: marketapi.SnapshotDetail{
			SnapshotSummary: marketapi.SnapshotSummary{
				CoinID:    "bitcoin",
				Symbol:    "btc",
				Name:      "Bitcoin",
				Rank:      1,
				UpdatedAt: now,
				Availability: marketapi.Availability{
					Tier: "A",
				},
				ExchangeCount: 3,
			},
		},
	}

	mux := http.NewServeMux()
	NewMarketHandler(reader).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/market?limit=1&offset=0", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"bitcoin"`) {
		t.Fatalf("expected bitcoin in list response, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"coin_id":"bitcoin"`) {
		t.Fatalf("expected bitcoin in detail response, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin/ohlcv?exchange=binance&timeframe=1m", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for ohlcv, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"timeframe":"1m"`) {
		t.Fatalf("expected ohlcv response, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin/tickers", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for tickers, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"coin_id":"bitcoin"`) {
		t.Fatalf("expected tickers response, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin/orderbook", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for orderbook, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"coin_id":"bitcoin"`) {
		t.Fatalf("expected orderbook response, got %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin/arbitrage", nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for arbitrage, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"coin_id":"bitcoin"`) {
		t.Fatalf("expected arbitrage response, got %s", rec.Body.String())
	}
}

func TestMarketDetailNotFound(t *testing.T) {
	reader := &fakeMarketReader{
		err: errors.New("coin \"bitcoin\" not found"),
	}
	mux := http.NewServeMux()
	NewMarketHandler(reader).Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestMarketArbitragePremiumGate(t *testing.T) {
	reader := &fakeMarketReader{
		items:  []marketapi.SnapshotSummary{},
		total:  0,
		detail: marketapi.SnapshotDetail{},
	}
	guard := middleware.NewAuthMiddleware(fakeAuthService{
		responses: map[string]authapi.MeResponse{
			"free-token": {
				User: authapi.UserProfile{Plan: "free"},
			},
			"premium-token": {
				User: authapi.UserProfile{Plan: "premium"},
			},
		},
	})

	mux := http.NewServeMux()
	NewMarketHandler(reader, guard).Register(mux)

	t.Run("unauthorized", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin/arbitrage", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("free forbidden", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin/arbitrage", nil)
		req.Header.Set("Authorization", "Bearer free-token")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("premium allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/market/bitcoin/arbitrage", nil)
		req.Header.Set("Authorization", "Bearer premium-token")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
	})
}

type fakeAuthService struct {
	responses    map[string]authapi.MeResponse
	entitlements map[string]authapi.EntitlementResponse
	err          error
}

func (f fakeAuthService) Me(ctx context.Context, token string) (authapi.MeResponse, error) {
	if f.err != nil {
		return authapi.MeResponse{}, f.err
	}
	if resp, ok := f.responses[token]; ok {
		return resp, nil
	}
	return authapi.MeResponse{}, authapi.ErrUnauthorized
}

func (f fakeAuthService) Entitlement(ctx context.Context, token string) (authapi.EntitlementResponse, error) {
	if f.err != nil {
		return authapi.EntitlementResponse{}, f.err
	}
	if resp, ok := f.entitlements[token]; ok {
		return resp, nil
	}
	if resp, ok := f.responses[token]; ok {
		return authapi.EntitlementResponse{Plan: resp.User.Plan}, nil
	}
	return authapi.EntitlementResponse{}, authapi.ErrUnauthorized
}
