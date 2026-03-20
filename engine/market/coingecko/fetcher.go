package coingecko

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	sharedconfig "dwizzyBRAIN/shared/config"
)

const (
	defaultBaseURL    = "https://api.coingecko.com/api/v3"
	defaultPageSize   = 250
	defaultPageCount  = 4
	defaultPageDelay  = 2 * time.Second
	defaultRetryLimit = 4
)

type MarketCoin struct {
	ID                              string     `json:"id"`
	Symbol                          string     `json:"symbol"`
	Name                            string     `json:"name"`
	Image                           string     `json:"image"`
	CurrentPrice                    *float64   `json:"current_price"`
	MarketCap                       *float64   `json:"market_cap"`
	MarketCapRank                   *int       `json:"market_cap_rank"`
	MarketCapRankWithRehypothecated *int       `json:"market_cap_rank_with_rehypothecated"`
	FullyDilutedValuation           *float64   `json:"fully_diluted_valuation"`
	TotalVolume                     *float64   `json:"total_volume"`
	High24h                         *float64   `json:"high_24h"`
	Low24h                          *float64   `json:"low_24h"`
	PriceChange24h                  *float64   `json:"price_change_24h"`
	PriceChangePercentage24h        *float64   `json:"price_change_percentage_24h"`
	MarketCapChange24h              *float64   `json:"market_cap_change_24h"`
	MarketCapChangePercentage24h    *float64   `json:"market_cap_change_percentage_24h"`
	CirculatingSupply               *float64   `json:"circulating_supply"`
	TotalSupply                     *float64   `json:"total_supply"`
	MaxSupply                       *float64   `json:"max_supply"`
	ATH                             *float64   `json:"ath"`
	ATHChangePercentage             *float64   `json:"ath_change_percentage"`
	ATHDate                         *time.Time `json:"ath_date"`
	ATL                             *float64   `json:"atl"`
	ATLChangePercentage             *float64   `json:"atl_change_percentage"`
	ATLDate                         *time.Time `json:"atl_date"`
	LastUpdated                     time.Time  `json:"last_updated"`
}

type Fetcher struct {
	baseURL        string
	apiKey         string
	client         *http.Client
	pageDelay      time.Duration
	retryLimit     int
	defaultPerPage int
	defaultPages   int
}

func NewFetcherFromEnv() (*Fetcher, error) {
	baseURL := strings.TrimSpace(os.Getenv("COINGECKO_BASE_URL"))
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	apiKey, err := sharedconfig.ReadOptional("COINGECKO_API_KEY")
	if err != nil {
		return nil, err
	}

	return NewFetcher(baseURL, apiKey, nil), nil
}

func NewFetcher(baseURL, apiKey string, client *http.Client) *Fetcher {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	return &Fetcher{
		baseURL:        strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		apiKey:         strings.TrimSpace(apiKey),
		client:         client,
		pageDelay:      defaultPageDelay,
		retryLimit:     defaultRetryLimit,
		defaultPerPage: defaultPageSize,
		defaultPages:   defaultPageCount,
	}
}

func (f *Fetcher) LoadTopMarkets(ctx context.Context, pages, perPage int) ([]MarketCoin, error) {
	if f.client == nil {
		return nil, fmt.Errorf("http client is required")
	}
	if pages <= 0 {
		pages = f.defaultPages
	}
	if perPage <= 0 || perPage > 250 {
		perPage = f.defaultPerPage
	}

	coins := make([]MarketCoin, 0, pages*perPage)
	for page := 1; page <= pages; page++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		batch, err := f.loadPage(ctx, page, perPage)
		if err != nil {
			return nil, fmt.Errorf("load coingecko page %d: %w", page, err)
		}
		coins = append(coins, batch...)

		if page < pages && f.pageDelay > 0 {
			timer := time.NewTimer(f.pageDelay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return nil, ctx.Err()
			case <-timer.C:
			}
		}
	}

	return coins, nil
}

func (f *Fetcher) loadPage(ctx context.Context, page, perPage int) ([]MarketCoin, error) {
	endpoint, err := url.Parse(f.baseURL + "/coins/markets")
	if err != nil {
		return nil, fmt.Errorf("parse coingecko base url: %w", err)
	}

	q := endpoint.Query()
	q.Set("vs_currency", "usd")
	q.Set("order", "market_cap_desc")
	q.Set("per_page", strconv.Itoa(perPage))
	q.Set("page", strconv.Itoa(page))
	q.Set("sparkline", "false")
	q.Set("price_change_percentage", "24h")
	q.Set("include_rehypothecated", "true")
	endpoint.RawQuery = q.Encode()

	var lastErr error
	var payload []MarketCoin
	for attempt := 0; attempt <= f.retryLimit; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		if f.apiKey != "" {
			req.Header.Set("x-cg-pro-api-key", f.apiKey)
		}

		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			func() {
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
					lastErr = f.retryableStatusError(resp)
					return
				}
				if resp.StatusCode != http.StatusOK {
					lastErr = fmt.Errorf("coingecko returned %s", resp.Status)
					return
				}

				dec := json.NewDecoder(resp.Body)
				dec.UseNumber()
				if err := dec.Decode(&payload); err != nil {
					lastErr = fmt.Errorf("decode markets payload: %w", err)
					return
				}
				if payload == nil {
					payload = []MarketCoin{}
				}
				lastErr = nil
			}()

			if lastErr == nil {
				return payload, nil
			}
		}

		if attempt < f.retryLimit {
			if err := waitForRetry(ctx, f.retryAfter(resp, attempt)); err != nil {
				return nil, err
			}
		}
	}

	return nil, lastErr
}

func (f *Fetcher) retryableStatusError(resp *http.Response) error {
	return fmt.Errorf("retryable coingecko status %s", resp.Status)
}

func (f *Fetcher) retryAfter(resp *http.Response, attempt int) time.Duration {
	if resp != nil {
		if raw := strings.TrimSpace(resp.Header.Get("Retry-After")); raw != "" {
			if secs, err := strconv.Atoi(raw); err == nil && secs > 0 {
				return time.Duration(secs) * time.Second
			}
		}
	}

	backoff := time.Duration(attempt+1) * time.Second
	if backoff < time.Second {
		return time.Second
	}
	return backoff
}

func waitForRetry(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
