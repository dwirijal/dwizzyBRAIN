# PRD-QUANT.md — dwizzyBRAIN Quant Engine
**Version:** 1.1.0
**Last Updated:** 2026-03-19
**Status:** Core quant complete; optional extensions remain
**Owner:** Rijal (dwizzyBRAIN)

---

## Table of Contents

1. [Overview](#1-overview)
2. [Goals & Non-Goals](#2-goals--non-goals)
3. [Architecture Position](#3-architecture-position)
4. [Data Layer](#4-data-layer)
   - 4.1 [Storage Tiers](#41-storage-tiers)
   - 4.2 [Database Schema](#42-database-schema)
   - 4.3 [Backfill Strategy](#43-backfill-strategy)
5. [Technical Analysis Library](#5-technical-analysis-library)
   - 5.1 [Indicator Catalog](#51-indicator-catalog)
   - 5.2 [Derived Features](#52-derived-features)
   - 5.3 [Candlestick Patterns](#53-candlestick-patterns)
   - 5.4 [SMC Concepts](#54-smc-smart-money-concepts)
6. [Event Labelling System](#6-event-labelling-system)
   - 6.1 [Macro Events](#61-macro-events)
   - 6.2 [Proximity Labels](#62-proximity-labels)
   - 6.3 [Regime Labels](#63-regime-labels)
   - 6.4 [Composite Label](#64-composite-macro-environment-label)
7. [Pattern Matching Engine](#7-pattern-matching-engine)
   - 7.1 [Fingerprint Vector](#71-fingerprint-vector)
   - 7.2 [Similarity Search](#72-similarity-search)
   - 7.3 [Pattern Output](#73-pattern-output)
8. [External Data Sources](#8-external-data-sources)
9. [Knowledge & Documentation Layer](#9-knowledge--documentation-layer)
10. [Implementation Roadmap](#10-implementation-roadmap)
11. [Schema Update Log](#11-schema-update-log)

---

## 1. Overview

The **dwizzyBRAIN Quant Engine** is the intelligence backbone of dwizzyBRAIN — a self-hosted market analysis platform running on a homelab Mini PC. The quant layer is a **dedicated Python service** (`quant/`) that sits between the Go `engine/` market data pipeline and the AI agent layer. It is responsible for:

- Subscribing to raw OHLCV data published by `engine/market/` via Valkey channel `ch:ohlcv:raw:*`
- Computing a comprehensive suite of technical indicators on OHLCV data in real-time
- Computing a composite `quant_score` (0–100) and funding rate sentiment per symbol
- Detecting anomalies: volume spikes, price deviations
- Publishing computed signals to `ch:signal:processed:{symbol}` for consumption by `engine/agent/`
- Running bulk backfill computation on historical candles for pattern library construction
- Labelling every historical candle with macro news/event context
- Building a searchable pattern library via vector embeddings in pgvector
- Identifying historical analogs: *"When did conditions like today occur before, and what happened next?"*

The output of the quant engine feeds `engine/agent/` (AgentRouter → LLM providers), the `api/` service (REST + WebSocket), and the dwizzyBOT (TypeScript Discord bot) for real-time alerts, dashboards, and trading intelligence.

---

## 1.1 Implementation Matrix

| Area | PRD intent | Implementation | Status |
|---|---|---|---|
| Realtime quant loop | Subscribe raw OHLCV, compute signals, publish processed output | `quant/main.py`, indicators/scorer/funding/anomaly, Valkey hot cache | done |
| Indicator catalog | 60+ TA features, derived features, candlestick patterns, SMC | Trend/momentum/volatility/volume/pattern/SMC stack implemented | done |
| Backfill | Historical OHLCV, bulk indicators, cold archive | CCXT + Yahoo Finance backfill, DuckDB/Parquet export, bulk compute | done |
| Macro events | FRED + ForexFactory, candle labels, regime labels | Macro event tables + labeling pipeline live | done |
| Pattern engine | Fingerprints, embeddings, similarity search, confidence gating | pgvector bulk load + API + bot consumer live | done |
| Consumer integration | API/bot access to quant outputs | `/v1/quant/pattern` and `/pattern` live | done |
| External data sources | CCXT, Yahoo Finance, FRED, ForexFactory | Implemented for the quant slice | done |
| ML expansion | Outcome model + local embeddings | Training and local embedding generation live | done |
| Optional provider coverage | Extra integrations like CryptoPanic | Still optional | partial |
| Ops hardening / rollout | Container/runtime wiring and deployment | Implemented in repo; deployment remains operational | partial |

## 2. Goals & Non-Goals

### Goals
- Build a universal TA computation pipeline covering all major asset classes
- Backfill 3–8 years of historical OHLCV data per symbol
- Label every candle with macro event proximity, surprise context, and rate regime
- Enable historical pattern matching with statistically significant sample sizes (30+ matches)
- Keep infrastructure cost at $0/month using homelab-only resources
- Maintain lean storage via TimescaleDB compression + Parquet cold archive

### Non-Goals
- Real-time order execution / trading bot (separate system)
- Storing raw data in Google Sheets, Notion, or Google Drive as primary storage
- Supporting proprietary broker APIs in v1
- Building a fully automated ML strategy (v2+)

---

## 3. Architecture Position

```
┌──────────────────────────────────────────────────────────────────────┐
│                        dwizzyBRAIN Monorepo                          │
│                                                                      │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                      engine/  (Go — internal)                   │ │
│  │                                                                 │ │
│  │  market/         →  PUBLISH ch:ohlcv:raw:{symbol}:{market}:{tf} │ │
│  │  news/           →  PUBLISH ch:news:raw                        │ │
│  │  agent/          ←  SUBSCRIBE ch:signal:processed:{symbol}     │ │
│  │  pipeline/       →  hot/warm/cold tiering                      │ │
│  │  storage_ext/    →  Google Drive, Cloudflare R2, rclone        │ │
│  └──────────────────────────────┬──────────────────────────────────┘ │
│                                 │ Valkey pub/sub                     │
│                    ┌────────────▼────────────┐                       │
│                    │     quant/  (Python)    │                       │
│                    │   ← INTERNAL SERVICE →  │                       │
│                    │                         │                       │
│                    │  main.py      subscribe │                       │
│                    │  indicators.py  pandas-ta│                       │
│                    │  scorer.py    0-100 score│                       │
│                    │  funding.py   futures   │                       │
│                    │  anomaly.py   detection │                       │
│                    │  schema.py    types     │                       │
│                    └────────────┬────────────┘                       │
│                                 │ PUBLISH ch:signal:processed:*      │
│                    ┌────────────▼────────────┐                       │
│                    │   engine/agent/ (Go)    │                       │
│                    │   AgentRouter priority  │                       │
│                    │   irag → Groq → Gemini  │                       │
│                    │   → OpenRouter          │                       │
│                    └────────────┬────────────┘                       │
│                                 │ PUBLISH ch:ai:result:{symbol}      │
│              ┌──────────────────┴──────────────────┐                 │
│              ▼                                      ▼                │
│  ┌───────────────────────┐            ┌─────────────────────────┐    │
│  │     api/  (Go :8080)  │            │   dwizzyBOT  (TS)       │    │
│  │  REST + WebSocket     │            │   Discord bot interface │    │
│  │  api.dwizzy.my.id     │            └─────────────────────────┘    │
│  └───────────────────────┘                                           │
│                                                                      │
│  ┌───────────────────────────────────────────────────────────────┐   │
│  │                       Storage Layer                            │   │
│  │  Hot:    Valkey          — real-time signals < 1h             │   │
│  │  Warm:   TimescaleDB     — 6 months compressed OHLCV+signals  │   │
│  │  Cold:   Parquet/local   — > 6 months, DuckDB queryable       │   │
│  │  Ext:    Google Drive    — pg_dump + Parquet backup (weekly)  │   │
│  │  CDN:    Cloudflare R2   — coin/chain logos (permanent)       │   │
│  │  Files:  Telegram file_id— generated charts (permanent)      │   │
│  └───────────────────────────────────────────────────────────────┘   │
└──────────────────────────────────────────────────────────────────────┘
```

**Key separation of concerns:**
- `engine/market/` = ingest raw OHLCV + tickers, publish to Valkey — never computes indicators
- `quant/` = Python service, subscribes to raw OHLCV, computes all TA + scores, publishes signals
- `engine/agent/` = AgentRouter, consumes quant signals, routes to LLM providers
- `engine/pipeline/` = consumes processed signals, routes to correct storage tier
- `api/` = public interface only, reads from DB/Valkey cache, never computes
- `dwizzyBOT` = Discord consumer of `api/` and `ch:ai:result:*`
- `irag/` = Indonesian REST API Gateway, internal port 8081, proxied through `api/`

**Valkey channels (quant-relevant):**

| Channel | Producer | Consumer |
|---|---|---|
| `ch:ohlcv:raw:{symbol}:{market}:{tf}` | `engine/market/` | `quant/` |
| `ch:news:raw` | `engine/news/` | `quant/` |
| `ch:signal:processed:{symbol}` | `quant/` | `engine/agent/` |
| `ch:ai:result:{symbol}` | `engine/agent/` | `api/ws`, `dwizzyBOT` |

---

## 4. Data Layer

### 4.1 Storage Tiers

| Tier | Technology | TTL / Scope | Use Case | Est. Size |
|---|---|---|---|---|
| **Hot** | Valkey | `signal:{symbol}:{tf}` TTL 1h | Real-time quant signals, live indicators | ~100 MB |
| **Warm** | TimescaleDB (compressed) | Last 6 months | Pattern matching, backtesting, API queries | ~500 MB–1 GB |
| **Cold** | Parquet files (local disk) | > 6 months | Historical pattern library, long-range backtest | ~2–5 GB |
| **Backup DB** | Google Drive | Weekly pg_dump gzip | Disaster recovery for PostgreSQL + TimescaleDB | Mirror |
| **Backup Cold** | Google Drive / Storj | Permanent | CSV/Parquet cold archive | Mirror |
| **Charts** | Telegram `file_id` | Permanent | Generated chart images, `file_id` stored in PostgreSQL | — |
| **CDN** | Cloudflare R2 | Permanent | Coin/chain/protocol logos | — |

**Cold tier migration policy:**
- Candles older than 6 months exported to Parquet via weekly batch job in `quant/`
- Parquet files partitioned by `symbol/year/timeframe`
- DuckDB used to query cold Parquet without loading into memory
- `engine/storage_ext/rclone.go` triggers rclone sync to Google Drive every Sunday at 02:00
- `engine/storage_ext/gdrive.go` handles Google Drive API uploads directly for pg_dump backups

**Compression settings (TimescaleDB):**
```sql
ALTER TABLE candles SET (
  timescaledb.compress,
  timescaledb.compress_segmentby = 'symbol, timeframe'
);
SELECT add_compression_policy('candles', INTERVAL '7 days');
```

---

### 4.2 Database Schema

#### Migration Numbering

dwizzyBRAIN v4 already has migrations `001–038` covering core, market, arbitrage, DeFi, news, AI signal, and storage tables. Quant-specific migrations **continue from `039`**:

```
039_quant_candle_indicators.sql
040_quant_candle_features.sql
041_quant_candle_embeddings.sql
042_quant_macro_events.sql
043_quant_candle_event_labels.sql
044_quant_compression_policies.sql
045_quant_hnsw_index.sql
```

> Note: `010_ohlcv_hypertable.sql` (existing) already creates the base `candles` table.
> Quant migrations extend it with computed layers — never recreate `candles`.

#### Core Tables

```sql
-- Raw OHLCV candles (hypertable, partitioned by time)
CREATE TABLE candles (
    time        TIMESTAMPTZ NOT NULL,
    symbol      TEXT NOT NULL,          -- 'BTC/USDT', 'AAPL', 'EUR/USD', 'XAUUSD'
    timeframe   TEXT NOT NULL,          -- '1m','5m','15m','1h','4h','1d','1w'
    exchange    TEXT NOT NULL,          -- 'binance', 'bybit', 'yahoo', 'oanda'
    asset_class TEXT NOT NULL,          -- 'crypto', 'stock', 'forex', 'commodity'
    open        DECIMAL(20,8) NOT NULL,
    high        DECIMAL(20,8) NOT NULL,
    low         DECIMAL(20,8) NOT NULL,
    close       DECIMAL(20,8) NOT NULL,
    volume      DECIMAL(20,8) NOT NULL,
    PRIMARY KEY (time, symbol, timeframe, exchange)
);
SELECT create_hypertable('candles', 'time');

-- Computed technical indicators
CREATE TABLE candle_indicators (
    time        TIMESTAMPTZ NOT NULL,
    symbol      TEXT NOT NULL,
    timeframe   TEXT NOT NULL,

    -- Trend
    ema_9       DECIMAL(20,8),
    ema_21      DECIMAL(20,8),
    ema_50      DECIMAL(20,8),
    ema_200     DECIMAL(20,8),
    sma_50      DECIMAL(20,8),
    sma_200     DECIMAL(20,8),
    vwap        DECIMAL(20,8),
    supertrend  DECIMAL(20,8),
    supertrend_dir SMALLINT,            -- 1 bullish, -1 bearish
    adx         DECIMAL(8,4),
    ichimoku_tenkan  DECIMAL(20,8),
    ichimoku_kijun   DECIMAL(20,8),
    ichimoku_senkou_a DECIMAL(20,8),
    ichimoku_senkou_b DECIMAL(20,8),

    -- Momentum
    rsi_14      DECIMAL(8,4),
    rsi_2       DECIMAL(8,4),
    macd        DECIMAL(20,8),
    macd_signal DECIMAL(20,8),
    macd_hist   DECIMAL(20,8),
    stoch_k     DECIMAL(8,4),
    stoch_d     DECIMAL(8,4),
    cci_20      DECIMAL(8,4),
    roc_10      DECIMAL(8,4),
    mfi_14      DECIMAL(8,4),

    -- Volatility
    atr_14      DECIMAL(20,8),
    bb_upper    DECIMAL(20,8),
    bb_mid      DECIMAL(20,8),
    bb_lower    DECIMAL(20,8),
    bb_pct_b    DECIMAL(8,4),
    bb_width    DECIMAL(8,4),
    kc_upper    DECIMAL(20,8),
    kc_lower    DECIMAL(20,8),
    hist_vol_20 DECIMAL(8,4),

    -- Volume
    obv         DECIMAL(20,8),
    cmf_20      DECIMAL(8,4),
    volume_sma20 DECIMAL(20,8),
    volume_ratio DECIMAL(8,4),
    volume_trend DECIMAL(8,4),

    -- Support/Resistance
    pivot_classic DECIMAL(20,8),
    pivot_r1    DECIMAL(20,8),
    pivot_s1    DECIMAL(20,8),
    fib_382     DECIMAL(20,8),
    fib_500     DECIMAL(20,8),
    fib_618     DECIMAL(20,8),

    PRIMARY KEY (time, symbol, timeframe)
);
SELECT create_hypertable('candle_indicators', 'time');

-- Derived / composite features
CREATE TABLE candle_features (
    time        TIMESTAMPTZ NOT NULL,
    symbol      TEXT NOT NULL,
    timeframe   TEXT NOT NULL,

    -- Price action
    candle_body_pct     DECIMAL(8,4),
    upper_wick_pct      DECIMAL(8,4),
    lower_wick_pct      DECIMAL(8,4),
    dist_from_ema9      DECIMAL(8,4),   -- % distance
    dist_from_ema21     DECIMAL(8,4),
    dist_from_ema50     DECIMAL(8,4),
    dist_from_ema200    DECIMAL(8,4),
    dist_from_vwap      DECIMAL(8,4),
    bb_position         DECIMAL(8,4),   -- 0.0 = lower band, 1.0 = upper band
    atr_ratio           DECIMAL(8,4),   -- current ATR / 20d avg ATR

    -- Momentum slopes (3-candle look-back)
    rsi_slope           DECIMAL(8,4),
    macd_hist_slope     DECIMAL(8,4),
    obv_slope           DECIMAL(8,4),

    -- Price change windows
    change_1h           DECIMAL(8,4),
    change_4h           DECIMAL(8,4),
    change_1d           DECIMAL(8,4),
    change_1w           DECIMAL(8,4),

    -- Candlestick pattern flags (from ta-lib)
    pattern_doji            BOOLEAN,
    pattern_hammer          BOOLEAN,
    pattern_shooting_star   BOOLEAN,
    pattern_engulfing       BOOLEAN,
    pattern_morning_star    BOOLEAN,
    pattern_evening_star    BOOLEAN,
    pattern_marubozu        BOOLEAN,
    pattern_inside_bar      BOOLEAN,
    pattern_pinbar          BOOLEAN,

    -- SMC flags (computed separately)
    smc_order_block         BOOLEAN,
    smc_fvg                 BOOLEAN,
    smc_bos                 BOOLEAN,
    smc_choch               BOOLEAN,
    smc_liquidity_sweep     BOOLEAN,
    smc_premium_zone        BOOLEAN,
    smc_discount_zone       BOOLEAN,

    PRIMARY KEY (time, symbol, timeframe)
);
SELECT create_hypertable('candle_features', 'time');

-- pgvector fingerprint embeddings
CREATE TABLE candle_embeddings (
    time        TIMESTAMPTZ NOT NULL,
    symbol      TEXT NOT NULL,
    timeframe   TEXT NOT NULL,
    embedding   vector(30),             -- 30-dimensional fingerprint
    PRIMARY KEY (time, symbol, timeframe)
);
SELECT create_hypertable('candle_embeddings', 'time');

-- HNSW index for fast similarity search
CREATE INDEX idx_embedding_hnsw
    ON candle_embeddings
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
```

---

#### Event & Label Tables

```sql
-- Macro economic events master table
CREATE TABLE macro_events (
    id              SERIAL PRIMARY KEY,
    event_key       TEXT NOT NULL UNIQUE,   -- 'FOMC_2024_03_20'
    event_name      TEXT NOT NULL,          -- 'FOMC Rate Decision'
    event_type      TEXT NOT NULL,          -- 'fomc','cpi','nfp','gdp','ppi','pce'
    category        TEXT NOT NULL,          -- 'monetary_policy','inflation','employment'
    impact_level    TEXT NOT NULL,          -- 'high','medium','low'
    scheduled_at    TIMESTAMPTZ NOT NULL,
    released_at     TIMESTAMPTZ,

    -- Core data
    forecast        DECIMAL(10,4),
    actual          DECIMAL(10,4),
    previous        DECIMAL(10,4),

    -- Derived
    surprise        DECIMAL(10,4),          -- actual - forecast
    surprise_pct    DECIMAL(10,4),          -- surprise / |previous| * 100
    surprise_label  TEXT,                   -- 'massive_beat','beat','inline','miss','massive_miss'

    -- Market outcomes (populated post-event)
    btc_change_5m   DECIMAL(8,4),
    btc_change_1h   DECIMAL(8,4),
    btc_change_4h   DECIMAL(8,4),
    btc_change_1d   DECIMAL(8,4),

    source          TEXT,                   -- 'FRED','ForexFactory','manual'
    created_at      TIMESTAMPTZ DEFAULT now()
);

-- Per-candle event context labels
CREATE TABLE candle_event_labels (
    id                  BIGSERIAL PRIMARY KEY,
    time                TIMESTAMPTZ NOT NULL,
    symbol              TEXT NOT NULL,
    timeframe           TEXT NOT NULL,

    -- Proximity to nearest event
    nearest_event_id    INTEGER REFERENCES macro_events(id),
    nearest_event_type  TEXT,
    proximity_label     TEXT,               -- 'imminent','pre_near','pre_far',
                                            --  'post_near','post_mid','post_far','neutral'
    hours_to_event      DECIMAL(8,2),       -- negative = before event

    -- Surprise context (from last released event)
    last_event_id       INTEGER REFERENCES macro_events(id),
    last_surprise_label TEXT,
    last_surprise_value DECIMAL(10,4),

    -- Macro regime
    fed_rate_current    DECIMAL(6,4),
    rate_regime         TEXT,               -- 'very_high','high','medium','low_zirp'
    rate_direction      TEXT,               -- 'hiking','cutting','paused'
    cpi_value           DECIMAL(6,4),
    cpi_trend           TEXT,               -- 'accelerating','stable','cooling'

    -- Pre-event drift (only for pre_* proximity labels)
    pre_event_drift     TEXT,               -- 'strong_bullish','mild_bullish','sideways',
                                            --  'mild_bearish','strong_bearish'
    -- Volatility context
    vol_context         TEXT,               -- 'extreme_vol','high_vol','normal_vol',
                                            --  'low_vol_compression'

    -- Composite label (used for fast pattern matching filter)
    macro_environment   TEXT,               -- e.g. 'hiking|high_rates|cpi_cooling|pre_near|beat_surprise'

    created_at          TIMESTAMPTZ DEFAULT now(),
    UNIQUE(time, symbol, timeframe)
);

CREATE INDEX idx_cel_time ON candle_event_labels(symbol, timeframe, time DESC);
CREATE INDEX idx_cel_proximity ON candle_event_labels(proximity_label, nearest_event_type);
CREATE INDEX idx_cel_macro ON candle_event_labels(macro_environment);
CREATE INDEX idx_cel_regime ON candle_event_labels(rate_direction, rate_regime, cpi_trend);
```

---

### 4.3 Backfill Strategy

Backfill runs as **one-off Python scripts** inside `quant/` — separate from the real-time subscription loop in `quant/main.py`. After backfill, the real-time loop takes over.

**Phase 1 — Foundation (aligned with v4 Build Phase 1–2)**
- BTC/USDT, ETH/USDT — 4h and 1h — 5 years
- Top 10 altcoins — 1h — 3 years
- Source: CCXT via exchanges already configured in `engine/market/` (Binance/Bybit/OKX/Gate/KuCoin)
- Write directly to `candles` hypertable (existing `010_ohlcv_hypertable.sql`)

**Phase 2 — Expansion (aligned with v4 Build Phase 2)**
- Add 15m and 1d timeframes for core pairs
- Add stock indices (SPY, QQQ, DXY) via Yahoo Finance
- Add XAUUSD (Gold) and crude oil

**Phase 3 — Macro Events (new, after Phase 2)**
- Backfill FRED API: Fed Funds Rate, CPI, GDP, PCE (2015–present) → `macro_events`
- Backfill ForexFactory calendar: FOMC, NFP, CPI dates + forecast/actual

**Phase 4 — Compute (runs after Phase 1–3 complete)**
- Run `quant/indicators.py` bulk mode on all backfilled candles → `candle_indicators`, `candle_features`
- Run event labelling pipeline → `candle_event_labels`
- Run `quant/scorer.py` bulk mode → `quant_score` per candle
- Generate fingerprint vectors → load into `candle_embeddings` (pgvector)

**Backfill priority order:**
```
1. BTC/USDT 4h (5yr)    → Core pattern library baseline
2. ETH/USDT 4h (5yr)    → Largest altcoin corroboration
3. BTC/USDT 1h (3yr)    → Higher resolution patterns
4. Top 10 alts 1h (3yr) → Altcoin diversity
5. Macro events (FRED)  → Required before event labelling
6. Run labelling pipeline → Tag all candles with event context
7. Generate embeddings   → Populate pgvector HNSW
```

---

## 4.4 quant/ Service Structure

`quant/` is a standalone **Python internal service** — not part of `engine/`. It has its own Docker container (`Dockerfile.quant`), runs on an internal port, and communicates exclusively via Valkey pub/sub.

```
quant/
├── main.py            # Entry point — subscribe ch:ohlcv:raw:* → process → publish
├── indicators.py      # pandas-ta: RSI(14), MACD, BB, EMA, ATR + full catalog
├── scorer.py          # quant_score composite 0–100
├── funding.py         # Funding rate parser + sentiment (futures only)
├── anomaly.py         # Volume spike + price deviation detector
├── schema.py          # SignalOutput dataclass (mirrors shared/schema/signal.py)
├── config.py          # Env config loader
└── requirements.txt   # pandas-ta, valkey, numpy, python-dotenv, psycopg2, pgvector
```

**Real-time flow (main.py):**
```
SUBSCRIBE ch:ohlcv:raw:{symbol}:{market}:{tf}
    ↓
indicators.py  → compute RSI, MACD, BB, EMA, ATR, etc.
scorer.py      → compute quant_score (0–100)
funding.py     → parse funding rate if futures
anomaly.py     → detect spikes / deviations
    ↓
PUBLISH ch:signal:processed:{symbol}   (SignalOutput JSON)
    ↓
Write computed indicators → TimescaleDB (candle_indicators, candle_features)
Cache signal → Valkey  signal:{symbol}:{tf}  TTL 1h
```

**Bulk backfill flow (separate scripts, not main.py):**
```
python quant/backfill/fetch_ohlcv.py    → fetch + store raw candles
python quant/backfill/compute_bulk.py   → indicators on historical data
python quant/backfill/label_events.py   → event labelling pipeline
python quant/backfill/build_vectors.py  → generate + insert fingerprint embeddings
```

**SignalOutput schema** (shared between `quant/schema.py` and `shared/schema/signal.py`):
```python
@dataclass
class SignalOutput:
    symbol:       str
    timeframe:    str
    timestamp:    datetime
    close:        float
    quant_score:  float          # 0–100 composite
    rsi_14:       float
    macd_hist:    float
    bb_position:  float
    atr_14:       float
    volume_ratio: float
    funding_rate: float | None   # None for spot
    anomaly:      bool
    anomaly_type: str | None     # 'volume_spike' | 'price_deviation' | None
```

---

## 5. Technical Analysis Library

**Stack:** `pandas-ta` (primary), `ta-lib` (candlestick patterns), `numpy` (derived features)
**Location:** `quant/indicators.py` (real-time) + `quant/backfill/compute_bulk.py` (historical)

### 5.1 Indicator Catalog

#### Trend

| Indicator | Parameters | Notes |
|---|---|---|
| EMA | 9, 21, 50, 200 | Core trend filter |
| SMA | 50, 200 | Macro trend, golden/death cross |
| VWAP | Daily reset | Institutional price reference; crypto 24h rolling |
| Supertrend | (10, 3) | Trend-following signal; direction flag ±1 |
| Ichimoku | (9, 26, 52) | Multi-timeframe trend system |
| ADX | 14 | Trend strength only, not direction |

#### Momentum

| Indicator | Parameters | Notes |
|---|---|---|
| RSI | 14 | Primary overbought/oversold + divergence |
| RSI | 2 | Short-term mean reversion timing |
| MACD | (12, 26, 9) | Momentum shift, histogram slope key |
| Stochastic | (14, 3, 3) | Cycle timing in ranging markets |
| CCI | 20 | Commodity cycles; works across all asset classes |
| ROC | 10 | Raw rate of change momentum |
| MFI | 14 | Money Flow Index; requires volume |

#### Volatility

| Indicator | Parameters | Notes |
|---|---|---|
| ATR | 14 | Stop loss sizing, vol measurement |
| Bollinger Bands | (20, 2) | Expansion/contraction, squeeze detection |
| Keltner Channel | (20, 2) | Trend channel; BB inside KC = squeeze |
| BB %B | (20, 2) | Normalized position within BB (0–1) |
| BB Width | (20, 2) | Volatility compression signal |
| Historical Volatility | 20 | Realized vol % annualized |

#### Volume

| Indicator | Parameters | Notes |
|---|---|---|
| OBV | — | Cumulative volume pressure |
| VWAP | — | Also classified under trend |
| Volume SMA | 20 | Baseline for volume ratio |
| Volume Ratio | 20 | Current volume / 20-candle avg |
| CMF | 20 | Chaikin Money Flow |
| Volume Trend | 5/20 | Short SMA / Long SMA ratio |

#### Support & Resistance

| Indicator | Parameters | Notes |
|---|---|---|
| Pivot Points (Classic) | Daily | Standard R1/R2/S1/S2 levels |
| Fibonacci Retracement | 0.236–0.786 | Swing-based, updated per structure |
| Fibonacci Extension | 1.272, 1.618 | Profit target projection |
| Swing High/Low | lookback 10 | Structure reference points |

---

### 5.2 Derived Features

Computed from raw indicators — critical for fingerprint vector quality:

```python
# Price action
candle_body_pct     = abs(close - open) / (high - low + epsilon)
upper_wick_pct      = (high - max(open, close)) / (high - low + epsilon)
lower_wick_pct      = (min(open, close) - low) / (high - low + epsilon)

# Distance features (normalized as % from price)
dist_from_ema9      = (close - ema9) / close * 100
dist_from_ema21     = (close - ema21) / close * 100
dist_from_ema50     = (close - ema50) / close * 100
dist_from_ema200    = (close - ema200) / close * 100
dist_from_vwap      = (close - vwap) / close * 100

# Volatility state
bb_position         = (close - bb_lower) / (bb_upper - bb_lower + epsilon)
atr_ratio           = atr_14 / atr_sma20        # > 1.5 = elevated vol

# Momentum slopes (3-candle)
rsi_slope           = rsi[-1] - rsi[-3]
macd_hist_slope     = macd_hist[-1] - macd_hist[-3]
obv_slope           = (obv[-1] - obv[-3]) / abs(obv[-3] + epsilon)

# Price change windows
change_1h           = (close - close_1h_ago) / close_1h_ago * 100
change_4h           = (close - close_4h_ago) / close_4h_ago * 100
change_1d           = (close - close_1d_ago) / close_1d_ago * 100
change_1w           = (close - close_1w_ago) / close_1w_ago * 100
```

---

### 5.3 Candlestick Patterns

Computed via `ta-lib` `CDL*` functions. Stored as boolean flags per candle.

| Pattern | ta-lib Function | Signal Type |
|---|---|---|
| Doji | `CDLDOJI` | Indecision / reversal |
| Hammer | `CDLHAMMER` | Bullish reversal (bottom) |
| Shooting Star | `CDLSHOOTINGSTAR` | Bearish reversal (top) |
| Engulfing | `CDLENGULFING` | Strong momentum shift |
| Morning Star | `CDLMORNINGSTAR` | 3-candle bullish reversal |
| Evening Star | `CDLEVENINGSTAR` | 3-candle bearish reversal |
| Marubozu | `CDLMARUBOZU` | Strong directional momentum |
| Inside Bar | `CDLHARAMI` | Consolidation / breakout pending |
| Pinbar | `CDLHIGHWAVE` + wick ratio | SMC rejection confirmation |

---

### 5.4 SMC (Smart Money Concepts)

Custom-computed (not in standard libraries). Primary use case: crypto and forex.

| Concept | Definition | Computation Method |
|---|---|---|
| **Order Block (OB)** | Last bearish/bullish candle before impulsive move | Detect impulse, walk back to origin candle |
| **Fair Value Gap (FVG)** | 3-candle imbalance gap | `candle[i-1].high < candle[i+1].low` (bull FVG) |
| **Break of Structure (BOS)** | Price breaks swing high/low with close | Compare close vs previous swing points |
| **Change of Character (CHoCH)** | First BOS in opposite direction | BOS that counters current trend |
| **Liquidity Sweep** | Wick beyond swing high/low followed by reversal | Wick exceeds level, close back inside |
| **Premium Zone** | Upper 50% of a price range | `(close - range_low) / range_size > 0.5` |
| **Discount Zone** | Lower 50% of a price range | `(close - range_low) / range_size < 0.5` |

**SMC applicability by asset class:**

| Asset Class | SMC Applicability |
|---|---|
| Crypto | ✅ High — algorithmic market structure clear |
| Forex | ✅ High — institutional flow visible |
| Stocks | ⚠️ Medium — works on liquid large-caps |
| Commodities | ⚠️ Medium — works on futures markets |

---

## 6. Event Labelling System

Every candle in the historical database is tagged with macro event context. This allows pattern matching queries to filter by news environment.

### 6.1 Macro Events

**Priority events (high impact):**

| Event | Frequency | Source |
|---|---|---|
| FOMC Rate Decision | 8x/year | FRED + ForexFactory |
| CPI (Consumer Price Index) | Monthly | FRED (CPIAUCSL) |
| NFP (Non-Farm Payroll) | Monthly | FRED (PAYEMS) |
| PCE Deflator | Monthly | FRED (PCEPI) |
| GDP | Quarterly | FRED (GDP) |
| PPI | Monthly | FRED (PPIACO) |

**Crypto-specific events (manual/semi-automated):**
- Bitcoin halving dates
- Major exchange events (FTX collapse, etc.)
- ETF approvals/rejections
- Major regulatory actions

---

### 6.2 Proximity Labels

```python
PROXIMITY_WINDOWS = {
    "imminent":  (-4,    0),    # hours: 0–4h before release
    "pre_near":  (-24,  -4),    # 4–24h before
    "pre_far":   (-72, -24),    # 24–72h before
    "post_near": (0,    +4),    # 0–4h after release
    "post_mid":  (+4,  +24),    # 4–24h after
    "post_far":  (+24, +72),    # 24–72h after
    "neutral":   None           # > 72h from any high-impact event
}
```

---

### 6.3 Regime Labels

```python
# Fed Rate Regime
def classify_rate_regime(fed_rate: float) -> str:
    if   fed_rate >= 5.0: return "very_high"
    elif fed_rate >= 3.0: return "high"
    elif fed_rate >= 1.0: return "medium"
    else:                 return "low_zirp"

# Rate Direction (3-meeting lookback)
def classify_rate_direction(rate_series: list) -> str:
    diff = rate_series[-1] - rate_series[-3]
    if   diff > 0:  return "hiking"
    elif diff < 0:  return "cutting"
    else:           return "paused"

# CPI Trend (3-month rolling slope)
def classify_cpi_trend(cpi_series: list) -> str:
    slope = np.polyfit(range(3), cpi_series[-3:], 1)[0]
    if   slope >  0.2: return "accelerating"
    elif slope > -0.2: return "stable"
    else:              return "cooling"

# Surprise Classification (normalized by historical std)
def classify_surprise(actual, forecast, std_history) -> str:
    if forecast is None: return "no_forecast"
    deviation = (actual - forecast) / (std_history + 1e-9)
    if   deviation >  1.5: return "massive_beat"
    elif deviation >  0.5: return "beat"
    elif deviation > -0.5: return "inline"
    elif deviation > -1.5: return "miss"
    else:                  return "massive_miss"

# Pre-event drift (24h price change before event)
def classify_pre_drift(change_24h: float) -> str:
    if   change_24h >  0.03: return "strong_bullish_drift"
    elif change_24h >  0.01: return "mild_bullish_drift"
    elif change_24h > -0.01: return "sideways"
    elif change_24h > -0.03: return "mild_bearish_drift"
    else:                    return "strong_bearish_drift"

# Volatility Context
def classify_vol_context(atr_ratio: float) -> str:
    if   atr_ratio > 2.0: return "extreme_vol"
    elif atr_ratio > 1.5: return "high_vol"
    elif atr_ratio > 0.8: return "normal_vol"
    else:                 return "low_vol_compression"
```

---

### 6.4 Composite Macro Environment Label

Single concatenated string used as a fast filter before vector similarity search:

```python
def build_macro_environment(labels: dict) -> str:
    parts = [
        labels.get('rate_direction', ''),
        f"{labels.get('rate_regime', '')}_rates",
        f"cpi_{labels.get('cpi_trend', '')}",
        labels.get('proximity_label', 'neutral'),
        f"{labels.get('last_surprise_label', 'no_surprise')}_surprise"
    ]
    return "|".join(p for p in parts if p)

# Example outputs:
# "hiking|high_rates|cpi_cooling|pre_near|beat_surprise"
# "paused|very_high_rates|cpi_stable|neutral|inline_surprise"
# "cutting|medium_rates|cpi_cooling|imminent|miss_surprise"
```

---

## 7. Pattern Matching Engine

The pattern matching engine answers: *"Given current market conditions, when has this happened before — and what happened next?"*

### 7.1 Fingerprint Vector

30-dimensional normalized float vector built per candle:

```python
def build_fingerprint(row) -> list[float]:
    """
    All values normalized to [-1, 1] or [0, 1] range before storage.
    """
    return [
        # === TREND (6) ===
        clamp(row.dist_from_ema9   / 10),   # % distance, capped at ±10%
        clamp(row.dist_from_ema21  / 10),
        clamp(row.dist_from_ema50  / 15),
        clamp(row.dist_from_ema200 / 30),
        row.adx / 100,                      # already 0–100
        float(row.supertrend_dir),          # 1.0 or -1.0

        # === MOMENTUM (6) ===
        row.rsi_14 / 100,
        row.rsi_2  / 100,
        clamp(row.macd_hist / row.atr_14),  # normalized by ATR
        clamp(row.macd_hist_slope / row.atr_14),
        row.stoch_k / 100,
        clamp(row.cci_20 / 200),            # CCI range approx ±200

        # === VOLATILITY (5) ===
        row.bb_position,                    # already 0–1
        clamp(row.bb_width * 10),
        clamp(row.atr_ratio / 3),           # ratio, capped at 3x
        clamp(row.hist_vol_20 / 100),
        row.kc_position,                    # 0–1

        # === VOLUME (4) ===
        clamp(row.volume_ratio / 5),        # ratio, capped at 5x
        clamp(row.volume_trend * 2),
        clamp(row.cmf_20),                  # already -1 to 1
        clamp(row.obv_slope),

        # === PRICE ACTION (5) ===
        row.candle_body_pct,               # already 0–1
        row.upper_wick_pct,
        row.lower_wick_pct,
        clamp(row.dist_from_vwap / 5),
        clamp(row.rsi_slope / 20),

        # === EVENT CONTEXT (4) ===
        clamp(row.hours_to_event / 72),    # normalized: -1 pre, +1 post
        clamp(row.last_surprise_value / 3),
        row.rate_regime_encoded / 3,       # 0=zirp, 1=med, 2=high, 3=very_high
        row.vol_context_encoded / 3,       # 0=compression, 1=normal, 2=high, 3=extreme
    ]

def clamp(x, lo=-1.0, hi=1.0):
    return max(lo, min(hi, x))
```

---

### 7.2 Similarity Search

```sql
-- Find top 20 most similar historical candles to current fingerprint
-- with macro environment pre-filter for statistical quality

SELECT
    ce.time,
    ce.symbol,
    c.close,
    ci.rsi_14,
    ci.macd_hist,
    cel.macro_environment,
    cel.proximity_label,
    -- Outcome: price change 24h after this candle
    LEAD(c.close, 24) OVER (
        PARTITION BY c.symbol, c.timeframe
        ORDER BY c.time
    ) as close_24h_later,
    1 - (emb.embedding <=> $1::vector) as similarity_score
FROM candle_embeddings emb
JOIN candles c
    ON c.time = emb.time AND c.symbol = emb.symbol AND c.timeframe = emb.timeframe
JOIN candle_indicators ci
    ON ci.time = emb.time AND ci.symbol = emb.symbol AND ci.timeframe = emb.timeframe
JOIN candle_event_labels cel
    ON cel.time = emb.time AND cel.symbol = emb.symbol AND cel.timeframe = emb.timeframe
WHERE
    emb.symbol    = $2  AND
    emb.timeframe = $3  AND
    -- Pre-filter by macro environment similarity (optional but improves quality)
    cel.rate_direction = $4 AND
    cel.cpi_trend      = $5
ORDER BY emb.embedding <=> $1::vector
LIMIT 20;
```

---

### 7.3 Pattern Output

For each pattern match query, the engine returns:

```json
{
  "query": {
    "symbol": "BTC/USDT",
    "timeframe": "4h",
    "fingerprint": [...],
    "macro_environment": "hiking|high_rates|cpi_cooling|pre_near|beat_surprise"
  },
  "matches": {
    "count": 34,
    "sample_size_sufficient": true
  },
  "outcomes": {
    "1h":  { "median": 1.2,  "win_rate": 0.68, "avg_win": 2.8,  "avg_loss": -1.4 },
    "4h":  { "median": 3.1,  "win_rate": 0.71, "avg_win": 5.2,  "avg_loss": -2.1 },
    "1d":  { "median": 5.8,  "win_rate": 0.73, "avg_win": 8.3,  "avg_loss": -3.6 },
    "1w":  { "median": -1.2, "win_rate": 0.44, "avg_win": 9.1,  "avg_loss": -7.2 }
  },
  "top_matches": [
    {
      "date": "2023-11-10T14:00:00Z",
      "similarity": 0.94,
      "macro": "hiking|high_rates|cpi_cooling|pre_near|beat_surprise",
      "outcome_1d": 8.7
    }
  ]
}
```

**Minimum sample threshold:** 30 matches. Queries returning fewer than 30 are flagged as `low_confidence: true`.

---

## 8. External Data Sources

| Source | Data | Access | Cost |
|---|---|---|---|
| **CCXT** (Binance, Bybit) | OHLCV crypto historis + real-time | Library | Free |
| **FRED API** | Fed rate, CPI, GDP, PCE, NFP (historis) | REST API | Free |
| **Yahoo Finance** (yfinance) | Stocks, indices, forex, commodities | Library | Free |
| **ForexFactory** | Economic calendar dates, forecast/actual | Scraping | Free |
| **CoinGlass** | Crypto-specific: funding rate, OI, liquidations | API (free tier) | Free |
| **CryptoPanic** | News sentiment + headlines | API (free tier) | Free |
| **OpenRouter / Gemini Flash** | AI news summarization + sentiment scoring | API | Low cost |

**FRED Series used:**

| Series ID | Description |
|---|---|
| `FEDFUNDS` | Effective Federal Funds Rate |
| `CPIAUCSL` | CPI All Urban Consumers |
| `GDP` | Gross Domestic Product |
| `PCEPI` | PCE Price Index |
| `PAYEMS` | Total Nonfarm Payroll |
| `PPIACO` | PPI All Commodities |
| `DFF` | Daily Fed Funds Rate |

---

## 9. Knowledge & Documentation Layer

Data that lives outside the database — human-readable, shareable via Notion and Google Sheets.

### Google Sheets (Aggregate / Review Data)

| Sheet | Content |
|---|---|
| `Backtest Results` | Strategy name, win rate, avg return, max drawdown, Sharpe, tested period |
| `Event Calendar` | Date, event, forecast, actual, surprise, BTC reaction |
| `Trading Journal` | Date, symbol, entry/exit, PnL, strategy, notes |
| `Asset Watchlist` | Symbol, sector, market cap, 7d/30d change, status |
| `PnL Tracker` | Monthly/weekly performance summary |

### Notion (Strategy & Research Docs)

| Database | Content |
|---|---|
| `Strategy Library` | Each strategy: thesis, entry/exit rules, risk management, backtest link |
| `Pattern Library` | Documented patterns: description, conditions, historical examples, screenshots |
| `Macro Research` | Fed cycle analysis, CPI thesis, BTC halving notes |
| `Trade Reviews` | Per-trade retrospective: setup, decision, outcome, lesson |
| `Architecture Docs` | System design, data flow diagrams, schema references |

**Rule:** Notion/Sheets = human-readable summaries and documentation only. Never raw time-series data.

---

## 10. Implementation Roadmap

Quant build is **additive on top of** the v4 main build phases. It does not replace them.

### Pre-requisite: v4 Build Phase 1 (Engine Core) must be complete
- `engine/market/ws/` + `ccxt.go` must be publishing `ch:ohlcv:raw:*`
- Migrations `001–013` must be applied (core + market + arbitrage schema)
- `010_ohlcv_hypertable.sql` (base `candles` table) must exist

---

### Quant Phase 1 — Service Bootstrap (alongside v4 Phase 1 Day 3)
- [x] `quant/main.py` — subscribe `ch:ohlcv:raw:*`, basic pipeline loop
- [x] `quant/indicators.py` — RSI(14), MACD, BB, EMA, ATR (core set first)
- [x] `quant/scorer.py` — quant_score composite (0–100)
- [x] `quant/schema.py` — SignalOutput dataclass
- [x] `quant/funding.py` — funding rate parser (futures symbols)
- [x] `quant/anomaly.py` — volume spike + price deviation
- [x] `Dockerfile.quant` wire-up in `docker-compose.yml`
- [x] `quant` env contract + healthcheck CLI mode
- [x] Apply migrations `060–062` (`candle_indicators`, `candle_features`, `candle_embeddings`) — PRD-QUANT equivalent of `039–041`

### Quant Phase 2 — Full Indicator Suite + Backfill
- [x] Complete `quant/indicators.py` — all 60+ indicators from Section 5
- [x] `quant/backfill/fetch_ohlcv.py` — CCXT backfill BTC/ETH 4h+1h 5 years
- [x] `quant/backfill/fetch_yfinance.py` — Yahoo Finance backfill for stocks, forex, commodities
- [x] `quant/backfill/compute_bulk.py` — run indicators on historical candles
- [x] Cold tier: Parquet export script + DuckDB query layer

### Quant Phase 3 — Event System
- [x] Apply migrations `063–064` (`macro_events`, `candle_event_labels`) — PRD-QUANT equivalent of event backfill tables
- [x] `quant/backfill/fetch_fred.py` — FRED API backfill 2015–present
- [x] ForexFactory scraper — event calendar with forecast/actual
- [x] `quant/backfill/label_events.py` — full event labelling pipeline
- [x] Composite `macro_environment` string generation

### Quant Phase 4 — Pattern Engine
- [x] Apply migration `044–045` (compression policies + HNSW index) — PRD equivalent already live via `062`
- [x] `quant/backfill/build_vectors.py` — fingerprint builder + pgvector bulk load
- [x] Similarity search query + outcome aggregation
- [x] Pattern match endpoint in `api/handler/` (new `quant.go`)
- [x] Minimum sample confidence gate (30+ matches)
- [x] dwizzyBOT command: `/pattern BTC/USDT 4h`

### Quant Phase 5 — Expansion + ML (v2)
- [x] Expand backfill to stocks (Yahoo Finance), forex, commodities
- [x] SMC computation layer
- [x] Outcome prediction model (XGBoost / LightGBM) on fingerprint + labels
- [x] RTX 5060 Ti local embedding generation (replace OpenRouter)

---

## 11. Schema Update Log

| Version | Date | Changes |
|---|---|---|
| v1.0.0 | 2026-03-19 | Initial PRD-QUANT creation. TA library (8 categories, 60+ features), Event Labelling System, Pattern Matching Engine (30-dim fingerprint, pgvector HNSW), Storage Strategy, Backfill Strategy, External data sources, Knowledge layer. |
| v1.1.0 | 2026-03-19 | **Corrected against dwizzyBRAIN v4 structure.** Renamed dwizzyOS → dwizzyBRAIN. Corrected architecture: `quant/` is a standalone Python service (not inside `engine/`), communicates via Valkey pub/sub. Added Section 4.4 quant/ service structure + SignalOutput schema. Fixed storage tier table (Telegram file_id, Cloudflare R2, Google Drive strategy). Fixed migration numbering: quant migrations start at `039` (v4 already has `001–038`). Added note that `candles` hypertable already exists in `010_ohlcv_hypertable.sql`. Updated roadmap to align with v4 build phases. Added Valkey channel map. Fixed `engine/storage_ext/` as the backup mechanism. |

> **Note:** This PRD-QUANT.md is an addendum to the main dwizzyBRAIN v4 structure document. It covers the quant intelligence layer only. For market pipeline, DeFi, news, irag, auth, and deployment — refer to `dwizzyBRAIN-structure-v4.pdf` and the docs in `dwizzyBRAIN/docs/`.

---

*PRD-QUANT.md — dwizzyBRAIN Quant Engine — Rijal*
*Document lives in: `dwizzyBRAIN/docs/PRD-QUANT.md`*
