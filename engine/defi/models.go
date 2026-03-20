package defi

import "time"

type ProtocolListItem struct {
	Slug        string    `json:"slug"`
	Name        string    `json:"name"`
	Symbol      string    `json:"symbol"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Logo        string    `json:"logo"`
	URL         string    `json:"url"`
	Twitter     string    `json:"twitter"`
	Chains      []string  `json:"chains"`
	TVL         float64   `json:"tvl"`
	Change1D    float64   `json:"change_1d"`
	Change7D    float64   `json:"change_7d"`
	Audits      string    `json:"audits"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ChainListItem struct {
	Chain       string    `json:"name"`
	Name        string    `json:"-"`
	TokenSymbol string    `json:"tokenSymbol"`
	TVL         float64   `json:"tvl"`
	Change1D    float64   `json:"change_1d"`
	Change7D    float64   `json:"change_7d"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProtocolDetail struct {
	Slug             string                          `json:"slug"`
	Name             string                          `json:"name"`
	Symbol           string                          `json:"symbol"`
	Category         string                          `json:"category"`
	Description      string                          `json:"description"`
	Logo             string                          `json:"logo"`
	URL              string                          `json:"url"`
	Twitter          string                          `json:"twitter"`
	Chains           []string                        `json:"chains"`
	TVL              []TVLPoint                      `json:"tvl"`
	ChainTvls        map[string]ProtocolChainHistory `json:"chainTvls"`
	CurrentChainTvls map[string]float64              `json:"currentChainTvls"`
}

type ProtocolChainHistory struct {
	TVL []TVLPoint `json:"tvl"`
}

type TVLPoint struct {
	Date              int64   `json:"date"`
	TotalLiquidityUSD float64 `json:"totalLiquidityUSD"`
}

type ChainTVLPoint struct {
	Date int64   `json:"date"`
	TVL  float64 `json:"tvl"`
}

type ProtocolHistoryRecord struct {
	Time         time.Time
	ProtocolSlug string
	Chain        string
	TVLUSD       float64
	Metadata     map[string]any
}

type ChainHistoryRecord struct {
	Time   time.Time
	Chain  string
	TVLUSD float64
}
