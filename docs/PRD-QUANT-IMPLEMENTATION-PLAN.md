# PRD-QUANT Implementation Plan
**Version:** 1.0.0  
**Last Updated:** 2026-03-19  
**Status:** Execution Plan  
**Scope:** `dwizzyBRAIN/quant/` only

## Purpose

This plan breaks `docs/PRD-QUANT.md` into small, verifiable implementation steps. Each step should be completable in one focused PR or one short work session, with a clear verification gate before the next step starts.

Current state:
- The realtime quant loop is already live.
- Core indicators, scoring, funding sentiment, anomaly flags, signal caching, and pub/sub output are implemented.
- What remains is to harden the service contract, expand indicator coverage, add backfill/event/pattern layers, and wire deployment/ops.

## Dependency Graph

```
Step 1  ──► Step 2 ──► Step 3 ──► Step 4 ──► Step 5 ──► Step 6
   │                          │                          │
   └──────────────► Step 7 ◄──┴──────────────► Step 8 ◄─┘
```

- Step 1 must land first.
- Steps 2 and 3 can be partially parallelized after Step 1 if they touch disjoint files.
- Steps 4, 5, and 6 depend on the earlier data model and persistence work.
- Step 7 is an ops/deployment gate that can run after Step 1 or alongside Step 2.
- Step 8 is the final API/consumer integration gate.

## Step 1: Quant Runtime Contract and Packaging

**Goal:** make `quant/` deployable as a first-class service with explicit config, health, and runtime flags.

**Work items**
- Add `Dockerfile.quant`.
- Add `quant` service wiring to docker compose / deployment docs.
- Define and document env contract:
  - `VALKEY_URL`
  - `QUANT_WINDOW_SIZE`
  - `QUANT_SIGNAL_TTL_SECONDS`
  - `QUANT_PUBLISH_SIGNALS`
  - `QUANT_CACHE_PREFIX`
  - `QUANT_SIGNAL_CHANNEL_PREFIX`
- Add a small healthcheck entrypoint or CLI mode.
- Keep secret loading compatible with `NAME` / `NAME_FILE`.

**Verification**
- `python -m compileall quant/src`
- `pytest quant/tests`
- `docker compose config` or equivalent config validation

**Exit criteria**
- Service can be started in a container without code changes.
- Env contract is documented and stable.

**Status:** completed for packaging/runtime contract and quant data-layer migrations.

## Step 2: Persist Hot Quant Signals

**Goal:** persist computed quant signals into the hot storage layer, not only pub/sub.

**Work items**
- Write `QuantSignal` to Valkey hot cache with TTL.
- Store the latest signal per `symbol/timeframe`.
- If needed, add Timescale persistence for indicator snapshots.
- Keep publish-on-success behavior unchanged.

**Verification**
- `pytest quant/tests`
- live smoke:
  - push OHLCV sample data
  - confirm `signal:{symbol}:{timeframe}` exists in Valkey
  - confirm published message is emitted

**Exit criteria**
- A fresh subscriber can read the latest signal from Valkey without replaying history.

**Status:** completed and live-smoked against Valkey with `signal:BTCUSDT:1m`.

## Step 3: Expand Indicator Coverage

**Goal:** broaden `quant/indicators.py` beyond the bootstrap set.

**Work items**
- Add remaining trend / momentum / volatility / volume indicators from PRD-QUANT.
- Add derived features:
  - distance-from-EMA fields
  - wick/body ratios
  - BB position / width
  - ATR ratio
  - slope fields
- Add candlestick pattern helpers.
- Keep the realtime path backward compatible.

**Verification**
- `pytest quant/tests`
- new indicator unit tests for representative samples
- compile check for the whole `quant` package

**Exit criteria**
- The realtime row includes the richer indicator set without breaking existing consumers.

**Status:** the full indicator catalog from Section 5 is implemented, including trend, momentum, volatility, volume, candlestick patterns, Supertrend, Ichimoku, pivot/Fib levels, and lightweight SMC heuristics. Yahoo Finance multi-asset backfill is live for stocks, forex, and commodities. Outcome prediction training is live on fingerprint + labels. Local embedding generation is fully in-process. The quant slice is functionally complete.

## Step 4: Backfill Pipeline

**Goal:** make `quant/` capable of historical processing and cold storage export.

**Work items**
- [x] Add `quant/backfill/fetch_ohlcv.py`.
- [x] Add `quant/backfill/compute_bulk.py`.
- [x] Add cold archive export helpers.
- [x] Ensure historical candles can be read, transformed, and stored in batches.

**Verification**
- [x] small backfill smoke on BTC/ETH sample window
- [x] compare row counts before/after backfill
- [x] validate output file structure for cold archive

**Exit criteria**
- [x] Historical data can be backfilled deterministically in batch mode.

**Status:** completed with live DB backfill + Parquet cold archive export.

## Step 5: Macro Event System

**Goal:** attach macro context to each candle.

**Work items**
- [x] Add macro event ingestion scripts.
- [x] Add event labeling logic.
- [x] Generate composite `macro_environment` labels.
- [x] Persist event labels alongside candles.

**Verification**
- [x] run a limited event-label backfill on a small date window
- [x] confirm labels are generated for candles near known events

**Exit criteria**
- [x] Candles can be labeled by event proximity, regime, and surprise context.

**Status:** completed for the FRED + ForexFactory macro event slice and candle labeling pipeline.

## Step 6: Pattern Engine

**Goal:** build similarity search and historical analog output.

**Work items**
- Add fingerprint vector generation.
- Add pgvector bulk load for embeddings.
- Add similarity query and outcome aggregation.
- Add confidence gating for sample size.

**Verification**
- run a tiny embedding backfill
- execute one similarity search end to end
- confirm top-matches and outcome stats are returned

**Exit criteria**
- The system can answer “when has this happened before?” with a statistically filtered result set.

**Status:** completed for the quant-side pattern engine slice, the public API pattern endpoint, and the dwizzyBOT `/pattern` consumer command.

## Step 7: Deployment and Ops Hardening

**Goal:** make quant safe to run with the rest of the stack.

**Work items**
- Add docker-compose wiring for `quant`.
- Add resource limits and restart policy.
- Add logs / metrics conventions.
- Ensure `VALKEY_URL` and any other secret-bearing config use `NAME_FILE` compatibility.

**Verification**
- `docker compose config`
- container start smoke
- crash/restart smoke if applicable

**Exit criteria**
- Quant can be started/stopped with the stack and does not require manual fixes.

**Status:** partial. Runtime/container wiring is in the repo, but full stack rollout is still an ops concern.

## Step 8: Consumer Integration

**Goal:** expose quant output to downstream consumers.

**Work items**
- Add API read endpoint for quant signal lookups.
- Add pattern lookup endpoint if Step 6 is done.
- Connect downstream consumers only after the output contract is stable.

**Verification**
- API tests for new quant endpoints
- one live smoke request against `api/`

**Exit criteria**
- Frontend / bot / agent consumers can use the quant outputs without touching internal service state.

**Status:** completed for the consumer integration slice.

## Suggested Execution Order

1. Step 1: runtime contract and packaging.
2. Step 2: hot signal persistence.
3. Step 3: indicator expansion.
4. Step 4: backfill pipeline.
5. Step 5: macro event system.
6. Step 6: pattern engine.
7. Step 7: deployment hardening.
8. Step 8: consumer integration.

## Anti-Patterns to Avoid

- Do not build the backfill pipeline before the realtime signal contract is stable.
- Do not add macro labeling before candles and features are structurally stable.
- Do not expose API endpoints for pattern search before embeddings are generated and verified.
- Do not make `quant/` depend on `engine/` internals beyond the pub/sub and storage contracts already defined.
- Do not keep secrets only in plain `.env` for production deployment; use `NAME_FILE` or a secret manager.
