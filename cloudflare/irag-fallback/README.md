# irag-fallback

Cloudflare Worker for the IRAG edge failover layer.

## Purpose

- Proxies `/v1/*` requests to the primary IRAG/backend origin.
- Falls back to a secondary origin on timeout, network failure, `408`, `429`, and `5xx`.
- Preserves IRAG business logic in `dwizzyBRAIN` instead of moving it into the Worker.

This worker is not the IRAG service itself. It is a thin resilience layer in front of the existing backend.

## Required Vars

- `IRAG_PRIMARY_ORIGIN`

## Optional Vars

- `IRAG_SECONDARY_ORIGIN`
- `IRAG_ALLOWED_ORIGINS`
- `IRAG_TIMEOUT_MS`

## Local Test

```bash
cd cloudflare/irag-fallback
node --test ./index.test.ts
```

## Deploy

```bash
cd cloudflare/irag-fallback
wrangler deploy
```
