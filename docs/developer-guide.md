# dwizzyBRAIN Developer Guide

Updated: March 19, 2026

This guide is the practical entry point for developers working on `dwizzyBRAIN`. It focuses on how the system is wired today, how to run it locally, and how to verify changes without reading every PRD first.

## 1. What dwizzyBRAIN Is

`dwizzyBRAIN` is the backend control plane for the `dwizzyOS` stack.

It owns:
- market ingestion and aggregation
- DeFi reads and sync jobs
- news ingest, AI tagging, and price-impact tracking
- authentication, premium gating, and entitlement resolution
- external storage bridges
- quant signal generation and pattern search

It exposes:
- REST APIs under `/v1/*`
- OpenAPI at `/openapi.json`
- browser docs at `/docs`

## 2. Service Map

### API

Path: [`api/`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/api)

Responsibilities:
- read APIs for market, DeFi, news, quant
- Discord OAuth
- Web3 auth
- entitlement and premium gating
- OpenAPI and docs serving

Run:
```bash
go run ./api/cmd/api
```

### Engine

Path: [`engine/`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/engine)

Responsibilities:
- market ingestion
- CoinGecko cold load
- mapping sync
- OHLCV sync
- ticker aggregation
- arbitrage scan
- DeFi sync
- news ingest, AI, archive, price-impact
- storage bridges

Run:
```bash
go run ./engine/cmd/engine
```

### Quant

Path: [`quant/`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/quant)

Responsibilities:
- realtime quant signals from `ch:ohlcv:raw:*`
- indicator computation
- macro event labeling
- pattern embeddings
- backfill exports
- outcome model training

Run healthcheck:
```bash
cd quant
.venv/bin/python -m quant.main --healthcheck
```

Run worker:
```bash
cd quant
.venv/bin/python -m quant.main
```

### Cloudflare Workers

Path: [`cloudflare/`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/cloudflare)

Workers:
- `auth-worker` at `auth.dwizzy.my.id`
- `irag-fallback` at `api2.dwizzy.my.id`

### Contracts

Path: [`contracts/`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/contracts)

Responsibilities:
- `SubscriptionManager.sol`
- multi-chain deploy tooling

## 3. Runtime Topology

Current live topology:
- `dwizzy.my.id` -> frontend
- `api.dwizzy.my.id` -> Go API
- `auth.dwizzy.my.id` -> Cloudflare auth worker
- `api2.dwizzy.my.id` -> Cloudflare IRAG fallback worker

Backend data flow:
1. Engine ingests raw market/news/DeFi data.
2. Engine writes normalized rows to Postgres / Timescale / Valkey.
3. Quant subscribes to OHLCV pub/sub and emits processed signals.
4. API reads the normalized data and exposes it to clients.
5. Cloudflare workers provide edge auth / fallback resilience.

## 4. Local Prerequisites

You need:
- Go toolchain
- Python 3.12+
- Node.js for Cloudflare workers and contracts
- Postgres
- TimescaleDB
- Valkey / Redis
- optional: Docker / Docker Compose

The repo already contains:
- `quant/.venv`
- `quant/Dockerfile.quant`
- `quant/docker-compose.yml`

## 5. Secrets and Configuration

The code supports direct env values or file-mounted secrets using `NAME` or `NAME_FILE`.

Important env vars:
- `POSTGRES_URL` or `POSTGRES_URL_FILE`
- `TIMESCALE_URL` or `TIMESCALE_URL_FILE`
- `VALKEY_URL` or `VALKEY_URL_FILE`
- `JWT_SECRET` or `JWT_SECRET_FILE`
- `DISCORD_CLIENT_ID`
- `DISCORD_CLIENT_SECRET` or `DISCORD_CLIENT_SECRET_FILE`
- `DISCORD_REDIRECT_URI`
- `DISCORD_BOT_TOKEN` or `DISCORD_BOT_TOKEN_FILE`
- `COINGECKO_API_KEY` or `COINGECKO_API_KEY_FILE`
- `TELEGRAM_BOT_TOKEN` or `TELEGRAM_BOT_TOKEN_FILE`
- `SUBSCRIPTION_NETWORKS`
- `SUBSCRIPTION_DEPLOYER_PRIVATE_KEY` or `SUBSCRIPTION_DEPLOYER_PRIVATE_KEY_FILE`

Quant-specific env vars:
- `QUANT_WINDOW_SIZE`
- `QUANT_SIGNAL_TTL_SECONDS`
- `QUANT_PUBLISH_SIGNALS`
- `QUANT_CACHE_PREFIX`
- `QUANT_SIGNAL_CHANNEL_PREFIX`

Engine feature flags:
- `MAPPING_SYNC_ENABLED`
- `BINANCE_WS_ENABLED`
- `TICKER_POLL_TARGETS`
- `OHLCV_SYNC_TARGETS`
- `COINGECKO_COLDLOAD_ENABLED`
- `DEFI_TVL_ENABLED`
- `DEFI_YIELD_ENABLED`
- `DEFI_STABLECOIN_ENABLED`
- `NEWS_ENABLED`
- `NEWS_AI_ENABLED`
- `NEWS_IMPACT_ENABLED`
- `NEWS_ARCHIVE_ENABLED`
- `STORAGE_EXT_ENABLED`
- `ARBITRAGE_ENABLED`

## 6. Common Developer Commands

### Go backend

Run tests:
```bash
go test ./... -count=1
```

Run API:
```bash
go run ./api/cmd/api
```

Run engine:
```bash
go run ./engine/cmd/engine
```

### Quant

Run tests:
```bash
cd quant
.venv/bin/python -m pytest -q tests
```

Healthcheck:
```bash
cd quant
.venv/bin/python -m quant.main --healthcheck
```

Cold backfill:
```bash
cd quant
.venv/bin/python -m quant.backfill.compute_bulk --coin-id bitcoin --exchange binance --timeframe 1m --limit 300 --json
```

Pattern embeddings:
```bash
cd quant
.venv/bin/python -m quant.backfill.build_vectors --coin-id bitcoin --exchange binance --timeframe 1m --limit 300 --json
```

Outcome model:
```bash
cd quant
.venv/bin/python -m quant.backfill.train_outcome_model --coin-id spy --exchange yfinance --timeframe 1d --limit 300 --json
```

### Cloudflare Workers

Auth worker:
```bash
cd cloudflare/auth-worker
npm test
npm run typecheck
wrangler deploy
```

IRAG fallback:
```bash
cd cloudflare/irag-fallback
npm test
npm run typecheck
wrangler deploy
```

IRAG gateway:
```bash
go run ./irag/cmd/irag
```

Optional env overrides:
- `IRAG_KANATA_URL`
- `IRAG_NEXURE_URL`
- `IRAG_RYZUMI_URL`
- `IRAG_CHOCOMILK_URL`
- `IRAG_YTDLP_URL`

### Contracts

Compile:
```bash
cd contracts
npm run compile
```

Deploy all supported networks:
```bash
cd contracts
node scripts/deploy-all.mjs
```

## 7. Verification Checklist

When you change backend code, use the smallest relevant verification loop first, then run the repo-wide check.

Recommended order:
1. unit tests for the touched package
2. compile/typecheck
3. live smoke against the relevant service
4. repo-wide test run

Examples:
- Go package tests: `go test ./engine/market/... -v`
- Python quant: `quant/.venv/bin/python -m pytest -q tests`
- API: `go test ./api/... -count=1`
- contracts: `npm run compile`

## 8. Data Model Notes

### Market

Key live tables:
- `coins`
- `coin_exchange_mappings`
- `unknown_symbols`
- `ohlcv`
- `candle_indicators`
- `candle_features`
- `candle_embeddings`
- `macro_events`
- `candle_event_labels`
- `arbitrage_signals`
- `exchange_spread_history`
- `coin_coverage`

### DeFi

Key live tables:
- `defi_protocols`
- `defi_protocol_tvl_latest`
- `defi_protocol_tvl_history`
- `defi_chain_tvl_latest`
- `defi_chain_tvl_history`
- `defi_yield_latest`
- `defi_yield_history`
- `defi_stablecoin_backing`
- `defi_stable_mcap_history`

### News

Key live tables:
- `news_sources`
- `news_articles`
- `news_ai_metadata`
- `news_entities`
- `news_price_impact`
- `news_price_impact_history`
- `news_article_markdown_exports`

### Auth

Key live tables:
- `users`
- `user_identities`
- `user_sessions`
- `refresh_tokens`
- `auth_nonces`
- `telegram_file_cache`

## 9. Practical Data Flows

### Market

- Native Binance WS or CCXT poll -> resolver -> publisher -> Valkey
- shared ticker aggregator -> spread recorder / arbitrage
- OHLCV sync -> Timescale -> quant pub/sub

### Quant

- `ch:ohlcv:raw:*` -> realtime indicators -> `signal:{symbol}:{timeframe}`
- historical OHLCV -> indicators -> features -> embeddings
- embeddings -> pattern search API
- FRED + ForexFactory -> event labels -> macro context
- Yahoo Finance -> stock / forex / commodity backfill

### News

- RSS / Telegram -> raw article storage
- AI pass -> metadata + entities
- price-impact pass -> impact rows
- archive pass -> Drive-backed markdown export

### Auth

- Discord OAuth -> session + refresh token
- Web3 nonce/signature -> wallet linkage
- entitlement resolver -> premium gating
- Cloudflare worker -> edge auth fallback

## 10. Deployment Notes

### Core backend

The backend expects live services for:
- Postgres
- TimescaleDB
- Valkey

### Edge

Workers:
- `auth-worker` is deployed and bound to `auth.dwizzy.my.id`
- `irag-fallback` is deployed and bound to `api2.dwizzy.my.id`

### Contracts

`SubscriptionManager.sol` is deployed on:
- Base
- BSC
- Kaia
- Arbitrum

Sonic, Abstract, and HyperEVM are optional future targets if funded.

## 11. Troubleshooting

- If the API starts but auth is disabled, check `DISCORD_CLIENT_ID`, `DISCORD_CLIENT_SECRET`, `DISCORD_REDIRECT_URI`, and `JWT_SECRET`.
- If quant starts but does not emit signals, check `VALKEY_URL`, `QUANT_WINDOW_SIZE`, and whether OHLCV pub/sub is active.
- If `ohlcv` writes fail, verify the live schema still has `symbols`, `symbol_id`, and `interval`.
- If Yahoo Finance backfill fails, make sure `yfinance` is installed in `quant/.venv`.
- If Cloudflare deploy fails, verify Wrangler login and the route binding in `wrangler.jsonc`.

## 12. Recommended Reading Order

If you are new to the repo, read in this order:
1. [`dwizzyBRAIN-implementation-status.md`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/docs/dwizzyBRAIN-implementation-status.md)
2. [`PRD.md`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/PRD.md)
3. [`PRD-QUANT.md`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/docs/PRD-QUANT.md)
4. [`PRD-QUANT-IMPLEMENTATION-PLAN.md`](/home/dwizzy/workspace/projects/dwizzyOS/dwizzyBRAIN/docs/PRD-QUANT-IMPLEMENTATION-PLAN.md)

## 13. Current Scope

Current backend status:
- Core backend is complete and live.
- `quant` is complete for the current backend scope.
- Remaining items are optional expansions only.
