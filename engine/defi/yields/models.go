package yields

import "time"

type PoolsResponse struct {
	Status string         `json:"status"`
	Data   []PoolSnapshot `json:"data"`
}

type PoolSnapshot struct {
	Chain            string         `json:"chain"`
	Project          string         `json:"project"`
	Symbol           string         `json:"symbol"`
	Pool             string         `json:"pool"`
	TVLUSD           float64        `json:"tvlUsd"`
	APY              *float64       `json:"apy"`
	APYBase          *float64       `json:"apyBase"`
	APYReward        *float64       `json:"apyReward"`
	APYPct1D         *float64       `json:"apyPct1D"`
	APYPct7D         *float64       `json:"apyPct7D"`
	APYPct30D        *float64       `json:"apyPct30D"`
	Stablecoin       bool           `json:"stablecoin"`
	ILRisk           string         `json:"ilRisk"`
	Exposure         string         `json:"exposure"`
	RewardTokens     []string       `json:"rewardTokens"`
	UnderlyingTokens []string       `json:"underlyingTokens"`
	Predictions      map[string]any `json:"predictions"`
	PoolMeta         any            `json:"poolMeta"`
	APYMean30D       *float64       `json:"apyMean30d"`
	VolumeUsd1D      *float64       `json:"volumeUsd1d"`
	VolumeUsd7D      *float64       `json:"volumeUsd7d"`
	APYBaseInception *float64       `json:"apyBaseInception"`
	Mu               *float64       `json:"mu"`
	Sigma            *float64       `json:"sigma"`
	Count            *int           `json:"count"`
	Outlier          bool           `json:"outlier"`
}

type ChartResponse struct {
	Status string       `json:"status"`
	Data   []ChartPoint `json:"data"`
}

type ChartPoint struct {
	Timestamp time.Time `json:"timestamp"`
	TVLUSD    float64   `json:"tvlUsd"`
	APY       *float64  `json:"apy"`
	APYBase   *float64  `json:"apyBase"`
	APYReward *float64  `json:"apyReward"`
	IL7D      *float64  `json:"il7d"`
	APYBase7D *float64  `json:"apyBase7d"`
}

type LatestRecord struct {
	Pool             string
	Chain            string
	Project          string
	Symbol           string
	ProtocolSlug     string
	TVLUSD           float64
	APY              *float64
	APYBase          *float64
	APYReward        *float64
	APYPct1D         *float64
	APYPct7D         *float64
	APYPct30D        *float64
	APYMean30D       *float64
	VolumeUsd1D      *float64
	VolumeUsd7D      *float64
	Stablecoin       bool
	ILRisk           string
	Exposure         string
	RewardTokens     []string
	UnderlyingTokens []string
	Predictions      map[string]any
	PoolMeta         map[string]any
	Outlier          bool
	Count            *int
	UpdatedAt        time.Time
	SyncedAt         time.Time
}

type HistoryRecord struct {
	Time      time.Time
	Pool      string
	Chain     string
	Project   string
	Symbol    string
	TVLUSD    float64
	APY       *float64
	APYBase   *float64
	APYReward *float64
	Metadata  map[string]any
}

type Result struct {
	PoolsFetched    int
	PoolsUpserted   int
	PoolsBackfilled int
}
