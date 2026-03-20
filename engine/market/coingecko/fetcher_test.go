package coingecko

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLoadTopMarketsRetries429(t *testing.T) {
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprint(w, `[{"id":"bitcoin","symbol":"btc","name":"Bitcoin","image":"x","market_cap_rank":1,"last_updated":"2026-03-18T00:00:00Z"}]`)
	}))
	defer server.Close()

	fetcher := NewFetcher(server.URL, "", server.Client())
	fetcher.pageDelay = 0
	fetcher.retryLimit = 1

	coins, err := fetcher.LoadTopMarkets(context.Background(), 1, 250)
	if err != nil {
		t.Fatalf("LoadTopMarkets() returned error: %v", err)
	}
	if len(coins) != 1 || coins[0].ID != "bitcoin" {
		t.Fatalf("unexpected coins: %+v", coins)
	}
}
