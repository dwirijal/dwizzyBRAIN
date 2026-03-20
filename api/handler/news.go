package handler

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	newsapi "dwizzyBRAIN/api/news"
)

type newsReader interface {
	IsCategory(value string) bool
	List(ctx context.Context, limit, offset int, category string) (newsapi.ArticlePage, error)
	Detail(ctx context.Context, id int64) (newsapi.ArticleDetail, error)
	ByCoin(ctx context.Context, coinID string, limit, offset int) (newsapi.ArticlePage, error)
	Trending(ctx context.Context, window time.Duration, limit int) (newsapi.TrendingResponse, error)
}

type NewsHandler struct {
	reader newsReader
}

func NewNewsHandler(reader newsReader) *NewsHandler {
	return &NewsHandler{reader: reader}
}

func (h *NewsHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/news", h.list)
	mux.HandleFunc("GET /v1/news/trending", h.trending)
	mux.HandleFunc("GET /v1/news/coin/{coin_id}", h.coin)
	mux.HandleFunc("GET /v1/news/{value}", h.value)
}

func (h *NewsHandler) list(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "news reader is unavailable")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	category := strings.TrimSpace(r.URL.Query().Get("category"))

	items, err := h.reader.List(r.Context(), limit, offset, category)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items.Items,
		"meta": map[string]any{
			"total":  items.Total,
			"limit":  clampNewsResponseLimit(limit),
			"offset": max(offset, 0),
		},
	})
}

func (h *NewsHandler) value(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "news reader is unavailable")
		return
	}

	value := strings.TrimSpace(r.PathValue("value"))
	if value == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "value is required")
		return
	}

	if id, err := strconv.ParseInt(value, 10, 64); err == nil {
		item, err := h.reader.Detail(r.Context(), id)
		if err != nil {
			writeErrorFromErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": item})
		return
	}

	if !h.reader.IsCategory(value) {
		writeError(w, http.StatusNotFound, "not_found", "news item not found")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	items, err := h.reader.List(r.Context(), limit, offset, value)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items.Items,
		"meta": map[string]any{
			"total":    items.Total,
			"limit":    clampNewsResponseLimit(limit),
			"offset":   max(offset, 0),
			"category": value,
		},
	})
}

func (h *NewsHandler) coin(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "news reader is unavailable")
		return
	}

	coinID := strings.TrimSpace(r.PathValue("coin_id"))
	if coinID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "coin id is required")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	items, err := h.reader.ByCoin(r.Context(), coinID, limit, offset)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items.Items,
		"meta": map[string]any{
			"total":   items.Total,
			"limit":   clampNewsResponseLimit(limit),
			"offset":  max(offset, 0),
			"coin_id": coinID,
		},
	})
}

func (h *NewsHandler) trending(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "news reader is unavailable")
		return
	}

	window := parseDurationQuery(r, "window", 24*time.Hour)
	limit := parseIntQuery(r, "limit", 10)

	item, err := h.reader.Trending(r.Context(), window, limit)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

func parseDurationQuery(r *http.Request, key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func clampNewsResponseLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 100 {
		return 100
	}
	return limit
}
