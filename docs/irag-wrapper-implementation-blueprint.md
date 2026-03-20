# IRAG Wrapper Implementation Blueprint

Updated: March 19, 2026

This document turns `docs/irag-wrapper-prd.docx.md` into an executable plan for `dwizzyBRAIN`.

## 1. Purpose

The PRD defines a large Indonesian REST API gateway surface spanning downloader, AI, search, BMKG, Islamic content, anime, manga, film, tools, stalk, game data, news/media, and upload routes.

`dwizzyBRAIN` already has a working IRAG core gateway in `irag/` and a Cloudflare fallback edge layer in `cloudflare/irag-fallback`, but the PRD is not fully closed yet. This blueprint is the backlog for finishing the remaining route coverage, normalization, caching, observability, and deployment integration required to treat the PRD as fully implemented.

## 2. Current State

### Implemented now

- Go IRAG gateway binary under `irag/cmd/irag`
- Provider registry and fallback routing
- Built-in provider base URLs
- Optional env overrides for provider URLs
- No API key requirement in the gateway request path
- Valkey L1 cache
- Request logging
- Unified JSON envelope for JSON responses
- Raw passthrough for media/download responses
- `/health`, `/healthz`, `/v1/health`, `/metrics`
- `/v1/providers` and `/v1/providers/{id}`
- Route coverage for:
  - `/v1/ai/*`
  - `/v1/download/*`
  - `/v1/search/*`
  - `/v1/tools/*`
  - `/v1/stalk/*`
  - `/v1/bmkg/*`
  - `/v1/islamic/*`
  - `/v1/anime/*`
  - `/v1/manga/*`
  - `/v1/novel/*`
  - `/v1/film/*`
  - `/v1/drama/*`
  - `/v1/game/*`
  - `/v1/news/*`
  - `/v1/media/*`
  - `/v1/upload/*`
- Cloudflare `irag-fallback` worker as edge proxy/fallback

### Remaining gaps

- PRD route inventory is not yet exhaustively normalized one-to-one with the wrapper doc
- Some route families still need endpoint-level parameter aliases and response shape harmonization
- Exact root collection routes need to be closed systematically across all route groups
- Some upstream calls still return provider errors for sample inputs, which needs route-specific handling and/or request shaping
- Timescale persistence for wrapper logs and warm cache snapshots can be expanded
- OpenAPI contract should be generated from the final route registry, not maintained manually
- Deployment wiring to `core-infrastructure` should be standardized for the IRAG gateway binary, the edge fallback, and tunnel/public hostnames

## 3. Build Target

The target is:

1. A single canonical IRAG route registry that covers every endpoint in the PRD.
2. A gateway service that normalizes input/output across all upstream providers.
3. A cache and observability layer that records hot/warm responses and request outcomes.
4. A runtime deployment path that keeps `api.dwizzy.my.id` stable and lets `api2.dwizzy.my.id` act as an edge fallback wrapper.
5. A documented, testable public contract that matches the PRD and the live implementation.

## 4. Phased Plan

## Phase 1 - Route Inventory and Contract Closure

### Goal

Turn the PRD endpoint list into a machine-readable route registry with explicit path, method, provider order, parameter aliases, response mode, and cache policy.

### Tasks

1. Build a canonical route catalog for every section in the PRD.
2. Classify each route as:
   - JSON envelope
   - raw binary passthrough
   - multipart upload
   - download redirect/link payload
3. Add exact root route handling where the PRD expects collection routes without trailing slashes.
4. Document valid parameters and required values for each endpoint family.
5. Add regression tests for route matching only.

### Verification

- `go test ./irag/... -count=1`
- route catalog review against the PRD
- smoke test of all exact root routes and `/{group}/*` routes

### Exit Criteria

- Every PRD route family has a matching registry entry.
- No route family is left as a catch-all placeholder.
- Exact root routes and trailing-slash variants behave consistently.

## Phase 2 - Provider Adapter Completion

### Goal

Make each provider adapter handle the exact request shapes required by the PRD, including query aliases, multipart payloads, and provider-specific special cases.

### Tasks

1. Review each provider-backed route family against the PRD examples.
2. Normalize query parameter names and defaults per route.
3. Implement multipart handling for upload and image-processing routes.
4. Special-case media/download routes that need raw responses or redirects.
5. Close the missing `upload` and `media` root-route behaviors across all provider chains.
6. Make provider fallback ordering explicit per route family.

### Verification

- table-driven tests for request mapping
- live smoke against the local IRAG service
- provider-specific request/response fixture tests

### Exit Criteria

- Each route family sends the correct upstream path and parameters.
- Root collection routes no longer return `route not found`.
- Multipart upload routes are classified correctly.

## Phase 3 - Normalization, Caching, and Persistence

### Goal

Make responses deterministic and cacheable where the PRD expects it.

### Tasks

1. Finalize the unified JSON envelope for all JSON endpoints.
2. Keep raw passthrough for binary/media responses.
3. Expand Valkey cache coverage for cacheable categories.
4. Persist request logs and response metadata to the long-lived store.
5. Add cache key normalization so semantically identical requests dedupe correctly.
6. Add TTL policies per category aligned with the PRD.

### Verification

- cache hit/miss tests
- request log insertion tests
- response envelope snapshot tests

### Exit Criteria

- Repeated requests hit cache where expected.
- Non-cacheable routes bypass cache.
- Logs preserve category, provider, latency, and status.

## Phase 4 - Reliability and Observability

### Goal

Make provider health, fallback behavior, and latency visible and stable.

### Tasks

1. Keep circuit breakers and per-provider failure backoff.
2. Expose provider snapshots and provider detail endpoints.
3. Ensure `/metrics` reports useful gateway counters and latency buckets.
4. Record fallback chain and upstream error class in logs.
5. Add health probes for gateway readiness and provider availability.

### Verification

- circuit breaker tests
- fallback chain tests
- metrics endpoint smoke

### Exit Criteria

- provider failures are observable
- fallback behavior is predictable
- health endpoints are stable under load

## Phase 5 - Deployment Integration

### Goal

Make the gateway run as a first-class service in the homelab stack.

### Tasks

1. Wire the IRAG binary into `core-infrastructure`.
2. Standardize env/secrets for provider URLs and runtime flags.
3. Keep `api.dwizzy.my.id` as the primary origin path.
4. Keep `api2.dwizzy.my.id` as the fallback edge wrapper.
5. Ensure Cloudflare tunnel hostnames resolve to the intended origin.
6. Keep Docker/Caddy routing compatible with existing `dwizzyOS` services.

### Verification

- local origin smoke
- tunnel smoke
- public hostname smoke

### Exit Criteria

- `api.dwizzy.my.id` and `api2.dwizzy.my.id` both resolve correctly
- the gateway is reachable from the public edge
- deployment can be restarted without route drift

## Phase 6 - Contract Finalization

### Goal

Make the public contract easy to consume by `dwizzyBRAIN`, `dwizzyBOT`, and future clients.

### Tasks

1. Generate or maintain OpenAPI from the final route registry.
2. Synchronize the docs index and developer guide.
3. Keep a route-by-route compatibility table against the PRD.
4. Add example requests and sample valid parameters for each route family.
5. Publish a concise consumer guide for downstream services.

### Verification

- OpenAPI validation
- docs link integrity
- consumer smoke against at least one route per category

### Exit Criteria

- the wrapper contract is stable
- docs and runtime match
- downstream consumers can integrate without reading the PRD

## 5. Parallel Work Streams

After Phase 1, the following can run in parallel because they touch mostly separate route groups:

- downloader adapters
- search and AI adapters
- content domains: BMKG, Islamic, anime, manga, film, drama
- tools utilities
- upload/media handling

Shared work that should stay serial:

- route registry and contract closure
- cache/persistence policy
- deployment integration
- OpenAPI finalization

## 6. Anti-Patterns

- Do not let provider-specific behavior leak into the public contract.
- Do not leave exact root routes undocumented or untested.
- Do not rely on hardcoded API keys in the request path.
- Do not split route logic across too many unrelated packages.
- Do not treat a 403 upstream denial as a gateway bug unless the route mapping is wrong.
- Do not hand-maintain a contract that should be derived from the route registry.

## 7. Validation Matrix

Minimum smoke coverage before calling the PRD complete:

- `GET /health`
- `GET /v1/health`
- `GET /v1/providers`
- `GET /v1/providers/{id}`
- one route from each major family:
  - ai
  - download
  - search
  - tools
  - bmkg
  - islamic
  - anime
  - manga
  - novel
  - film
  - drama
  - game
  - news
  - media
  - upload

## 8. Recommended Execution Order

1. Route inventory and contract closure
2. Provider adapter completion
3. Normalization, caching, and persistence
4. Reliability and observability
5. Deployment integration
6. Contract finalization

## 9. Current Ownership Model

- `irag/` owns the core gateway binary and route registry
- `cloudflare/irag-fallback` owns edge failover
- `core-infrastructure` owns tunnel and origin routing
- `docs/` owns the public contract and implementation status

## 10. Status Summary

- Core gateway: implemented
- PRD parity: in progress
- Deployment integration: partial
- Contract finalization: pending

