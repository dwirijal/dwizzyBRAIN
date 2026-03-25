package api

import (
	"net/http"

	"dwizzyBRAIN/api/download"
	"dwizzyBRAIN/api/handler"
	"dwizzyBRAIN/irag"
)

func NewRouter(defi *handler.DefiHandler, news *handler.NewsHandler, auth *handler.AuthHandler, content *handler.ContentHandler, samehadaku ...*handler.SamehadakuHandler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", serveIndex)
	mux.HandleFunc("GET /v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	})
	mux.HandleFunc("GET /docs", serveDocsHTML)
	mux.HandleFunc("GET /docs/", serveDocsHTML)
	mux.HandleFunc("GET /openapi.json", serveOpenAPIJSON)
	if defi != nil {
		defi.Register(mux)
	}
	if news != nil {
		news.Register(mux)
	}
	if auth != nil {
		auth.Register(mux)
	}
	if content != nil {
		content.Register(mux)
	}
	downloadCfg, _ := irag.ConfigFromEnv()
	downloadHandler := handler.NewDownloadHandler(download.NewService(downloadCfg, nil, nil))
	downloadHandler.Register(mux)
	if len(samehadaku) > 0 && samehadaku[0] != nil {
		samehadaku[0].Register(mux)
	}
	return mux
}
