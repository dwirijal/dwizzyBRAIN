package handler

import (
	"net/http"
	"strings"
)

func (h *MarketHandler) ohlcv(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "market reader is unavailable")
		return
	}

	coinID := strings.TrimSpace(r.PathValue("id"))
	if coinID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "coin id is required")
		return
	}

	items, err := h.reader.OHLCV(
		r.Context(),
		coinID,
		strings.TrimSpace(r.URL.Query().Get("exchange")),
		strings.TrimSpace(r.URL.Query().Get("timeframe")),
		parseIntQuery(r, "limit", 200),
	)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"coin_id":   coinID,
			"exchange":  strings.TrimSpace(r.URL.Query().Get("exchange")),
			"timeframe": strings.TrimSpace(r.URL.Query().Get("timeframe")),
			"limit":     clampResponseLimit(parseIntQuery(r, "limit", 200)),
		},
	})
}

func (h *MarketHandler) tickers(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "market reader is unavailable")
		return
	}

	coinID := strings.TrimSpace(r.PathValue("id"))
	if coinID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "coin id is required")
		return
	}

	item, err := h.reader.Tickers(r.Context(), coinID)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": item,
	})
}

func (h *MarketHandler) orderbook(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "market reader is unavailable")
		return
	}

	coinID := strings.TrimSpace(r.PathValue("id"))
	if coinID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "coin id is required")
		return
	}

	item, err := h.reader.OrderBook(r.Context(), coinID)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": item,
	})
}

func (h *MarketHandler) arbitrage(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "market reader is unavailable")
		return
	}

	coinID := strings.TrimSpace(r.PathValue("id"))
	if coinID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "coin id is required")
		return
	}

	items, err := h.reader.Arbitrage(r.Context(), coinID, parseIntQuery(r, "limit", 20))
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
	})
}
