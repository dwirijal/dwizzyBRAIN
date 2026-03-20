package stablecoins

import "time"

type Response struct {
	PeggedAssets []Asset `json:"peggedAssets"`
	Chains       []Chain `json:"chains"`
}

type Asset struct {
	ID                   string                  `json:"id"`
	Name                 string                  `json:"name"`
	Symbol               string                  `json:"symbol"`
	GeckoID              string                  `json:"gecko_id"`
	PegType              string                  `json:"pegType"`
	PriceSource          string                  `json:"priceSource"`
	PegMechanism         string                  `json:"pegMechanism"`
	Circulating          PeggedAmount            `json:"circulating"`
	CirculatingPrevDay   PeggedAmount            `json:"circulatingPrevDay"`
	CirculatingPrevWeek  PeggedAmount            `json:"circulatingPrevWeek"`
	CirculatingPrevMonth PeggedAmount            `json:"circulatingPrevMonth"`
	ChainCirculating     map[string]ChainBalance `json:"chainCirculating"`
	Chains               []string                `json:"chains"`
	Price                *float64                `json:"price"`
}

type PeggedAmount struct {
	PeggedUSD float64 `json:"peggedUSD"`
}

type ChainBalance struct {
	Current              PeggedAmount `json:"current"`
	CirculatingPrevDay   PeggedAmount `json:"circulatingPrevDay"`
	CirculatingPrevWeek  PeggedAmount `json:"circulatingPrevWeek"`
	CirculatingPrevMonth PeggedAmount `json:"circulatingPrevMonth"`
}

type Chain struct {
	GeckoID             string       `json:"gecko_id"`
	TotalCirculatingUSD PeggedAmount `json:"totalCirculatingUSD"`
	TokenSymbol         string       `json:"tokenSymbol"`
	Name                string       `json:"name"`
}

type LatestRecord struct {
	CoinID             string
	SnapshotDate       time.Time
	PegType            string
	PegMechanism       string
	PriceUSD           *float64
	MCAPUSD            float64
	Circulating        float64
	BackingComposition map[string]float64
	AttestationURL     *string
	AttestedAt         *time.Time
	SyncedAt           time.Time
}

type HistoryRecord struct {
	Time        time.Time
	CoinID      string
	MCAPUSD     float64
	Circulating *float64
	PriceUSD    *float64
}

type Result struct {
	AssetsFetched   int
	AssetsUpserted  int
	HistoryRows     int
	DepegsDetected  int
	SkippedUnmapped int
}
