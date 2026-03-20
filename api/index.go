package api

import (
	"net/http"
)

func serveIndex(w http.ResponseWriter, r *http.Request) {
	setOpenAPIContractHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{
  "data": {
    "name": "dwizzyBRAIN API",
    "docs": "/docs",
    "openapi": "/openapi.json",
    "health": "/v1/health",
    "auth": "/v1/auth/discord/start"
  }
}`))
}
