package irag

import "net/http"

func NewRouter(service *Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeEnvelopeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"service": "irag",
		})
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeEnvelopeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"service": "irag",
		})
	})
	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeEnvelopeJSON(w, http.StatusOK, map[string]any{
			"ok":      true,
			"service": "irag",
		})
	})
	mux.HandleFunc("GET /v1/providers", func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeEnvelopeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"error": map[string]any{"message": "irag service unavailable"},
			})
			return
		}
		writeEnvelopeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": service.ProviderSnapshot(),
		})
	})
	mux.HandleFunc("GET /v1/providers/{id}", func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeEnvelopeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"error": map[string]any{"message": "irag service unavailable"},
			})
			return
		}
		id := r.PathValue("id")
		if id == "" {
			writeEnvelopeJSON(w, http.StatusBadRequest, map[string]any{
				"error": map[string]any{"message": "provider id is required"},
			})
			return
		}
		item, ok := service.ProviderDetail(id)
		if !ok {
			writeEnvelopeJSON(w, http.StatusNotFound, map[string]any{
				"error": map[string]any{"message": "provider not found"},
			})
			return
		}
		writeEnvelopeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"data": item,
		})
	})
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("# irag metrics placeholder\n"))
	})
	if service != nil {
		mux.Handle("/v1/", service)
	}
	return mux
}
