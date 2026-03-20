# dwizzyBRAIN — File Structure v4

> Incorporates: Unified Market Architecture, News Plan, DeFi Plan, Storage Strategy Plan

```
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
│   ├── pyproject.toml                           # package metadata + dependencies
│   ├── src/
│   │   └── quant/
│   │       ├── main.py                          # subscribe ch:ohlcv:raw:* → process → publish ch:signal:processed:*
│   │       ├── indicators.py                    # RSI, MACD, BB, EMA computation
│   │       ├── scorer.py                        # quant_score composite (0-100)
│   │       ├── models.py                        # OHLCVPayload, QuantSignal
│   │       └── __init__.py
│   └── tests/                                   # pytest suite for quant worker
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

---

## Service Map

| Service | Language | Port | Public |
|---------|----------|------|--------|
| engine | Go | internal | ❌ |
| api | Go | 8080 | ✅ api.dwizzy.my.id |
| irag | Go | 8081 | ✅ via api/ proxy |
| quant | Python | internal | ❌ |
| cf auth-worker | TypeScript | edge | ✅ auth.dwizzy.my.id |
| cf irag-fallback | TypeScript | edge | ✅ otomatis saat homelab down |

---

## Public Route Map (api.dwizzy.my.id)

| Path | Handler | Auth | Plan |
|------|---------|------|------|
| `GET /v1/market` | market.go | ❌ | PUBLIC |
| `GET /v1/market/{id}` | market.go | ❌ | PUBLIC |
| `GET /v1/market/{id}/ohlcv` | market_ext.go | ✅ | FREE |
| `GET /v1/market/{id}/orderbook` | market_ext.go | ✅ | FREE |
| `GET /v1/market/{id}/tickers` | market_ext.go | ✅ | FREE |
| `GET /v1/market/{id}/arbitrage` | market_ext.go | ✅ | PREMIUM |
| `GET /v1/market/arbitrage/active` | market_ext.go | ✅ | PREMIUM |
| `GET /v1/defi` | defi.go | ❌ | PUBLIC |
| `GET /v1/defi/chain` | defi.go | ❌ | PUBLIC |
| `GET /v1/defi/protocol` | defi.go | ❌ | PUBLIC |
| `GET /v1/defi/protocol/{slug}` | defi.go | ❌ | PUBLIC |
| `GET /v1/defi/dex` | defi_ext.go | ❌ | PUBLIC |
| `GET /v1/defi/stable` | defi_ext.go | ❌ | PUBLIC |
| `GET /v1/defi/yields` | defi_ext.go | ✅ | FREE |
| `GET /v1/news` | news.go | ❌ | PUBLIC |
| `GET /v1/news/{category}` | news.go | ❌ | PUBLIC |
| `GET /v1/news/{id}` | news.go | ❌ | PUBLIC |
| `GET /v1/news/coin/{coin_id}` | news_ext.go | ❌ | PUBLIC |
| `GET /v1/news/trending` | news_ext.go | ❌ | PUBLIC |
| `POST /v1/ai/summarize` | ai.go | ✅ | PREMIUM |
| `POST /v1/ai/analyze` | ai.go | ✅ | PREMIUM |
| `WS /v1/ws/market` | ws/hub.go | ✅ | PREMIUM |
| `GET /v1/bmkg/*` | irag/bmkg.go | ❌ | PUBLIC |
| `GET /v1/islamic/*` | irag/islamic.go | ❌ | PUBLIC |
| `GET /v1/game/*` | irag/game.go | ❌ | PUBLIC |
| `GET /v1/download/*` | irag/downloader.go | ✅ | FREE |
| `GET /v1/search/*` | irag/search.go | ✅ | FREE |
| `GET /v1/tools/*` | irag/tools.go | ✅ | FREE |
| `GET /v1/stalk/*` | irag/stalk.go | ✅ | FREE |
| `GET /v1/ai/text/*` | irag/ai.go | ✅ | FREE |
| `GET /v1/ai/image/*` | irag/ai.go | ✅ | FREE |
| `GET /health` | health.go | ❌ | PUBLIC |

---

## Valkey Key Schema

### Market — Hot Tier
| Key | Type | TTL | Source |
|-----|------|-----|--------|
| `price:{symbol}:{market}` | STRING | 10s | Binance/Bybit/OKX WS |
| `ohlcv:{symbol}:{market}:{tf}` | LIST | no TTL | LPUSH+LTRIM 200 candles |
| `hot_prices:{coin_id}` | STRING | 30s | WS ticker |
| `warm_prices:{coin_id}` | STRING | 10m | REST poll |
| `orderbook:{exchange}:{symbol}` | STRING | 5s/30s | WS/CCXT REST |
| `ticker:{exchange}:{symbol}` | STRING | 10s | WS/CCXT REST |
| `ticker:aggregated:{coin_id}` | STRING | 10s | TickerAggregator |
| `symmap:{exchange}:{id}` | STRING | 1h | SymbolResolver |
| `arb:last_alert:{coin}:{buy}:{sell}` | STRING | cooldown | ArbitrageEngine |
| `ohlcv:backfill:{coin_id}:{tf}` | STRING | permanent | OHLCVService flag |

### Market — Signal
| Key | Type | TTL | Source |
|-----|------|-----|--------|
| `signal:{symbol}:{tf}` | STRING | 1h | quant |
| `ai:result:{symbol}` | STRING | 1h | AgentRouter |
| `inflight:{type}:{sourceID}` | STRING | 10m | dedup lock |

### DeFi
| Key | Type | TTL | Source |
|-----|------|-----|--------|
| `defi:overview:snapshot` | STRING | 15m | scheduler |
| `defi:protocols:list` | STRING | 1h | scheduler |
| `defi:protocol:{slug}:detail` | STRING | 1h | on-demand |
| `defi:protocol:{slug}:fees` | STRING | 1h | on-demand |
| `defi:chain:{chain}:detail` | STRING | 1h | on-demand |
| `defi:dex:{slug}:detail` | STRING | 15m | on-demand |
| `defi:stable:{asset}:detail` | STRING | 1h | on-demand |
| `defi:stable:{asset}:depeg` | STRING | 1h | DepegScanner |
| `defi:pools:list` | STRING | 6h | scheduler |
| `defi:whale:{coin_id}` | STRING | 6h | Etherscan API |
| `alert:defi:cooldown:{type}:{id}` | STRING | cooldown | alert engine |

### News
| Key | Type | TTL | Source |
|-----|------|-----|--------|
| `news:feed:all` | STRING | 5m | scheduler |
| `news:feed:{category}` | STRING | 5m | scheduler |
| `news:coin:{coin_id}` | STRING | 5m | scheduler |
| `news:protocol:{llama_slug}` | STRING | 5m | scheduler |
| `news:trending:24h` | STRING | 1h | TrendingCompute |
| `news:article:{id}:ai` | STRING | 24h | AI processor |
| `news:pending_ai` | LIST | 5m | ingest queue |
| `news:pending_price_impact` | LIST | 2h | impact queue |
| `alert:news:cooldown:{type}:{id}` | STRING | cooldown | alert engine |

### Storage
| Key | Type | TTL | Source |
|-----|------|-----|--------|
| `telegram:file:{file_key}` | STRING | no TTL | telegram_file_cache mirror |

---

## Valkey Channel Map

| Channel | Producer | Consumer |
|---------|----------|----------|
| `ch:ohlcv:raw:{symbol}:{market}:{tf}` | engine/market | quant |
| `ch:signal:processed:{symbol}` | quant | engine/agent |
| `ch:ai:result:{symbol}` | engine/agent | api/ws, dwizzyBOT |
| `ch:news:raw` | engine/news | quant |
| `ch:defi:alert` | engine/defi/alert | api/ws, dwizzyBOT |
| `ch:news:alert` | engine/news/alert | api/ws, dwizzyBOT |
| `ch:arb:signal` | engine/market/arbitrage | api/ws, dwizzyBOT |

---

## AgentRouter Provider Priority

| Priority | Provider | Use Case | Cost |
|----------|----------|----------|------|
| 1 | irag → `/v1/ai/text/groq` | heartbeat, summarize | $0 |
| 2 | irag → `/v1/ai/text/gemini` | heartbeat, summarize | $0 |
| 3 | irag → `/v1/ai/text/deepseek` | heartbeat, summarize | $0 |
| 4 | irag → `/v1/ai/text/qwen` | heartbeat, summarize | $0 |
| 5 | Groq API direct | RAG, heavy tasks | low |
| 6 | Gemini API direct | RAG, heavy tasks | low |
| 7 | OpenRouter | RAG, last resort | variable |

---

## Storage Tier Strategy

| Data | Primary | Backup | TTL/Retention |
|------|---------|--------|---------------|
| PostgreSQL aktif | Mini PC local | Google Drive daily gzip | operational |
| TimescaleDB OHLCV | Mini PC local | Google Drive weekly dump | per timeframe |
| Valkey snapshot | Mini PC local | Google Drive mingguan | cache |
| Coin/chain logos | Cloudflare R2 | MEGA mirror | permanent CDN |
| Generated charts | Telegram file_id | file_id di PostgreSQL | permanent |
| CSV/Parquet export | Google Drive | Storj cold archive | permanent |
| System logs | Discord channels | PostgreSQL duplicate | 7 hari local |
| Alert history | Discord channels | PostgreSQL | persistent |

---

## Build Phases

### Phase 1 — Engine Core (Week 1)
```
Day 1 — engine/market/ws/ + ccxt.go        Binance WS + REST, CCXT minor exchanges
Day 2 — engine/market/publisher.go          Valkey writer + channel publish
Day 3 — quant/                              RSI, MACD, BB, funding rate, scorer
Day 4 — migrations/001-013                  Core + market + arbitrage schema
Day 5 — engine/agent/ + irag provider pool  AgentRouter priority 1-7
```

### Phase 2 — Market Full (Week 2)
```
engine/market/mapping/       SymbolResolver, MappingBuilder, GapDetector
engine/market/ohlcv/         OHLCVService, BackfillOHLCV top 100
engine/market/orderbook/     OrderBookService native WS + CCXT minor
engine/market/ticker/        TickerAggregator, RecordSpread
engine/market/arbitrage/     ArbitrageEngine 5s scan, Discord alert
engine/market/coingecko/     Cold load 1000 coins 24h
api/handler/market*.go       /v1/market + /v1/market/{id}/* endpoints
```

### Phase 3 — DeFi (Week 3)
```
engine/defi/                 Protocol registry, TVL sync, chain, DEX, stables
migrations/014-026           DeFi schema
api/handler/defi*.go         /v1/defi/* endpoints
```

### Phase 4 — News (Week 4)
```
engine/news/                 CryptoPanic WS+REST, RSS, AI batch processor
migrations/027-034           News schema
api/handler/news*.go         /v1/news/* endpoints
```

### Phase 5 — Auth + Storage + Frontend (Week 5-6)
```
api/auth/                    Discord OAuth + Web3 EVM
contracts/                   SubscriptionManager.sol deploy
engine/storage_ext/          Google Drive backup, Telegram file_id, R2
cloudflare/                  Auth worker + irag fallback
dwizzyDBSD                   Next.js dashboard (separate repo)
```

---

## Environment Variables (.env.example)

```env
# Exchange
BINANCE_API_KEY=
BINANCE_API_SECRET=
BYBIT_API_KEY=
BYBIT_API_SECRET=
OKX_API_KEY=
OKX_API_SECRET=

# External APIs
COINGECKO_API_KEY=
CRYPTOPANIC_API_KEY=
ETHERSCAN_API_KEY=

# irag upstream
YTDLP_API_KEY=
IRAG_INTERNAL_URL=http://irag:8081

# LLM Providers (direct — RAG only)
GROQ_API_KEY=
GEMINI_API_KEY=
OPENROUTER_API_KEY=

# Storage
TIMESCALE_URL=postgres://user:pass@localhost:5432/dwizzyos
POSTGRES_URL=postgres://user:pass@localhost:5432/dwizzyos
VALKEY_URL=redis://localhost:6379

# External Storage
GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON=
CLOUDFLARE_R2_ACCOUNT_ID=
CLOUDFLARE_R2_ACCESS_KEY=
CLOUDFLARE_R2_SECRET_KEY=
CLOUDFLARE_R2_BUCKET=dwizzyo-assets

# Discord
DISCORD_CLIENT_ID=
DISCORD_CLIENT_SECRET=
DISCORD_REDIRECT_URI=
DISCORD_WEBHOOK_LOG=
DISCORD_WEBHOOK_ALERT=
DISCORD_WEBHOOK_MARKET=
DISCORD_WEBHOOK_DEFI=
DISCORD_WEBHOOK_NEWS=
DISCORD_BOT_TOKEN=
DISCORD_FILES_CHANNEL_ID=

# Telegram
TELEGRAM_BOT_TOKEN=
TELEGRAM_ALERTS_CHANNEL=
TELEGRAM_FILES_CHANNEL=
TELEGRAM_LOGS_SUPERGROUP=

# Auth + Web3
JWT_SECRET=
RPC_URL=https://polygon-rpc.com
CONTRACT_ADDRESS=

# Ports
API_PORT=8080
IRAG_PORT=8081
```
