package api

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"net/http"
)

//go:embed openapi.json
var openAPISpec []byte

const openAPIContractVersion = "1.8.1"

var openAPISpecSHA256 = func() string {
	sum := sha256.Sum256(openAPISpec)
	return hex.EncodeToString(sum[:])
}()

func setOpenAPIContractHeaders(w http.ResponseWriter) {
	w.Header().Set("X-OpenAPI-Version", openAPIContractVersion)
	w.Header().Set("X-OpenAPI-SHA256", openAPISpecSHA256)
}

func serveOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	setOpenAPIContractHeaders(w)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(openAPISpec)
}
