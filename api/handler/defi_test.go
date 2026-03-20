package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	defiapi "dwizzyBRAIN/api/defi"
)

type fakeDefiReader struct {
	overview  defiapi.Overview
	protocol  defiapi.ProtocolDetail
	protocols defiapi.ProtocolList
	chains    []defiapi.ChainSummary
	dexes     []defiapi.DexSummary
	err       error
}

func (f *fakeDefiReader) Overview(ctx context.Context, limit int) (defiapi.Overview, error) {
	return f.overview, f.err
}

func (f *fakeDefiReader) ListProtocols(ctx context.Context, limit, offset int, category string) (defiapi.ProtocolList, error) {
	return f.protocols, f.err
}

func (f *fakeDefiReader) Protocol(ctx context.Context, slug string) (defiapi.ProtocolDetail, error) {
	return f.protocol, f.err
}

func (f *fakeDefiReader) ListChains(ctx context.Context, limit int) ([]defiapi.ChainSummary, error) {
	return f.chains, f.err
}

func (f *fakeDefiReader) ListDexes(ctx context.Context, limit int) ([]defiapi.DexSummary, error) {
	return f.dexes, f.err
}

func TestDefiRoutes(t *testing.T) {
	now := time.Date(2026, 3, 18, 22, 30, 0, 0, time.UTC)
	reader := &fakeDefiReader{
		overview: defiapi.Overview{
			Protocols: []defiapi.ProtocolSummary{{Slug: "uniswap", Name: "Uniswap", UpdatedAt: now}},
			Chains:    []defiapi.ChainSummary{{Chain: "ethereum", UpdatedAt: now}},
			Dexes:     []defiapi.DexSummary{{Slug: "uniswap-v3", Name: "Uniswap V3", UpdatedAt: now}},
		},
		protocols: defiapi.ProtocolList{
			Total: 1,
			Items: []defiapi.ProtocolSummary{{Slug: "uniswap", Name: "Uniswap", UpdatedAt: now}},
		},
		protocol: defiapi.ProtocolDetail{Slug: "uniswap", Name: "Uniswap", UpdatedAt: now},
		chains:   []defiapi.ChainSummary{{Chain: "ethereum", UpdatedAt: now}},
		dexes:    []defiapi.DexSummary{{Slug: "uniswap-v3", Name: "Uniswap V3", UpdatedAt: now}},
	}

	mux := http.NewServeMux()
	NewDefiHandler(reader).Register(mux)

	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/v1/defi", want: `"uniswap"`},
		{path: "/v1/defi/protocols", want: `"uniswap"`},
		{path: "/v1/defi/protocols/uniswap", want: `"uniswap"`},
		{path: "/v1/defi/chains", want: `"ethereum"`},
		{path: "/v1/defi/dexes", want: `"uniswap-v3"`},
	} {
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200 for %s, got %d", tc.path, rec.Code)
		}
		if !strings.Contains(rec.Body.String(), tc.want) {
			t.Fatalf("expected %s in response for %s, got %s", tc.want, tc.path, rec.Body.String())
		}
	}
}
