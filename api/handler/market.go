package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	authapi "dwizzyBRAIN/api/auth"
	marketapi "dwizzyBRAIN/api/market"
	"dwizzyBRAIN/api/middleware"
)

type marketReader interface {
	List(ctx context.Context, limit, offset int) ([]marketapi.SnapshotSummary, int, error)
	Detail(ctx context.Context, coinID string) (marketapi.SnapshotDetail, error)
	OHLCV(ctx context.Context, coinID, exchange, timeframe string, limit int) ([]marketapi.OHLCVPoint, error)
	Tickers(ctx context.Context, coinID string) (marketapi.TickerSnapshot, error)
	OrderBook(ctx context.Context, coinID string) (marketapi.OrderBookSnapshot, error)
	Arbitrage(ctx context.Context, coinID string, limit int) ([]marketapi.ArbitrageSignal, error)
}

type MarketHandler struct {
	reader       marketReader
	premiumGuard *middleware.AuthMiddleware
}

func NewMarketHandler(reader marketReader, guard ...*middleware.AuthMiddleware) *MarketHandler {
	var premiumGuard *middleware.AuthMiddleware
	if len(guard) > 0 {
		premiumGuard = guard[0]
	}
	return &MarketHandler{reader: reader, premiumGuard: premiumGuard}
}

func (h *MarketHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/market", h.list)
	mux.HandleFunc("GET /v1/market/{id}", h.detail)
	mux.HandleFunc("GET /v1/market/{id}/ohlcv", h.ohlcv)
	mux.HandleFunc("GET /v1/market/{id}/tickers", h.tickers)
	mux.HandleFunc("GET /v1/market/{id}/orderbook", h.orderbook)
	arbitrageHandler := http.Handler(http.HandlerFunc(h.arbitrage))
	if h.premiumGuard != nil {
		arbitrageHandler = h.premiumGuard.RequirePlan("premium", arbitrageHandler)
	}
	mux.Handle("GET /v1/market/{id}/arbitrage", arbitrageHandler)
}

func (h *MarketHandler) list(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "market reader is unavailable")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	items, total, err := h.reader.List(r.Context(), limit, offset)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"total":  total,
			"limit":  clampResponseLimit(limit),
			"offset": max(offset, 0),
		},
	})
}

func (h *MarketHandler) detail(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "market reader is unavailable")
		return
	}

	coinID := strings.TrimSpace(r.PathValue("id"))
	if coinID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "coin id is required")
		return
	}

	item, err := h.reader.Detail(r.Context(), coinID)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": item,
	})
}

func parseIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func clampResponseLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

func writeErrorFromErr(w http.ResponseWriter, err error) {
	if err == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "unexpected error")
		return
	}

	if errors.Is(err, authapi.ErrUnauthorized) {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}
	if errors.Is(err, authapi.ErrInvalidState) || errors.Is(err, authapi.ErrInvalidToken) || errors.Is(err, authapi.ErrInvalidNonce) || errors.Is(err, authapi.ErrInvalidWalletAddress) || errors.Is(err, authapi.ErrUnsupportedPurpose) {
		writeError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if errors.Is(err, authapi.ErrTokenExpired) || errors.Is(err, authapi.ErrTokenReuse) || errors.Is(err, authapi.ErrSessionRevoked) || errors.Is(err, authapi.ErrNonceExpired) || errors.Is(err, authapi.ErrNonceConsumed) || errors.Is(err, authapi.ErrInvalidSignature) {
		writeError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var notFoundErr interface{ NotFound() bool }
	if errors.As(err, &notFoundErr) && notFoundErr.NotFound() {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	if strings.Contains(strings.ToLower(err.Error()), "not found") {
		writeError(w, http.StatusNotFound, "not_found", err.Error())
		return
	}

	writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
}
