package defi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientFetchesProtocolsAndChains(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/protocols":
			_, _ = w.Write([]byte(`[{"slug":"aave","name":"Aave","symbol":"AAVE","category":"Lending","description":"lending","logo":"logo","url":"https://aave.com","twitter":"aave","chains":["Ethereum"],"tvl":123.45,"change_1d":1.2,"change_7d":3.4,"audits":"2"}]`))
		case "/chains":
			_, _ = w.Write([]byte(`[{"name":"Ethereum","tokenSymbol":"ETH","tvl":456.78,"change_1d":2.3,"change_7d":4.5}]`))
		case "/protocol/aave":
			_, _ = w.Write([]byte(`{"slug":"aave","name":"Aave","symbol":"AAVE","chains":["Ethereum"],"tvl":[{"date":1700000000,"totalLiquidityUSD":123.45}],"chainTvls":{"Ethereum":{"tvl":[{"date":1700000000,"totalLiquidityUSD":123.45}]}}}`))
		case "/v2/historicalChainTvl/Ethereum":
			_, _ = w.Write([]byte(`[{"date":1700000000,"tvl":456.78}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client, err := NewClient(srv.URL)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	protocols, err := client.Protocols(context.Background())
	if err != nil {
		t.Fatalf("protocols: %v", err)
	}
	if len(protocols) != 1 || protocols[0].Slug != "aave" {
		t.Fatalf("unexpected protocols: %#v", protocols)
	}

	chains, err := client.Chains(context.Background())
	if err != nil {
		t.Fatalf("chains: %v", err)
	}
	if len(chains) != 1 || chains[0].Chain != "Ethereum" {
		t.Fatalf("unexpected chains: %#v", chains)
	}

	detail, err := client.Protocol(context.Background(), "aave")
	if err != nil {
		t.Fatalf("protocol detail: %v", err)
	}
	if detail.Slug != "aave" || len(detail.TVL) != 1 {
		t.Fatalf("unexpected detail: %#v", detail)
	}

	history, err := client.ChainHistory(context.Background(), "Ethereum")
	if err != nil {
		t.Fatalf("chain history: %v", err)
	}
	if len(history) != 1 || history[0].TVL != 456.78 {
		t.Fatalf("unexpected chain history: %#v", history)
	}
}
