package yields

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFetchesPoolsAndChart(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pools":
			_, _ = w.Write([]byte(`{"status":"success","data":[{"chain":"Ethereum","project":"lido","symbol":"STETH","pool":"pool-1","tvlUsd":123.45,"apy":2.3}]}`))
		case "/chart/pool-1":
			_, _ = w.Write([]byte(`{"status":"success","data":[{"timestamp":"2024-01-01T00:00:00.000Z","tvlUsd":123.45,"apy":2.3,"apyBase":2.1,"apyReward":0.2}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	pools, err := client.Pools(context.Background())
	if err != nil {
		t.Fatalf("pools: %v", err)
	}
	if len(pools) != 1 || pools[0].Pool != "pool-1" {
		t.Fatalf("unexpected pools: %#v", pools)
	}

	chart, err := client.PoolChart(context.Background(), "pool-1")
	if err != nil {
		t.Fatalf("chart: %v", err)
	}
	if len(chart) != 1 || chart[0].TVLUSD != 123.45 {
		t.Fatalf("unexpected chart: %#v", chart)
	}
}
