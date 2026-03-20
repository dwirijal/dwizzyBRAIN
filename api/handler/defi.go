package handler

import (
	"context"
	"net/http"
	"strings"

	defiapi "dwizzyBRAIN/api/defi"
)

type defiReader interface {
	Overview(ctx context.Context, limit int) (defiapi.Overview, error)
	ListProtocols(ctx context.Context, limit, offset int, category string) (defiapi.ProtocolList, error)
	Protocol(ctx context.Context, slug string) (defiapi.ProtocolDetail, error)
	ListChains(ctx context.Context, limit int) ([]defiapi.ChainSummary, error)
	ListDexes(ctx context.Context, limit int) ([]defiapi.DexSummary, error)
}

type DefiHandler struct {
	reader defiReader
}

func NewDefiHandler(reader defiReader) *DefiHandler {
	return &DefiHandler{reader: reader}
}

func (h *DefiHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/defi", h.overview)
	mux.HandleFunc("GET /v1/defi/protocols", h.protocols)
	mux.HandleFunc("GET /v1/defi/protocols/{slug}", h.protocol)
	mux.HandleFunc("GET /v1/defi/chains", h.chains)
	mux.HandleFunc("GET /v1/defi/dexes", h.dexes)
}

func (h *DefiHandler) overview(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "defi reader is unavailable")
		return
	}

	limit := parseIntQuery(r, "limit", 5)
	item, err := h.reader.Overview(r.Context(), limit)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": item,
		"meta": map[string]any{
			"limit": limit,
		},
	})
}

func (h *DefiHandler) protocols(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "defi reader is unavailable")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	offset := parseIntQuery(r, "offset", 0)
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	items, err := h.reader.ListProtocols(r.Context(), limit, offset, category)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items.Items,
		"meta": map[string]any{
			"total":    items.Total,
			"limit":    clampResponseLimit(limit),
			"offset":   max(offset, 0),
			"category": category,
		},
	})
}

func (h *DefiHandler) protocol(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "defi reader is unavailable")
		return
	}

	slug := strings.TrimSpace(r.PathValue("slug"))
	if slug == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "slug is required")
		return
	}

	item, err := h.reader.Protocol(r.Context(), slug)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": item,
	})
}

func (h *DefiHandler) chains(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "defi reader is unavailable")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	items, err := h.reader.ListChains(r.Context(), limit)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"limit": clampResponseLimit(limit),
		},
	})
}

func (h *DefiHandler) dexes(w http.ResponseWriter, r *http.Request) {
	if h.reader == nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "defi reader is unavailable")
		return
	}

	limit := parseIntQuery(r, "limit", 20)
	items, err := h.reader.ListDexes(r.Context(), limit)
	if err != nil {
		writeErrorFromErr(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": items,
		"meta": map[string]any{
			"limit": clampResponseLimit(limit),
		},
	})
}
