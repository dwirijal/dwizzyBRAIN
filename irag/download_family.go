package irag

import (
	"net/http"
	"strings"
)

type DownloadFamily struct {
	service *Service
}

func NewDownloadFamily(service *Service) *DownloadFamily {
	return &DownloadFamily{service: service}
}

func (f *DownloadFamily) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	lower := strings.ToLower(strings.TrimSpace(r.URL.Path))
	if !strings.HasPrefix(lower, "/v1/download/") {
		writeEnvelopeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]any{"message": "route not found"},
		})
		return
	}

	if f == nil || f.service == nil {
		writeEnvelopeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]any{"message": "service unavailable"},
		})
		return
	}

	if r.Method == http.MethodOptions {
		f.service.writeCORS(w, r)
		w.WriteHeader(http.StatusNoContent)
		return
	}

	spec := f.service.routeSpecForPath(r.URL.Path)
	if len(spec.Providers) == 0 {
		writeEnvelopeJSON(w, http.StatusNotFound, map[string]any{
			"error": map[string]any{"message": "route not found"},
		})
		return
	}

	f.service.serveRoute(w, r, spec)
}
