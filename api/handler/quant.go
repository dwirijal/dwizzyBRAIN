package handler

import (
	"context"
	"net/http"
	"strings"

	quantapi "dwizzyBRAIN/api/quant"
)

type quantReader interface {
	Pattern(ctx context.Context, symbol, timeframe string, limit, minMatches int) (quantapi.PatternResponse, error)
	SignalLatest(ctx context.Context, symbol, timeframe, exchange string) (quantapi.SignalLatestResponse, error)
	SignalHistory(ctx context.Context, symbol, timeframe, exchange string, limit int) (quantapi.SignalHistoryResponse, error)
	SignalSummary(ctx context.Context, symbol, timeframe, exchange string, limit int) (quantapi.SignalSummaryResponse, error)
}

type QuantHandler struct {
	reader quantReader
}

func NewQuantHandler(reader quantReader) *QuantHandler {
	return &QuantHandler{reader: reader}
}

func (h *QuantHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/quant/pattern", h.pattern)
	mux.HandleFunc("GET /v1/quant/signals/latest", h.latestSignal)
	mux.HandleFunc("GET /v1/quant/signals", h.signalHistory)
	mux.HandleFunc("GET /v1/quant/signals/summary", h.signalSummary)
}

func (h *QuantHandler) pattern(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "quant reader is unavailable")
		return
	}

	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	timeframe := strings.TrimSpace(r.URL.Query().Get("timeframe"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "symbol is required")
		return
	}
	if timeframe == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "timeframe is required")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	minMatches := parseIntQuery(r, "min_matches", 30)

	result, err := h.reader.Pattern(r.Context(), symbol, timeframe, limit, minMatches)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *QuantHandler) latestSignal(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "quant reader is unavailable")
		return
	}

	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	timeframe := strings.TrimSpace(r.URL.Query().Get("timeframe"))
	exchange := strings.TrimSpace(r.URL.Query().Get("exchange"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "symbol is required")
		return
	}
	if timeframe == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "timeframe is required")
		return
	}

	result, err := h.reader.SignalLatest(r.Context(), symbol, timeframe, exchange)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *QuantHandler) signalHistory(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "quant reader is unavailable")
		return
	}

	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	timeframe := strings.TrimSpace(r.URL.Query().Get("timeframe"))
	exchange := strings.TrimSpace(r.URL.Query().Get("exchange"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "symbol is required")
		return
	}
	if timeframe == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "timeframe is required")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	result, err := h.reader.SignalHistory(r.Context(), symbol, timeframe, exchange, limit)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}

func (h *QuantHandler) signalSummary(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "quant reader is unavailable")
		return
	}

	symbol := strings.TrimSpace(r.URL.Query().Get("symbol"))
	timeframe := strings.TrimSpace(r.URL.Query().Get("timeframe"))
	exchange := strings.TrimSpace(r.URL.Query().Get("exchange"))
	if symbol == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "symbol is required")
		return
	}
	if timeframe == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "timeframe is required")
		return
	}

	limit := parseIntQuery(r, "limit", 50)
	result, err := h.reader.SignalSummary(r.Context(), symbol, timeframe, exchange, limit)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": result})
}
