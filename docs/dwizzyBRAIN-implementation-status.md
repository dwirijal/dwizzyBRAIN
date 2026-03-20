# dwizzyBRAIN Implementation Status

Updated: March 19, 2026

This document records what has been implemented and verified in `dwizzyBRAIN` so far. It is a status snapshot, not a design proposal.

## Completed

### Market pipeline

- Native Binance WebSocket ingestion is live.
- CCXT REST fallback polling is live for configured exchanges.
- Symbol resolution is in place via `coin_exchange_mappings` and `unknown_symbols`.
- Ticker ingestion publishes resolved prices into Valkey and the shared ticker aggregator.
- OHLCV backfill and incremental sync write to TimescaleDB and publish to Valkey.
- Ticker aggregation and spread recording are live.
- Arbitrage detection is wired for the shared runtime aggregator.
- CoinGecko cold load is wired and verified.

### Quant pipeline

- `quant/` has a live real-time loop subscribing to `ch:ohlcv:raw:*`.
- The core quant signal pipeline computes RSI, MACD, Bollinger Bands, EMA, ATR, volume ratio, funding sentiment, anomaly flags, and a composite `quant_score`.
- Quant signals are cached into Valkey hot storage under `signal:{symbol}:{timeframe}` with TTL support and are published to `ch:signal:processed:{symbol}`.
- The quant package includes config loading and persistence helpers for secret-safe runtime operation.
- `quant` now has a container build artifact in `quant/Dockerfile.quant`, a compose example in `quant/docker-compose.yml`, and a `--healthcheck` CLI mode for runtime validation.
- The quant data layer is now live in Postgres via `candle_indicators`, `candle_features`, and `candle_embeddings`.
- Quant hot-signal persistence has been live-smoked against Valkey and the latest signal key is readable without replaying the pub/sub stream.
- Quant indicator coverage now includes the full Section 5 catalog: expanded trend, momentum, volatility, volume, candlestick pattern, Supertrend, Ichimoku, pivot/Fib, and lightweight SMC feature columns.
- Quant backfill helpers are live for historical OHLCV fetch, Yahoo Finance multi-asset backfill, bulk indicator computation, Parquet cold archive export via DuckDB, local fingerprint embeddings, and outcome model training on fingerprint + labels.
- Quant macro event labeling is live for the FRED + ForexFactory slice: macro series ingestion, candle event labels, and `macro_environment` generation are persisted in Postgres.
- Quant pattern engine is live for fingerprint generation, pgvector bulk load, and similarity search with confidence gating; the live embedding smoke has been verified against `candle_embeddings`.

### DeFi pipeline

- DeFi protocol registry API is live at `/v1/defi`.
- DeFi TVL sync is live:
  - latest protocol snapshots
  - latest chain snapshots
  - protocol history backfill
  - chain history backfill
- Yield tracking is live against `yields.llama.fi`.
- Stablecoin tracking and depeg scanning are live against `stablecoins.llama.fi`.

### News pipeline

- RSS news ingestion is live for the seeded sources in `news_sources`.
- BeInCrypto Indonesia is now ingested from the official Telegram news channel because the site RSS endpoint is Cloudflare-blocked.
- Raw articles are stored in `news_articles`.
- Heuristic AI processing is live and writes `news_ai_metadata` and `news_entities`.
- Price-impact tracking is live and writes `news_price_impact`; the history table is in place for completed windows.
- News article archives are live in an Obsidian-style layout: pending articles are rendered to `content.md` with frontmatter, uploaded to Google Drive, and persisted in `news_article_markdown_exports` with `title` and `drive_url`.
- The news read API is live at `/v1/news`, `/v1/news/{value}`, `/v1/news/coin/{coin_id}`, and `/v1/news/trending`.
- The engine can run a one-shot RSS sync from `engine/news`.
- The engine can run a one-shot AI batch against unprocessed RSS articles.
- Live RSS feeds from CoinDesk, CoinTelegraph, Decrypt, and The Block were verified.

### API surface

- `/v1/market`
- `/v1/market/{id}`
- `/v1/market/{id}/ohlcv`
- `/v1/market/{id}/tickers`
- `/v1/market/{id}/arbitrage`
- `/v1/defi`
- `/v1/defi/protocols`
- `/v1/defi/protocols/{slug}`
- `/v1/defi/chains`
- `/v1/defi/dexes`
- OpenAPI spec is published at `/openapi.json`.
- Browser docs UI is available at `/docs`.
- API root exposes discoverability links at `/`.

### Auth

- Discord OAuth start/callback, session issuance, refresh rotation, logout, and `me` are implemented in the API auth layer.
- Web3 wallet auth is implemented with nonce issuance and `personal_sign` verification over EVM addresses.
- Premium gating middleware is in place, and entitlement resolution can consult `SubscriptionManager.sol` when RPC and contract settings are provided.
- The repo now includes multi-chain `SubscriptionManager.sol` scaffolding plus deploy tooling for Base, BSC, and HyperEVM; live contract addresses still need to be deployed.
- Multi-chain subscription contracts are deployed on Base, BSC, Kaia, and Arbitrum; Sonic and Abstract were skipped because the deploy wallet did not have enough native balance.
- Secret-bearing runtime config now supports direct env values or `*_FILE` secrets for auth, Coingecko, Telegram, and contract deploy tooling.
- A Cloudflare `auth-worker` edge layer is scaffolded under `cloudflare/auth-worker` with JWT verification, auth proxying, CORS controls, and local Node tests.
- Auth tables for users, identities, sessions, refresh tokens, and nonces are present in the live schema.
- The auth flow is unit-tested with mocked Discord and live Web3 integration tests against Postgres.

### External storage bridges

- Telegram file_id caching is implemented against the live `telegram_file_cache` schema and mirrored into Valkey.
- Google Drive backup and Cloudflare R2 sync wrappers are implemented through rclone-backed bridge services.
- Telegram upload helpers support document and photo uploads for generated assets and exports.

## Real Data Verification

The following live checks were run successfully:

- Market runtime:
  - Binance WS stream
  - CCXT poll targets
  - spread recorder
  - arbitrage engine
- OHLCV:
  - one-shot sync for `bitcoin/binance/1m`
- Quant:
  - one-shot backfill compute on historical candles
  - one-shot cold archive export to Parquet and DuckDB read-back
  - one-shot FRED fetch + ForexFactory calendar scrape + candle event label backfill
- Quant API:
  - `/v1/quant/pattern`
  - `/v1/quant/signals`
  - `/v1/quant/signals/latest`
  - `/v1/quant/signals/summary`
- DeFi TVL:
  - one-shot sync wrote protocol and chain history rows
- Yields:
  - one-shot sync wrote latest pool rows and history rows
- Stablecoins:
  - one-shot sync wrote stablecoin backing rows and mcap history rows
- News:
  - one-shot RSS sync wrote raw article rows from four live feeds
  - one-shot content archive export wrote the markdown note to Google Drive and persisted the title/link row
- API:
  - `/docs`
  - `/openapi.json`
  - `/v1/market`
  - `/v1/market/bitcoin`
  - DeFi read endpoints
- Discord OAuth callback/session flow via mocked Discord endpoints and live Postgres
- Web3 nonce/signature flow via real ECDSA key generation and live Postgres
- Cloudflare auth worker local tests via `node --test ./index.test.ts`

## Live Counts Observed

- `defi_protocol_tvl_history`: populated
- `defi_chain_tvl_history`: populated
- `defi_yield_latest`: populated
- `defi_yield_history`: populated
- `defi_stablecoin_backing`: populated
- `defi_stable_mcap_history`: populated
- `news_articles`: populated
- `news_ai_metadata`: populated
- `news_entities`: populated
- `news_price_impact`: populated
- `news_price_impact_history`: present, awaiting completed 24h windows
- `candle_embeddings`: populated

Observed smoke values:

- DeFi TVL one-shot: `protocols=3 upserted=3 backfilled=2 chains=449 chain_upserted=449 chain_backfilled=2`
- Yield one-shot: `pools=3 upserted=3 backfilled=2`
- Stablecoin one-shot: `assets=6 upserted=2 history=8 depegs=0 skipped=4`

## Notes

- The live database schema is not identical to the earliest migration drafts in every place.
- Compatibility migrations were added where the live schema differed from the draft schema.
- Stablecoin tracking currently persists mapped stablecoins only; unmapped assets are skipped instead of failing the sync.
- Depeg scanning currently reports detections from live price data and threshold configuration. It does not yet write a dedicated depeg event table.
- News currently covers RSS ingestion, BeInCrypto Indonesia Telegram ingest, heuristic AI processing, raw article persistence, price-impact tracking, and the public read API. CryptoPanic WS/REST remains optional and unopened.
- Quant pattern search currently supports one-shot embedding generation and similarity lookup. The dwizzyBOT `/pattern` command is implemented as the consumer layer.
- Quant pattern search API is live at `/v1/quant/pattern` and returns low-confidence gated similarity results from the live pgvector store.
- Quant signal history, latest, and summary reads are exposed at `/v1/quant/signals`, `/v1/quant/signals/latest`, and `/v1/quant/signals/summary` on top of the live `signals` table.
- Discord OAuth, Web3 wallet auth, and premium gate middleware are live in the API code path.
- The IRAG gateway core is implemented in `irag/` with category fallback, Valkey caching, and request logging; provider base URLs are still env-driven.
- Cloudflare `auth-worker` is deployed and bound to `auth.dwizzy.my.id`.
- Cloudflare `irag-fallback` is deployed and bound to `api2.dwizzy.my.id`.
- External storage bridge services are implemented in `engine/storage_ext` and are wired into the engine scheduler for configurable GDrive backup and R2 sync jobs.

## Remaining Work

- Optional CryptoPanic integration
- Further IRAG provider-specific route transforms and upstream tuning
- Optional market order book API
- Phase 3 stablecoin alerting refinements
- Further hardening around alert persistence and notification delivery
- Optional extra chain deployments for `SubscriptionManager.sol` on Sonic, Abstract, and HyperEVM if you later fund the deploy wallet there.
