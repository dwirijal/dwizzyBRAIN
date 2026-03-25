package handler

import (
	"net/http"
)

type downloadReader interface {
	Enabled() bool
	ServeHTTP(http.ResponseWriter, *http.Request)
}

type DownloadHandler struct {
	service downloadReader
}

func NewDownloadHandler(service downloadReader) *DownloadHandler {
	return &DownloadHandler{service: service}
}

func (h *DownloadHandler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/download/{path...}", h.proxy)
}

func (h *DownloadHandler) proxy(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.service == nil || !h.service.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "service_unavailable", "download service unavailable")
		return
	}

	h.service.ServeHTTP(w, r)
}
