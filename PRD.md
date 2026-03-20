# dwizzyBRAIN - Comprehensive Product Requirements Document (PRD)

## 1. Executive Summary
**dwizzyBRAIN** is the core backend engine and API gateway for the dwizzyOS ecosystem. It is a highly concurrent, multi-pipeline system built primarily in Go, with supporting Python services for quantitative analysis. It aggregates, normalizes, and serves cryptocurrency market data, decentralized finance (DeFi) analytics, AI-curated crypto news, and real-time arbitrage signals. 

In addition to crypto, it features the **Indonesian REST API Gateway (IRAG)**, which wraps and standardizes hundreds of endpoints from 5 different upstream providers for general-purpose tools (downloaders, AI, BMKG, streaming, etc.).

## 2. Core Objectives
- **Unified Market Data:** Provide a 3-tier (Hot/Warm/Cold) availability architecture for 1000+ coins, abstracting away exchange-specific differences into a single unified API response.
- **DeFi & News Intelligence:** Track TVL, DEX volumes, stablecoin depegs, and process raw crypto news through LLMs for sentiment, importance scoring, and price impact analysis.
- **Real-Time Arbitrage:** Scan order books every 5 seconds across major exchanges to detect cross-exchange price discrepancies and fire alerts.
- **IRAG Wrapper:** Expose 180+ standardized endpoints from upstream providers (KanataAPI, Nexure, Ryzumi, Chocomilk, YTDLP) with high reliability through L1 (Valkey) and L2 (TimescaleDB) caching, combined with automatic fallback chains across providers.
- **Cost-Efficiency:** Optimize API usage through multi-tiered caching and fallback chains, minimizing upstream costs to near $0.

### 2.1 Strategic Direction (Launch + Core Features)

#### Product Positioning
**dwizzyBRAIN launches as the reliability-first crypto intelligence backend** for builders who need one API for live market data, decision signals, and news context without operating complex ingestion infrastructure.

#### Launch Thesis
Adoption depends on winning a single job-to-be-done better than alternatives:
1. **Fragmented crypto data:** users waste time stitching exchange, DeFi, and news sources.
2. **Operational complexity:** real-time ingestion and normalization are expensive to run reliably.
3. **Low decision utility:** raw feeds exist everywhere, but high-quality, usable signals do not.

Launch strategy:
1. **Trust first:** consistently accurate and fresh market responses.
2. **Utility second:** clear signal layers (arbitrage + news impact + DeFi stress indicators).
3. **Friction last:** fast integration with stable API contracts and clear plan tiers.

#### Primary Launch Segment
1. **Primary ICP (must win):** crypto bot builders, small quant teams, and dashboard developers.
2. **Secondary ICP (adjacent):** crypto research communities and alpha groups.
3. **Not target at launch:** institutions needing execution rails, compliance modules, or custom SLAs.

#### Core Feature Set for Launch (Must Ship)
1. **Unified Market API (`/v1/market/*`)**
   - Tiered coverage (A/B/C), symbol normalization, and freshness guarantees.
   - Stable REST response envelopes and premium real-time WebSocket feed.
2. **Arbitrage Signal Engine**
   - 5-second scanning with spread/depth filters and de-duplicated alerts.
3. **News Intelligence API (`/v1/news/*`)**
   - Ingestion, summarization, entity tagging, and importance scoring.
   - Price-impact snapshots at T+0, T+1h, T+4h, and T+24h.
4. **DeFi Core Analytics (`/v1/defi/*`)**
   - TVL trends, DEX volume tracking, and stablecoin/depeg monitoring.
5. **Freemium Access Control**
   - Free REST tier for adoption; premium tier for low-latency streams and advanced signals.

#### Launch Non-Goals (Post-Launch)
1. Expanding IRAG utility endpoints as the main growth lever.
2. Portfolio management and order execution features.
3. Automated strategy execution or bot hosting.
4. Enterprise sales motion and custom deployment SKUs.

#### Launch Success Metrics (First 90 Days)
1. **Reliability:** `>=99.5%` successful responses on core market endpoints.
2. **Latency:** p95 `<250ms` cached reads, p95 `<1200ms` uncached reads.
3. **Data Freshness:** Tier A ticker freshness `<5s`, Tier B `<60s`.
4. **Adoption:** 20+ active API integrators and 5+ paying premium teams/users.
5. **Signal Utility:** at least 30% of active users call both market + news/defi routes weekly.

#### Launch Gates (Must Pass Before Public Push)
1. **Reliability gate:** 14 consecutive days with SLO compliance on `/v1/market/*`.
2. **Data-quality gate:** no unresolved Tier A symbol-mapping errors across top 100 assets.
3. **Utility gate:** arbitrage/news endpoints produce non-empty outputs on live traffic windows.
4. **Monetization gate:** premium auth + usage metering is active and validated.

#### Strategic Sequencing
1. **Phase A - Foundation (Weeks 1-2):** market ingestion reliability, schema hardening, SLO instrumentation.
2. **Phase B - Signal Layer (Weeks 3-4):** arbitrage quality, news impact scoring, DeFi health signals.
3. **Phase C - Packaging (Weeks 5-6):** premium gating, usage limits, plan definitions, onboarding docs.
4. **Phase D - Controlled Launch (Week 7+):** private beta to ICP users, weekly feedback loops, rapid fixes.

#### Strategic Principle
For roadmap decisions, optimize for **trustworthiness + decision utility per endpoint**.  
A narrower API that is consistently accurate, fresh, and useful will outperform a broad surface with uneven quality.

## 3. Detailed System Architecture & Components

### 3.1 Services Map & Responsibilities
| Service | Language | Port | Public | Primary Responsibility |
|---------|----------|------|--------|----------------------|
| **Engine** | Go | internal | ❌ | Core data ingestion, WS clients, polling, arbitrage engine, AgentRouter, scheduler. |
| **API** | Go | 8080 | ✅ | Serves `/v1/market`, `/v1/defi`, `/v1/news`, WebSocket Hub, rate limiter, auth middleware. |
| **IRAG** | Go | 8081 | ✅ | Gateway for general API utility, acts as a unified facade for 5 upstream providers. |
| **Quant** | Python | internal | ❌ | Subscribes to raw OHLCV over Valkey, computes indicator signals (RSI/MACD/BB/EMA). |
| **Auth** | Worker (TS)| Edge | ✅ | JWT issuer handling Discord OAuth & Web3 wallet signatures (Cloudflare). |

### 3.2 Tiered Storage Strategy Detailed
1. **PostgreSQL** (Core relational data):
   - Handles identities: `coins`, `cold_coin_data`, `developer_data`, `defi_protocols`, `news_articles`.
   - Mappings: `coin_exchange_mappings`, `coin_coverage` (Tier A/B/C/D).
2. **TimescaleDB** (Time-series data, heavy write/read):
   - Market: `tickers_hypertable`, `ohlcv_hypertable`, `exchange_spread_history`.
   - DeFi limits: `defi_protocol_tvl_history`, `defi_dex_volume_history`.
   - IRAG: `irag_request_log` (L2 cache).
3. **Valkey / Redis** (Hot Cache & PubSub):
   - **Hot Cache:** `price:{symbol}:{market}` (10s), `orderbook:{exchange}:{symbol}` (5s).
   - **Warm Cache:** `warm_prices:{coin_id}` (10m).
   - **PubSub Channels:** `ch:ohlcv:raw`, `ch:signal:processed`, `ch:ai:result`, `ch:arb:signal`.
4. **External Storage** (Persistence & Backups):
   - **Cloudflare R2:** Logo CDN.
   - **Telegram:** Free `file_id` caching for charts and exports.
   - **Google Drive / Storj:** Automated `pg_dump` and cold exports.

## 4. Subsystem Deep-Dives

### 4.1 Market Data Pipeline
- **Availability Tiers:**
  - **Tier A (Top 100):** Real-time Hot + Warm + Cold. Fetched via Binance / Bybit WS.
  - **Tier B (Top 501):** Warm + Cold. Fetched via Binance REST polling. Fallback: CoinPaprika.
  - **Tier C (Top 1000):** Cold only, often DEX-exclusive. Fallback: DexScreener.
- **Transport Strategy (Hybrid):**
  - **Native WebSocket first** for hot-path exchanges and latency-sensitive streams (initially Binance, Bybit, OKX), especially for tickers and order books used by arbitrage.
  - **CCXT REST** remains the standard fallback for minor exchanges, backfill, and non-critical polling.
  - **CCXT WebSocket / Pro adapter** may be introduced selectively for non-critical exchanges or lower-priority streams, but it is not the default ingestion path for the hot tier.
  - This hybrid model avoids over-coupling the real-time path to CCXT websocket behavior while preserving a unified symbol and ticker normalization layer above transport.
- **Symbol Mapping Engine:** Bidirectional mapping translating universal `coin_id` (e.g. `bitcoin`) to exchange-specific pairs (e.g. `BTCUSDT`). It prefers USDT > USDC > BUSD > BTC.
- **On-Demand Enrichment:** If a coin detail is queried and price is missing, a non-blocking background goroutine fetches from the fallback chain, ensuring the next visitor gets cached data.

### 4.2 Arbitrage Engine
- **Scanning Mechanism:** Runs every 5 seconds reading the hottest Valkey orderbook data.
- **8-Step Detection:** Cross-references `best_bid` across exchanges against `best_ask` on others, factoring in minimum spread% + minimum depth config (`arbitrage_config`), then alerts via Discord.

### 4.3 DeFi Analytics Pipeline
- **Protocol & TVL Sync:** Polls top 500 DeFi protocols and stores snapshots (`defi_protocol_tvl_latest`). Periodically flushes to TimescaleDB history.
- **DEX & Stablecoins:** Hourly snapshots of top 30 DEX volumes, mapping of stablecoin market caps, and backing composition (Circle/Tether). Built-in **Depeg Scanner** checks if `abs(price-1) > 0.01` and alerts instantly.
- **Hacks / Exploits Database:** Synchronizes daily from `defihack-db` GitHub to augment protocol risk scoring.

### 4.4 News & AI Processing Pipeline
- **Ingestion:** CryptoPanic WS (breaking), RSS pollers (10m/30m), CoinGecko News.
- **AI Task Processor:** Ingested articles are queued into `news:pending_ai` (Valkey). The **AgentRouter** picks them up and dispatches to an LLM to:
  1. Generate bullet point summaries.
  2. Tag entities (coins / protocols).
  3. Determine sentiment.
  4. Compute `importance_score` based on credibility + panic + votes + sentiment.
- **Price Impact Tracker:** Records price snapshot instantly, 1h, 4h, and 24h after a major news publication to evaluate correlation.

### 4.5 Indonesian REST API Gateway (IRAG) Wrapper
- **Upstream Providers:**
  1. **KanataAPI:** High reliability, $0 (Primary for YouTube, BMKG, Quran).
  2. **Nexure API:** Primary for AI (19 models), Universal Downloader, Instagram.
  3. **Ryzumi API:** Primary for Search (Google, Pinterest, Spotify), Stalk endpoints.
  4. **Chocomilk:** Primary for Novels, NSFW checks, Tidal, Twitter.
  5. **YTDLP API:** Key-required premium API for Apple Music, Subtitles, Grow A Garden data.
- **Fallback Chain Logic:** Each logical endpoint has an ordered priority. E.g. TikTok Download tries: Nexure ➔ KanataAPI ➔ Ryzumi. If one fails, circuit breaker opens, moving to the next.
- **Unified JSON Envelope:** Regardless of upstream messiness, clients reliably receive:
  `{ "ok": true, "code": 200, "data": {...}, "meta": {"latency_ms": ...} }`

### 4.6 AI AgentRouter (LLM Dispatch)
Task dispatch logic to minimize cost while retaining capabilities, ordered by priority:
1. **Priority 1-4 ($0 Cost):** Uses IRAG's wrapped Nexure endpoints (`groq`, `gemini`, `deepseek`, `qwen`) for basic formatting and summarization.
2. **Priority 5-6 (Low Cost):** Direct Groq/Gemini API calls for RAG, complex reasoning, or when IRAG rate-limits.
3. **Priority 7 (Variable):** OpenRouter as the absolute last resort fallback.

## 5. API Surface
- **`/v1/market/*`**: Real-time prices, orderbooks, OHLCV, and arbitrage signals. (Freemium / Premium)
- **`/v1/defi/*`**: TVL, chains, DEX volumes, stablecoin analytics. (Public)
- **`/v1/news/*`**: News feeds, trending topics, entity sentiment, price impact. (Public)
- **`/v1/ai/*`**: AI logic integrations (Summarize, Analyze, Process Image) wrapping Groq, Gemini, and IRAG models. (Premium for core AI, Free for utility wrapper AI).
- **`/v1/download/*`, `/v1/search/*`, `/v1/tools/*`**: General IRAG utility endpoints backed by the KanataAPI / Nexure / Ryzumi / Chocomilk / YTDLP wrapper chain. (Free)
- **`/v1/ws/market`**: Live WebSocket multiplexed streams for orderbooks, tickers, and platform alerts.

## 6. Security & Access Control
- **Freemium Model:** Public endpoints (e.g., `/v1/market`, `/v1/defi`) are available generally. Premium features (Real-time WS, Arbitrage, Agent Analysis) require a JWT.
- **Web3 Auth:** Nonce generation and `personal_sign` verification using Ethereum/Polygon networks to check on-chain `SubscriptionManager.sol` payment status.
- **Rate-Limiting:** Strict token-bucket rate limiting (default 100 RPM limit per endpoint category).

## 7. Database Schema Reference

The primary relational datastore is PostgreSQL with TimescaleDB extensions for time-series data. 

### 7.1 Core Schema
- `001_init_extensions`: TimescaleDB, pgvector, PostGIS extensions.
- `002_coins`: Canonical registry for all trackable cryptocurrencies.
- `003_cold_coin_data`: ATH/ATL, descriptions, links, categories.
- `004_developer_data`: GitHub commit & star statistics.
- `005_coin_exchange_mappings`: `coin_id` ↔ exchange symbol crosswalk.
- `006_coin_coverage`: Tier tracking (A/B/C/D) and availability per exchange.
- `007_coin_data_completeness`: Field-level completeness scores.
- `008_unknown_symbols`: Catch-all queue for WS symbols lacking mappings.

### 7.2 Market Time-Series (TimescaleDB)
- `009_tickers_hypertable`: Real-time tickers per coin per exchange.
- `010_ohlcv_hypertable`: OHLCV candles (1m, 5m, 1h, 1d) with data retention policies.
- `011_exchange_spread_history`: Spread snapshots (30 days retention, 1 day compression).

### 7.3 Arbitrage
- `012_arbitrage_signals`: Historical detected signals and generated discrepancies.
- `013_arbitrage_config`: Core thresholds (`min_spread`, `min_depth`, `cooldown`).

### 7.4 DeFi Analytics
- **Snapshots:** `014_defi_protocols`, `015_defi_protocol_tvl_latest`, `016_defi_chain_tvl_latest`, `017_defi_dex_latest`, `018_defi_protocol_coverage`.
- **Anomalies / Stables:** `019_defi_hacks` (defihack-db sync), `020_defi_stablecoin_backing`, `021_defi_alert_config`.
- **Time-Series (TimescaleDB):** `022_defi_protocol_tvl_history`, `023_defi_dex_volume_history`, `024_defi_chain_tvl_history`, `025_defi_stable_mcap_history`, `026_defi_fees_history`.

### 7.5 News & Processing
- **Operational:** `027_news_articles`, `028_news_ai_metadata`, `029_news_entities`, `030_news_sources`, `032_news_trending_cache`, `033_news_alert_config`.
- **Impact Metrics:** `031_news_price_impact`, `034_news_price_impact_history` (TimescaleDB).

### 7.6 AI Signals & Storage Logging
- **Agent Outputs:** `035_signals_table` (quant indicators), `036_ai_results_table` (LLM summaries/decisions).
- **Storage/Logs:** `037_telegram_file_cache` (charts/exports), `038_irag_request_log` (L2 cache & metrics for IRAG wrappers).

## 8. Implementation Rollout Plan

The delivery of dwizzyBRAIN is structured into 5 cohesive execution phases.

### Phase 1: Engine Core (Week 1 / Day 1–5) - Detailed Development Guide
The primary goal of Phase 1 is to establish the raw data pipelines, storage mechanics, and LLM routers.

#### Day 1: Websocket & REST Ingestion (`engine/market/ws/`, `ccxt.go`)
**Objective:** Ingest real-time and fallback price data.
- **Dependencies:** `github.com/gorilla/websocket`, native JSON parsing, `ccxt/go/v4`.
- [x] **Implementation 1: `engine/market/ws/binance.go`**
  - Create standard `BinanceWSClient` struct with `Connect()`, `ReadMessage()`, and `Close()` methods.
  - Subscribe to `!ticker@arr` (All Market Tickers Stream).
  - Implement a `context.Context` with a 23-hour timeout to force a clean reconnect (handling Binance's 24h drop policy).
- [x] **Implementation 2: `engine/market/ccxt.go`**
  - Create `CCXTManager` utilizing `ccxt` to poll minor exchanges via REST API equivalents.
- [x] **Implementation Note: Hybrid Ingestion Policy**
  - Keep native WebSocket clients as the primary hot-path transport for major exchanges.
  - Use CCXT REST as the default fallback and backfill transport.
  - Treat CCXT WebSocket / Pro as an optional adapter layer for selected non-critical exchanges or experimental streams, not as a mandatory replacement for native WS clients.
- [x] **Struct Normalization (`shared/schema/market.go`):**
  - Create a unified `RawTicker` struct: `{Symbol, Price, Exchange, Bid, Ask, Volume, Timestamp}` to ensure the rest of the app doesn't deal with exchange-specific JSON.

#### Day 2: Valkey Publisher & Storage (`engine/market/publisher.go`)
**Objective:** Cache incoming ticks and broadcast them to the Quant service.
- **Dependencies:** `github.com/valkey-io/valkey-go` (or `redis/go-redis/v9`).
- [x] **Implementation 1: Valkey Client Setup (`engine/storage/valkey.go`)**
  - Initialize the connection pool using the `VALKEY_URL` env var.
- [x] **Implementation 2: Hot Cache (`publisher.go`)**
  - On receiving `RawTicker`, execute `SET price:{symbol}:{exchange} {price} EX 10`.
  - For OHLCV pipelines, execute `LPUSH ohlcv:{symbol}:{exchange}:{tf} {payload}` and cap it using `LTRIM ... 0 199` to retain only the last 200 candles in memory.
- [x] **Implementation 3: PubSub Broadcaster**
  - Execute `PUBLISH ch:ohlcv:raw:{symbol}:{exchange}:{tf} {payload}` for the Python daemon to consume.

#### Day 3: Python Quantitative Engine (`quant/`)
**Objective:** Process raw OHLCV to compute technical indicators independently.
- **Dependencies:** `pandas`, `pandas-ta`, `redis-py`.
- [x] **Implementation 1: `main.py` & PubSub Listener**
  - Connect to Valkey using `redis.Redis(decode_responses=True)`.
  - Subscribe to `ch:ohlcv:raw:*` using `pubsub.listen()`.
  - Batch incoming JSON signals into a Pandas DataFrame.
- [x] **Implementation 2: Technical Indicators (`indicators.py`)**
  - Apply `pandas_ta`: `df.ta.rsi(length=14)`, `df.ta.macd()`, `df.ta.bbands()`, `df.ta.ema()`.
- [x] **Implementation 3: Scoring & Output (`scorer.py`)**
  - Calculate `quant_score` (0-100) based on momentum and volatility heuristics.
  - Publish final JSON containing indicators back to Valkey: `PUBLISH ch:signal:processed:{symbol} {json}`.

#### Day 4: Core Database Schema & Migrations (`migrations/`)
**Objective:** Setup PostgreSQL and TimescaleDB foundations securely.
- **Dependencies:** `golang-migrate/migrate` or raw SQL scripts, `jackc/pgx/v5`.
- [x] **Implementation 1: Extensions & Base Tables**
  - Execute `001_init_extensions.sql` (requires superuser for `CREATE EXTENSION timescaledb`).
  - Create `coins` registry table (`002_coins.sql`) with `id, symbol, name, rank`.
- [x] **Implementation 2: TimescaleDB Hypertables**
  - `009_tickers_hypertable.sql`: Create table `tickers (time TIMESTAMPTZ, coin_id TEXT, price NUMERIC)` and invoke `SELECT create_hypertable('tickers', 'time')`.
  - `010_ohlcv_hypertable.sql`: Create the `ohlcv` hypertable similarly.
- [x] **Implementation 3: Arbitrage Schema**
  - `012_arbitrage_signals.sql`: Store historical cross-exchange discrepancies.
  - `013_arbitrage_config.sql`: Table for tracking `min_spread`, `cooldown`.

#### Day 5: AI Agent & LLM Router (`engine/agent/`)
**Objective:** Centralized dispatch for LLM tasks utilizing the most cost-effective provider.
- **Dependencies:** `github.com/go-resty/resty/v2` (for robust HTTP client logic).
- [x] **Implementation 1: `Provider` Interface & Standardizer**
  - Define `LLMProvider` interface in `engine/agent/provider.go` with `Ask(ctx, prompt) (string, error)`.
- [x] **Implementation 2: IRAG Upstream & Fallbacks (`providers/*.go`)**
  - Implement Priority 1-4 wrappers sending requests to `IRAG_INTERNAL_URL + /v1/ai/text/...` ($0 cost).
  - Implement Priority 5-6 wrappers for direct Groq/Gemini HTTP calls.
- [x] **Implementation 3: `AgentRouter` Logic (`router.go`)**
  - Iterate through `[]LLMProvider` ordered by priority. If IRAG times out or returns HTTP 429, elegantly failover to Groq.
  - Implement Redis `SETNX` lock (`inflight:ai:{resource_id}`) to uniquely lock duplicate identical LLM requests triggering simultaneously.

### Phase 2: Market Full (Week 2)
- [x] Implement `SymbolResolver` and runtime ingestion wiring.
- [x] Implement `MappingBuilder`, `MappingValidator`, unknown symbol resolution, and scheduled mapping sync job.
- [x] Deploy OHLCV backfill worker, incremental sync service, Timescale integration, and engine one-shot execution path.
- [x] Implement Ticker aggregators and Spread recorders.
- [x] Implement `GapDetector`.
- [x] Activate 5-second `ArbitrageEngine` with Discord alert dispatch.
- [x] Configure CoinGecko 1000-coin cold loader (24h loop).
- [x] Expose `/v1/market` and `/v1/market/{id}` endpoints to the public API.
- [x] Expose `/v1/market/{id}/ohlcv`, `/v1/market/{id}/tickers`, and `/v1/market/{id}/arbitrage` endpoints to the public API.
- [x] Publish standardized OpenAPI documentation at `/openapi.json`.
- [x] Serve browser-friendly API docs UI at `/docs`.
- [x] Expose discoverability links from `/`.

### Phase 3: DeFi (Week 3)
- [x] Construct DeFi protocol registry and expose the first `/v1/defi` read surface.
- [x] Add TVL sync workers and history backfill.
- [x] Add Yields tracking.
- [x] Add Stablecoin tracking and depeg scanning.
- Run DeFi migrations (`014–026`).
- Expose `/v1/defi/` endpoint family.

### Phase 4: News (Week 4)
- [x] Setup RSS Poller and raw article persistence.
- [x] Implement heuristic AI batch processor logic for summaries, entities, sentiment, and importance scoring.
- [x] Expose `/v1/news/` endpoints and sentiment scoring graphs.
- [x] Implement price-impact tracking worker and history tables.
- [x] Export news articles to Google Drive as Obsidian-style Markdown notes and persist title + share link in Postgres.
- [ ] Run CryptoPanic WS+REST as optional integration only when an API key is available.
- [x] Run News migrations (`027–034`) plus live compatibility tables for the current schema.

### Phase 5: Auth, Storage & End-to-End (Week 5–6)
- [x] Launch Discord OAuth login/session issuance in the API auth layer.
- [x] Add Web3 EVM validation.
- [x] Add premium gating middleware and on-chain plan resolver for `SubscriptionManager.sol`.
- [x] Add multi-chain `SubscriptionManager.sol` scaffolding and deploy tooling for Base, BSC, Kaia, Arbitrum, Sonic, Abstract, and HyperEVM.
- [x] Deploy `SubscriptionManager.sol` to Base, BSC, Kaia, and Arbitrum to gate premium features. Sonic/Abstract/HyperEVM remain pending until the deploy wallet is funded there.
- [x] Wire up external storage bridges (Google Drive backup, Telegram `file_id`, Cloudflare R2).
- [x] Deploy Cloudflare Workers (auth-worker and irag-fallback).
- Final integration with frontend dwizzyDBSD dashboards.

## 9. Directory Structure & File Functions

The `dwizzyBRAIN` monorepo separates its functionalities into highly cohesive modules, allowing independent scaling as microservices.

```text
dwizzyBRAIN/
│
├── .github/
│   └── workflows/
│       └── ci.yml
│
├── engine/                                      # Go — internal engine (tidak exposed)
│   ├── cmd/
│   │   └── engine/
│   │       └── main.go                          # entry point engine service
│   │
│   ├── market/                                  # ── MARKET DATA PIPELINE ──
│   │   ├── ccxt.go                              # CCXTManager: Gate/KuCoin/Kraken/MEXC/HTX REST fallback via ccxt/go/v4
│   │   ├── symbols.go                           # symbol config — whitelist, timeframes, market types
│   │   ├── filter.go                            # big cap whitelist: BTC/USDT, ETH/USDT (spot + futures)
│   │   ├── publisher.go                         # Valkey writer — SET price, LPUSH ohlcv, publish channel
│   │   │
│   │   ├── ws/                                  # Native WebSocket clients (Binance, Bybit, OKX) for hot-path streams
│   │   │   ├── binance.go                       # !ticker@arr + depth WS — reconnect tiap 23 jam
│   │   │   ├── bybit.go                         # tickers + orderbook.200ms WS
│   │   │   └── okx.go                           # tickers + books WS
│   │   ├── adapters/                            # Optional transport adapters (e.g. CCXT websocket/pro for non-critical streams)
│   │   │   └── ccxt_watch.go                    # experimental unified watch adapter; not primary for hot tier
│   │   │
│   │   ├── ohlcv/
│   │   │   ├── service.go                       # OHLCVService: GetOHLCV, BackfillOHLCV, IncrementalSync, PollShortTF, CleanupExpired
│   │   │   └── scheduler.go                     # 1m poll (top 50), 1h incremental, 24h cleanup
│   │   │
│   │   ├── orderbook/
│   │   │   ├── service.go                       # OrderBookService: SubscribeNativeWS, GetOrderBook, PollMinorExchanges
│   │   │   └── snapshot.go                      # Valkey write TTL 5s (native) / 30s (CCXT REST)
│   │   │
│   │   ├── ticker/
│   │   │   ├── aggregator.go                    # TickerAggregator: BestBid, BestAsk, MaxSpreadPct, VolumePct
│   │   │   └── spread.go                        # RecordSpread tiap 5 menit → exchange_spread_history
│   │   │
│   │   ├── arbitrage/
│   │   │   ├── engine.go                        # ArbitrageEngine: scan tiap 5 detik, 8-step detection
│   │   │   ├── config.go                        # arbitrage_config: min_spread, min_depth, cooldown per coin
│   │   │   └── alert.go                         # Discord embed alert + cooldown Valkey key
│   │   │
│   │   ├── coingecko/
│   │   │   ├── fetcher.go                       # CoinGeckoFetcher: 4 halaman × 250 coin, sleep 2s, retry 429
│   │   │   └── scheduler.go                     # cold load 24 jam
│   │   │
│   │   ├── coinpaprika/
│   │   │   └── fetcher.go                       # warm fallback REST — price + market data
│   │   │
│   │   ├── dexscreener/
│   │   │   └── fetcher.go                       # DEX-only token price lookup
│   │   │
│   │   ├── mapping/
│   │   │   ├── resolver.go                      # SymbolResolver: coin_id ↔ exchange_symbol, Valkey TTL 1h
│   │   │   ├── builder.go                       # MappingBuilder: auto-build dari Binance exchangeInfo + CoinGecko
│   │   │   ├── validator.go                     # MappingValidator: weekly verify semua active mapping
│   │   │   └── unknown.go                       # UnknownSymbolResolver: hourly auto-resolve WS unknown symbols
│   │   │
│   │   ├── coverage/
│   │   │   ├── tier.go                          # GapDetector: assign tier A/B/C/D per coin
│   │   │   └── enricher.go                      # Enricher: on-demand background fetch missing price data
│   │   │
│   │   └── merger.go                            # Merger: parallel fetch results → CoinDetail unified response + availability map
│   │
│   ├── defi/                                    # ── DEFI DATA PIPELINE ──
│   │   ├── protocols/
│   │   │   ├── registry.go                      # defi_protocols: top 500, coin_id mapping, match_confidence
│   │   │   ├── tvl.go                           # TVL sync 1 jam: defi_protocol_tvl_latest
│   │   │   ├── history.go                       # FullHistoryBackfill: top 50 full history → TimescaleDB
│   │   │   ├── coverage.go                      # defi_protocol_coverage: tier top50/top30/on-demand
│   │   │   └── fees.go                          # Fees + revenue sync dari /summary/fees/{protocol}
│   │   │
│   │   ├── chains/
│   │   │   ├── sync.go                          # Chain TVL sync 1 jam: defi_chain_tvl_latest
│   │   │   └── history.go                       # Top 15 chain full history → TimescaleDB
│   │   │
│   │   ├── dex/
│   │   │   ├── sync.go                          # DEX volume sync 15 menit: defi_dex_latest
│   │   │   ├── history.go                       # Top 30 DEX full history → TimescaleDB
│   │   │   └── pairs.go                         # The Graph subgraph: Uniswap/SushiSwap trading pairs
│   │   │
│   │   ├── stables/
│   │   │   ├── sync.go                          # Stablecoin mcap + price sync
│   │   │   ├── depeg.go                         # DepegScanner: abs(price-1) > 0.01 → alert
│   │   │   └── backing.go                       # Backing composition: Circle/Tether attestation API
│   │   │
│   │   ├── yields/
│   │   │   └── sync.go                          # GET /pools tiap 6 jam → Valkey (phase 1: supply APY only)
│   │   │
│   │   ├── hacks/
│   │   │   └── sync.go                          # banterous/defihack-db GitHub sync daily → defi_hacks
│   │   │
│   │   └── alert/
│   │       ├── scanner.go                       # TVLDropScanner, NewTop50Scanner, FeeSpikeScanner
│   │       └── sender.go                        # Discord + Telegram dual channel alert
│   │
│   ├── news/                                    # ── NEWS PIPELINE ──
│   │   ├── sources/
│   │   │   ├── cryptopanic.go                   # CryptoPanic REST (5m important, 15m rising) + WebSocket breaking
│   │   │   ├── rss.go                           # RSS poller: CoinDesk/CT/Decrypt (10m), minor (30m)
│   │   │   └── coingecko.go                     # CoinGecko /news (30m) — coin_id tag sudah ada
│   │   │
│   │   ├── ai/
│   │   │   ├── processor.go                     # Batch processor 5 menit: entity + category + summary + sentiment
│   │   │   ├── prompt.go                        # Prompt template — single call untuk semua task
│   │   │   └── importance.go                    # importance_score formula: credibility + panic + votes + sentiment + entity
│   │   │
│   │   ├── entities/
│   │   │   └── tagger.go                        # Entity tagging result handler: coin_id + llama_slug
│   │   │
│   │   ├── impact/
│   │   │   └── price.go                         # Price impact snapshot 1h + 24h setelah publish
│   │   │
│   │   ├── trending/
│   │   │   └── compute.go                       # TrendingCompute 1 jam: coin mentions, keyword freq, category spike
│   │   │
│   │   └── alert/
│   │       ├── detector.go                      # breaking_news, regulation, exploit_hack, watchlist
│   │       └── sender.go                        # Discord embed + Telegram message
│   │
│   ├── agent/                                   # ── AI AGENT LAYER ──
│   │   ├── router.go                            # AgentRouter — dispatch task ke LLM provider
│   │   ├── provider.go                          # LLMProvider interface
│   │   ├── task.go                              # AgentTask, AgentResult types
│   │   ├── ratelimit.go                         # per-provider rate limit tracker
│   │   ├── dedup.go                             # inflight dedup via Valkey SetNX
│   │   └── providers/
│   │       ├── irag.go                          # irag /v1/ai/text/* (priority 1-4, heartbeat $0)
│   │       ├── groq.go                          # Groq API direct (priority 5, RAG only)
│   │       ├── gemini.go                        # Gemini API direct (priority 6, RAG only)
│   │       └── openrouter.go                    # OpenRouter (priority 7, RAG last resort)
│   │
│   ├── pipeline/
│   │   ├── ingest.go                            # subscribe ch:signal:processed dari quant
│   │   └── tiering.go                           # hot/warm/cold storage routing
│   │
│   ├── storage/
│   │   ├── valkey.go                            # Valkey client wrapper
│   │   ├── timescale.go                         # TimescaleDB client + hypertable helpers
│   │   ├── postgres.go                          # PostgreSQL client
│   │   ├── discord.go                           # Discord cold storage logger + structured log sender
│   │   └── telegram.go                          # Telegram file_id cache + upload/forward logic
│   │
│   ├── storage_ext/                             # External storage layer
│   │   ├── gdrive.go                            # Google Drive API: cold export upload, backup status
│   │   ├── r2.go                                # Cloudflare R2: coin/chain/protocol logo sync
│   │   └── rclone.go                            # rclone wrapper: scheduled backup trigger
│   │
│   ├── scheduler/
│   │   └── main.go                              # Master scheduler — wire semua goroutine jobs
│   │
│   └── config/
│       └── config.go                            # env config loader
│
├── api/                                         # Go — public REST + WebSocket (api.dwizzy.my.id)
│   ├── cmd/
│   │   └── api/
│   │       └── main.go
│   │
│   ├── handler/
│   │   ├── market.go                            # /v1/market, /v1/market/{id} — CoinDetail + availability map
│   │   ├── market_ext.go                        # /v1/market/{id}/ohlcv, /orderbook, /tickers, /arbitrage
│   │   ├── defi.go                              # /v1/defi, /v1/defi/chain, /v1/defi/protocol
│   │   ├── defi_ext.go                          # /v1/defi/dex, /v1/defi/stable, /v1/defi/yields
│   │   ├── news.go                              # /v1/news, /v1/news/{category}, /v1/news/{id}
│   │   ├── news_ext.go                          # /v1/news/coin/{coin_id}, /v1/news/trending
│   │   ├── ai.go                                # /v1/ai/summarize, /v1/ai/analyze (PREMIUM)
│   │   └── health.go                            # /v1/health, /v1/providers (PUBLIC)
│   │
│   ├── middleware/
│   │   ├── auth.go                              # JWT validation (Discord OAuth + Web3 EVM)
│   │   ├── plan.go                              # free vs premium gate
│   │   └── ratelimit.go                         # per-client rate limit (100 RPM default)
│   │
│   ├── auth/
│   │   ├── discord.go                           # Discord OAuth2 flow
│   │   ├── web3.go                              # EVM wallet sign-in: nonce + verify signature
│   │   ├── jwt.go                               # JWT issue + verify (shared secret dengan CF Worker)
│   │   └── subscription.go                      # check on-chain subscription via go-ethereum (Polygon/Base)
│   │
│   ├── ws/
│   │   └── hub.go                               # WebSocket hub — broadcast real-time ke dwizzyDBSD
│   │
│   └── router.go                                # Fiber route registration — mount api + irag routes
│
├── irag/                                        # Go — Indonesian REST API Gateway
│   ├── cmd/
│   │   └── irag/
│   │       └── main.go                          # entry point irag (internal port 8081)
│   │
│   ├── provider/
│   │   ├── registry.go                          # provider registry + health check loop (30s)
│   │   ├── circuit.go                           # circuit breaker — open after 3 failures, reset 60s
│   │   ├── kanata.go                            # KanataAPI (highest reliability, primary)
│   │   ├── nexure.go                            # Nexure API + session management (AI primary)
│   │   ├── ryzumi.go                            # Ryzumi API (search primary, 115 endpoints)
│   │   ├── chocomilk.go                         # Chocomilk (novel, Tidal, Twitter)
│   │   └── ytdlp.go                             # YTDLP API (X-API-Key, playlist/subtitle)
│   │
│   ├── handler/
│   │   ├── ai.go                                # /v1/ai/text/*, /v1/ai/image/*, /v1/ai/process/*
│   │   ├── downloader.go                        # /v1/download/* (YouTube, TikTok, IG, Spotify, dll)
│   │   ├── search.go                            # /v1/search/* (Google, YouTube, Spotify, Lyrics, dll)
│   │   ├── bmkg.go                              # /v1/bmkg/* (PUBLIC)
│   │   ├── islamic.go                           # /v1/islamic/* (PUBLIC)
│   │   ├── anime.go                             # /v1/anime/*, /v1/manga/*
│   │   ├── film.go                              # /v1/film/*, /v1/drama/*, /v1/lk21
│   │   ├── tools.go                             # /v1/tools/* (translate, KBBI, QR, cekresi, dll)
│   │   ├── stalk.go                             # /v1/stalk/* (Instagram, GitHub, game profiles)
│   │   ├── game.go                              # /v1/game/growagarden/* (PUBLIC)
│   │   ├── news.go                              # /v1/news/*, /v1/media/* (PUBLIC)
│   │   ├── novel.go                             # /v1/novel/*
│   │   └── upload.go                            # /v1/upload/* (NexureCDN, KanataAPI CDN, RyzumiCDN)
│   │
│   ├── cache/
│   │   ├── l1.go                                # Valkey L1 cache (hot, TTL per kategori)
│   │   └── l2.go                                # TimescaleDB L2 cache (warm, persistent)
│   │
│   ├── normalizer/
│   │   └── envelope.go                          # unified response envelope: ok, code, data, error, meta, timestamp
│   │
│   ├── fallback/
│   │   └── chain.go                             # fallback chain resolver per endpoint category
│   │
│   └── router.go
│
├── quant/                                       # Python — indicator computation service
│   ├── main.py                                  # subscribe ch:ohlcv:raw:* → process → publish ch:signal:processed:*
│   ├── indicators.py                            # pandas-ta: RSI(14), MACD, BB, EMA
│   ├── scorer.py                                # quant_score composite (0-100)
│   ├── funding.py                               # funding rate parser + sentiment (futures)
│   ├── anomaly.py                               # volume spike, price deviation detector
│   ├── schema.py                                # SignalOutput dataclass
│   ├── config.py
│   └── requirements.txt                         # pandas-ta, valkey, numpy, python-dotenv
│
├── shared/
│   └── schema/
│       ├── signal.go                            # SignalOutput struct (Go)
│       ├── signal.py                            # SignalOutput dataclass (Python)
│       ├── coin.go                              # CoinDetail, AvailabilityMap structs
│       └── defi.go                              # ProtocolDetail, ChainDetail structs
│
├── deploy/
│   ├── docker-compose.yml
│   ├── docker-compose.prod.yml
│   ├── Dockerfile.engine
│   ├── Dockerfile.api
│   ├── Dockerfile.irag
│   └── Dockerfile.quant
│
├── cloudflare/                                  # Cloudflare Workers — fallback layer ($0)
│   ├── auth-worker/
│   │   └── index.ts                             # JWT issuer — Discord OAuth + Web3 (always online)
│   └── irag-fallback/
│       └── index.ts                             # hit upstream API langsung kalau homelab down
│
├── contracts/
│   └── SubscriptionManager.sol                  # EVM subscription — Polygon / Base
│
├── scripts/
│   ├── migrate.sh
│   ├── seed.sh
│   ├── healthcheck.sh
│   └── rclone-backup.sh                         # pg_dump + rclone copy ke gdrive
│
├── migrations/
│   │
│   ├── # ── CORE ──
│   ├── 001_init_extensions.sql                  # TimescaleDB, pgvector, PostGIS extensions
│   ├── 002_coins.sql                            # coins canonical registry
│   ├── 003_cold_coin_data.sql                   # ATH/ATL, description, links, categories
│   ├── 004_developer_data.sql                   # Github stats
│   ├── 005_coin_exchange_mappings.sql            # coin_id ↔ exchange symbol
│   ├── 006_coin_coverage.sql                    # tier A/B/C/D, availability per exchange
│   ├── 007_coin_data_completeness.sql           # field-level completeness score 0.0-1.0
│   ├── 008_unknown_symbols.sql                  # WS symbols queue untuk review
│   │
│   ├── # ── MARKET TIMESERIES ──
│   ├── 009_tickers_hypertable.sql               # realtime ticker per coin per exchange
│   ├── 010_ohlcv_hypertable.sql                 # OHLCV candles — retention per timeframe
│   ├── 011_exchange_spread_history.sql          # spread snapshot 30 hari, kompresi 1 hari
│   │
│   ├── # ── ARBITRAGE ──
│   ├── 012_arbitrage_signals.sql                # detected signals + generated columns
│   ├── 013_arbitrage_config.sql                 # threshold per coin: min_spread, min_depth, cooldown
│   │
│   ├── # ── DEFI ──
│   ├── 014_defi_protocols.sql                   # registry top 500 protocol + coin_id mapping
│   ├── 015_defi_protocol_tvl_latest.sql         # TVL snapshot terbaru + chain_tvls JSONB
│   ├── 016_defi_chain_tvl_latest.sql            # TVL snapshot per chain
│   ├── 017_defi_dex_latest.sql                  # volume snapshot per DEX
│   ├── 018_defi_protocol_coverage.sql           # tier classification + sync timestamps
│   ├── 019_defi_hacks.sql                       # hack history dari defihack-db
│   ├── 020_defi_stablecoin_backing.sql          # backing composition per stablecoin
│   ├── 021_defi_alert_config.sql                # alert threshold per type + target
│   │
│   ├── # ── DEFI TIMESERIES ──
│   ├── 022_defi_protocol_tvl_history.sql        # full history TVL top 50 protocol
│   ├── 023_defi_dex_volume_history.sql          # full history volume top 30 DEX
│   ├── 024_defi_chain_tvl_history.sql           # full history TVL top 15 chain
│   ├── 025_defi_stable_mcap_history.sql         # 2 tahun mcap semua stablecoin
│   ├── 026_defi_fees_history.sql                # 1 tahun fees + revenue top 100 protocol
│   │
│   ├── # ── NEWS ──
│   ├── 027_news_articles.sql                    # artikel mentah semua sumber, retensi 90 hari
│   ├── 028_news_ai_metadata.sql                 # summary, sentiment, importance. one-to-one
│   ├── 029_news_entities.sql                    # entity tagging: coin_id + llama_slug per artikel
│   ├── 030_news_sources.sql                     # registry sumber + credibility + polling config
│   ├── 031_news_price_impact.sql                # price snapshot 1h/4h/24h per coin per artikel
│   ├── 032_news_trending_cache.sql              # pre-computed trending topics, refresh 1 jam
│   ├── 033_news_alert_config.sql                # alert config per type + target
│   │
│   ├── # ── NEWS TIMESERIES ──
│   ├── 034_news_price_impact_history.sql        # permanent time-series price impact
│   │
│   ├── # ── AI SIGNAL ──
│   ├── 035_signals_table.sql                    # quant signal history
│   ├── 036_ai_results_table.sql                 # LLM summary + decision output
│   │
│   ├── # ── STORAGE ──
│   ├── 037_telegram_file_cache.sql              # file_id cache: chart, CSV, backup
│   └── 038_irag_request_log.sql                 # irag request log + L2 cache
│
├── docs/
│   ├── architecture.md                          # full system diagram
│   ├── api-reference.md                         # dwizzyBRAIN endpoint docs
│   ├── irag-reference.md                        # irag 180+ endpoint docs
│   ├── market-architecture.md                   # market data pipeline design
│   ├── defi-plan.md                             # DeFi subsystem design
│   ├── news-plan.md                             # news pipeline design
│   ├── storage-plan.md                          # storage tier strategy
│   └── subscription.md                          # Web3 payment + contract
│
├── go.work
├── go.work.sum
├── .env.example
├── .gitignore
└── README.md
```

### 9.1 Core File & Function List

The following tables explicitly map core files to their functional purposes across the primary sub-services.

#### Engine (Core Data Ingestion & Processing)
| Module | File | Core Functions / Purpose |
|---|---|---|
| `market/ws` | `binance.go`, `bybit.go`, `okx.go` | Native WebSocket clients for hot-path tickers and orderbooks |
| `market/adapters` | `ccxt_watch.go` | Optional CCXT websocket/pro transport adapters for selected non-critical streams |
| `market/ohlcv` | `service.go` | `GetOHLCV`, `BackfillOHLCV`, `IncrementalSync`, `PollShortTF`, `CleanupExpired` |
| `market/orderbook` | `service.go` | `SubscribeNativeWS`, `GetOrderBook`, `PollMinorExchanges` |
| `market/ticker` | `aggregator.go`, `spread.go` | `BestBid`, `BestAsk`, `MaxSpreadPct`, `VolumePct`, and spread snapshotting |
| `market/arbitrage` | `engine.go`, `alert.go` | 5-second `ArbitrageEngine` scanner, 8-step detection, and Discord alerting |
| `market/mapping` | `resolver.go`, `builder.go` | `SymbolResolver` (coin_id ↔ exchange_symbol), auto-building active mappings |
| `defi/protocols` | `tvl.go`, `history.go`, `fees.go`| TVL sync (hourly), history backfill (TimescaleDB), and fee/revenue syncing |
| `defi/stables` | `sync.go`, `depeg.go`, `backing.go`| `DepegScanner` (abs(price-1) > 0.01 threshold), backing composition sync |
| `news/ai` | `processor.go`, `importance.go` | Batch processing for entity extraction, summary, sentiment, `importance_score` |
| `news/trending` | `compute.go` | Hourly `TrendingCompute` for coin mentions and category spikes |
| `agent/providers` | `irag.go`, `groq.go`, `gemini.go`| LLM Provider interfaces implementing Priority 1-7 routing mechanism |
| `storage` | `valkey.go`, `timescale.go`, `discord.go`| Client wrappers, hypertable helpers, and structured cold storage logging |

#### API (Public REST & WS Gateway)
| Module | File | Core Functions / Purpose |
|---|---|---|
| `api/handler` | `market.go`, `market_ext.go` | Handlers for `/v1/market/*` (CoinDetail, OHLCV, orderbooks, tickers, arbitrage) |
| `api/handler` | `defi.go`, `defi_ext.go` | Handlers for `/v1/defi/*` (Protocols, specific chains, DEX, stables, yields) |
| `api/handler` | `news.go`, `ai.go` | Handlers for `/v1/news/*` (feeds, trending) and `/v1/ai/*` (summarize, analyze) |
| `api/middleware`| `auth.go`, `plan.go`, `ratelimit.go`| JWT validation (OAuth+Web3), Free vs Premium gating, token-bucket limits |
| `api/auth` | `discord.go`, `web3.go`, `jwt.go` | Discord OAuth2 flows, EVM wallet signature `personal_sign` verification |
| `api/ws` | `hub.go` | Multiplexed WebSocket state broadcaster for real-time dashboards |

#### IRAG (Indonesian REST API Gateway)
| Module | File | Core Functions / Purpose |
|---|---|---|
| `irag/provider` | `registry.go`, `circuit.go` | Upstream registry, health-checking, circuit breakers (open after 3 failures) |
| `irag/handler` | `ai.go`, `downloader.go`, etc. | Proxy endpoints categorizing 180+ tools (Search, AI, BMKG, Stalk, Games) |
| `irag/cache` | `l1.go`, `l2.go` | Valkey (1m TTL) and TimescaleDB (2h TTL) caching interceptors |
| `irag/fallback` | `chain.go` | Fallback routing logic (if Priority 1 fails ➔ Try Priority 2) |
| `irag/normalizer`| `envelope.go` | Wraps unstandardized upstream JSON into reliable `{ok, code, data, meta}` formats |

#### Quant (Python Analytics)
| Module | File | Core Functions / Purpose |
|---|---|---|
| `quant` | `main.py` | Daemon subscribing to Valkey `ch:ohlcv:raw:*` and publishing to `ch:signal:*` |
| `quant` | `indicators.py` | Computes RSI(14), MACD, BB, and EMA via the `pandas-ta` library |
| `quant` | `scorer.py` | Calculates a composite `quant_score` (0-100 scale) for signals |
| `quant` | `funding.py` | Parses funding rates from futures exchanges for sentiment analysis |
| `quant` | `anomaly.py` | Volume spike and rapid price deviation detector |
