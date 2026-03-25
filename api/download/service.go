package download

import (
	"encoding/json"
	"net/http"

	"dwizzyBRAIN/irag"
)

type Service struct {
	family http.Handler
}

func NewService(cfg irag.Config, cache irag.Cache, logs *irag.LogStore) *Service {
	backend := irag.NewService(cfg, cache, logs)
	if backend == nil || !backend.Enabled() {
		return &Service{}
	}
	return &Service{family: irag.NewDownloadFamily(backend)}
}

func (s *Service) Enabled() bool {
	return s != nil && s.family != nil
}

func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !s.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error": map[string]any{
				"message": "download service unavailable",
			},
		})
		return
	}
	s.family.ServeHTTP(w, r)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
