package yields

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
		baseURL = "https://yields.llama.fi"
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse yields base url: %w", err)
	}
	return &Client{
		baseURL: parsed,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

func (c *Client) Pools(ctx context.Context) ([]PoolSnapshot, error) {
	var response PoolsResponse
	if err := c.getJSON(ctx, "/pools", &response); err != nil {
		return nil, err
	}
	return response.Data, nil
}

func (c *Client) PoolChart(ctx context.Context, pool string) ([]ChartPoint, error) {
	var response ChartResponse
	if err := c.getJSON(ctx, "/chart/"+url.PathEscape(strings.TrimSpace(pool)), &response); err != nil {
		return nil, err
	}
	return response.Data, nil
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
