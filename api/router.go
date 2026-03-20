package api

import (
	"net/http"

	"dwizzyBRAIN/api/handler"
)

func NewRouter(market *handler.MarketHandler, defi *handler.DefiHandler, news *handler.NewsHandler, auth *handler.AuthHandler, quant *handler.QuantHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", serveIndex)
	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	})
	mux.HandleFunc("GET /docs", serveDocsHTML)
	mux.HandleFunc("GET /docs/", serveDocsHTML)
	mux.HandleFunc("GET /openapi.json", serveOpenAPIJSON)
	if market != nil {
		market.Register(mux)
	}
	if defi != nil {
		defi.Register(mux)
	}
	if news != nil {
		news.Register(mux)
	}
	if auth != nil {
		auth.Register(mux)
	}
	if quant != nil {
		quant.Register(mux)
	}
	return mux
}
