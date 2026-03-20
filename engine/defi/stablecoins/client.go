package stablecoins

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
		baseURL = "https://stablecoins.llama.fi"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse stablecoin base url: %w", err)
	}
	return &Client{
		baseURL:    parsed,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}, nil
}

func (c *Client) Assets(ctx context.Context) ([]Asset, error) {
	var resp Response
	if err := c.getJSON(ctx, "/stablecoins", "includePrices=true", &resp); err != nil {
		return nil, err
	}
	return resp.PeggedAssets, nil
}

func (c *Client) getJSON(ctx context.Context, endpoint, rawQuery string, dest any) error {
	if c == nil {
		return fmt.Errorf("client is required")
	}
	endpoint = strings.TrimSpace(endpoint)
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + endpoint
	u.RawQuery = strings.TrimSpace(rawQuery)
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
