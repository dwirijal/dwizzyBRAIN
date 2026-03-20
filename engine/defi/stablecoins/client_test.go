package stablecoins

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFetchesAssets(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/stablecoins" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"peggedAssets":[{"id":"1","name":"Tether","symbol":"USDT","gecko_id":"tether","pegType":"peggedUSD","pegMechanism":"fiat-backed","circulating":{"peggedUSD":100},"circulatingPrevDay":{"peggedUSD":99},"circulatingPrevWeek":{"peggedUSD":98},"circulatingPrevMonth":{"peggedUSD":97},"chainCirculating":{"Ethereum":{"current":{"peggedUSD":60}}},"price":1.0001}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	assets, err := client.Assets(context.Background())
	if err != nil {
		t.Fatalf("assets: %v", err)
	}
	if len(assets) != 1 || assets[0].Symbol != "USDT" {
		t.Fatalf("unexpected assets: %#v", assets)
	}
}
