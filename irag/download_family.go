package irag

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"
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

	if !f.service.Enabled() {
		writeEnvelopeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]any{"message": "service unavailable"},
		})
		return
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()

	cacheKey := f.service.cacheKey(r.Method, r.URL, body)
	if spec.CacheTTL > 0 && f.service.cfg.CacheEnabled && f.service.cache != nil && r.Method == http.MethodGet {
		if cached, ok, err := f.service.cache.Get(r.Context(), cacheKey); err == nil && ok {
			f.service.writeCached(w, r, cached, spec.CacheTTL)
			f.service.log(r, spec, cached.Provider, []string{}, "cache_hit_l1", cached.Status, 0, len(cached.Body), cacheKey, spec.CacheTTL, "", "", true)
			return
		}
	}

	start := time.Now()
	resp, attempted, err := f.service.proxyWithFallback(r.Context(), r, spec, body)
	if err != nil {
		status := http.StatusBadGateway
		if err == context.DeadlineExceeded {
			status = http.StatusGatewayTimeout
		}
		f.service.writeFailure(w, status, spec, attempted, err)
		f.service.log(r, spec, attemptedProvider(attempted), attempted, "provider_error", status, time.Since(start), 0, cacheKey, spec.CacheTTL, "provider_error", err.Error(), false)
		return
	}

	if spec.CacheTTL > 0 && f.service.cfg.CacheEnabled && f.service.cache != nil && r.Method == http.MethodGet {
		_ = f.service.cache.Set(r.Context(), cacheKey, resp.cachedResponse(), spec.CacheTTL)
	}

	f.service.writeResponse(w, r, resp)
	f.service.log(r, spec, resp.Provider, attempted, resp.StatusClass(), resp.Status, resp.Latency, len(resp.Body), cacheKey, spec.CacheTTL, "", "", false)
}
