# auth-worker

Cloudflare Worker for the auth edge layer.

## Purpose

- Proxies `/v1/auth/*` requests to the Go API origin.
- Verifies `dwizzy_access_token` at the edge through `/v1/auth/edge/session`.
- Applies strict security headers and credentialed CORS for approved frontend origins.

This worker does not issue tokens and does not replace the Go auth service. It is a thin edge layer in front of the existing API.

## Required Secrets

- `JWT_SECRET`

## Required Vars

- `AUTH_API_ORIGIN`
- `JWT_ISSUER`
- `JWT_AUDIENCE`
- `AUTH_ALLOWED_ORIGINS`

## Local Test

```bash
cd cloudflare/auth-worker
node --test ./index.test.ts
```

## Deploy

```bash
cd cloudflare/auth-worker
wrangler secret put JWT_SECRET
wrangler deploy
```
