package defi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

func NewClient(baseURL string) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.llama.fi"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse defillama base url: %w", err)
	}
	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *Client) Protocols(ctx context.Context) ([]ProtocolListItem, error) {
	var items []ProtocolListItem
	if err := c.getJSON(ctx, "/protocols", &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) Chains(ctx context.Context) ([]ChainListItem, error) {
	var items []ChainListItem
	if err := c.getJSON(ctx, "/chains", &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) Protocol(ctx context.Context, slug string) (ProtocolDetail, error) {
	var item ProtocolDetail
	if err := c.getJSON(ctx, "/protocol/"+url.PathEscape(strings.TrimSpace(slug)), &item); err != nil {
		return ProtocolDetail{}, err
	}
	return item, nil
}

func (c *Client) ChainHistory(ctx context.Context, chain string) ([]ChainTVLPoint, error) {
	var items []ChainTVLPoint
	if err := c.getJSON(ctx, "/v2/historicalChainTvl/"+url.PathEscape(strings.TrimSpace(chain)), &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint string, dest any) error {
	if c == nil {
		return fmt.Errorf("client is required")
	}
	endpoint = strings.TrimSpace(endpoint)
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + endpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fmt.Errorf("create request %s: %w", u.String(), err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", u.String(), err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetch %s: unexpected status %s", u.String(), resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		return fmt.Errorf("decode %s: %w", u.String(), err)
	}
	return nil
}
