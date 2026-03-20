# dwizzyOS Auth and Account System Design (MVP)

Date: 2026-03-18  
Owner: Founding Engineer  
Scope: API auth model, session lifecycle, account profile settings, and security constraints for MVP.

## 1. Goals and Non-Goals

### Goals
- Support sign-in via Discord OAuth2 and EVM wallet signature.
- Issue JWT access tokens for API usage with plan-aware claims.
- Maintain revocable refresh sessions with rotation and reuse detection.
- Provide a minimal account profile API for user-facing settings.
- Enforce baseline security controls suitable for public MVP launch.

### Non-Goals (MVP)
- Enterprise SSO (SAML/OIDC org login).
- Password-based auth (email/password).
- Multi-tenant organization membership.
- Fine-grained per-resource RBAC beyond free vs premium gating.

## 2. Authentication Model

### 2.1 Identity Providers
- Discord OAuth2: primary social login path.
- Web3 wallet login: nonce + `personal_sign` verification on EVM chains.

### 2.2 Canonical User Model
- `user` is the canonical account.
- External identities link to a user via `user_id`.
- One user can link multiple identities:
  - `discord` identity (`provider_user_id = discord user id`).
  - `evm` identity (`provider_user_id = lowercase wallet address`).

Linking policy:
- First successful auth for an unknown identity creates a new user.
- Subsequent auth with known identity resolves to existing user.
- Identity linking API requires active authenticated session plus re-auth challenge.

### 2.3 Token Model
- Access token: short-lived JWT (15 minutes).
- Refresh token: opaque random secret stored hashed server-side.
- Token claims (access JWT):
  - `sub` (user id)
  - `sid` (session id)
  - `plan` (`free` or `premium`)
  - `roles` (MVP: `user`)
  - `iat`, `exp`, `iss`, `aud`

Signing:
- MVP: HS256 with `JWT_SECRET`.
- Future-ready: key rotation via `kid`; migrate to asymmetric keys when needed.

### 2.4 Plan and Premium Entitlement
- Plan source order for MVP:
  1. Active on-chain subscription status (`SubscriptionManager.sol`).
  2. Manual override flag in DB (support operations).
  3. Default `free`.
- Entitlement is evaluated at refresh time and embedded in access token claims.

## 3. Session Lifecycle

### 3.1 Session Types
- Web/API session: browser or client app using access + refresh.
- Optional API key is out of scope for MVP.

### 3.2 Creation
1. User completes Discord OAuth callback or wallet signature verification.
2. System resolves/creates `user` and linked identity.
3. Create `auth_session` row with metadata:
   - device label/user agent hash
   - IP hash
   - created/last_seen timestamps
4. Issue:
   - access JWT (15 min)
   - refresh token (30 days absolute, 7 days idle)

### 3.3 Rotation and Reuse Detection
- Every refresh request rotates refresh token (one-time use).
- Old refresh token is marked consumed.
- If a consumed token is presented again, mark session as compromised and revoke all session tokens for that session family.

### 3.4 Revocation and Logout
- Logout current device: revoke current session.
- Logout all devices: revoke all active sessions for `user_id`.
- Revoked sessions invalidate further refresh and fail access checks via `sid` revocation cache.

### 3.5 Expiry Rules
- Access token hard expiry: 15 minutes.
- Refresh token idle timeout: 7 days since last refresh.
- Refresh token absolute timeout: 30 days from issue.

## 4. Account Profile Settings (MVP)

### 4.1 Profile Fields
Editable fields:
- `display_name` (3-32 chars)
- `username` (3-24 chars, unique, lowercase + underscore)
- `avatar_url` (optional, URL)
- `timezone` (IANA timezone)
- `locale` (BCP-47, default `id-ID`)

Read-only/computed fields:
- `plan`
- `created_at`
- linked identities summary

### 4.2 Preferences
- `email_opt_in` (default false)
- `product_updates_opt_in` (default true)
- `security_alert_opt_in` (default true)

### 4.3 Session Management UI/API
- List active sessions (created_at, last_seen_at, device label, rough location from IP geo if available).
- Revoke individual session.
- Revoke all other sessions.

## 5. Security Constraints (MVP Baseline)

### 5.1 Authentication Hardening
- Nonce TTL for wallet sign-in: 5 minutes.
- Nonce is single-use and bound to wallet address + challenge purpose.
- Discord OAuth state parameter mandatory and validated.
- PKCE recommended for public clients.

### 5.2 Transport and Secrets
- HTTPS only in production.
- `Secure`, `HttpOnly`, `SameSite=Lax` for refresh cookie when cookie mode is used.
- Secrets managed outside repo (`JWT_SECRET`, OAuth secrets, RPC keys).

### 5.3 Rate Limiting and Abuse Controls
- Per-IP + per-identity limits on auth endpoints:
  - nonce issue: 10/min
  - signature verify: 20/min
  - token refresh: 30/min
  - oauth callback: 30/min
- Temporary lockout window for repeated failed signature verification.

### 5.4 Input/Output Security
- Strict schema validation on all auth/profile payloads.
- Store only hashed refresh tokens (`sha256` + pepper).
- Never log raw tokens, signatures, or OAuth codes.
- Standardized error responses without leaking provider internals.

### 5.5 Session Integrity
- `sid` required in access JWT and checked against revoked-session cache.
- Refresh token family tracking for theft detection.
- Optional IP/user-agent drift alerting (warn-only in MVP).

## 6. Proposed API Surface (MVP)

Auth:
- `POST /v1/auth/discord/start`
- `GET /v1/auth/discord/callback`
- `POST /v1/auth/web3/nonce`
- `POST /v1/auth/web3/verify`
- `POST /v1/auth/refresh`
- `POST /v1/auth/logout`
- `POST /v1/auth/logout-all`
- `GET /v1/auth/me`

Account:
- `GET /v1/account/profile`
- `PATCH /v1/account/profile`
- `GET /v1/account/sessions`
- `DELETE /v1/account/sessions/{sessionId}`

## 7. Data Model Additions

Recommended new tables (next migrations after `038_irag_request_log.sql`):
- `039_users.sql`
- `040_auth_identities.sql`
- `041_auth_sessions.sql`
- `042_auth_refresh_tokens.sql`
- `043_auth_nonces.sql`
- `044_account_preferences.sql`

### 7.1 users
- `id` (uuid pk)
- `username` (unique)
- `display_name`
- `avatar_url`
- `timezone`
- `locale`
- `plan_override` (nullable)
- `created_at`, `updated_at`

### 7.2 auth_identities
- `id` (uuid pk)
- `user_id` (fk users.id)
- `provider` (`discord` | `evm`)
- `provider_user_id`
- `metadata_json`
- `created_at`, `updated_at`
- unique: (`provider`, `provider_user_id`)

### 7.3 auth_sessions
- `id` (uuid pk)
- `user_id` (fk users.id)
- `status` (`active`, `revoked`, `compromised`, `expired`)
- `session_family_id` (uuid)
- `ip_hash`
- `user_agent_hash`
- `last_seen_at`
- `expires_at`
- `created_at`, `revoked_at`

### 7.4 auth_refresh_tokens
- `id` (uuid pk)
- `session_id` (fk auth_sessions.id)
- `token_hash` (unique)
- `rotated_from_token_id` (nullable fk)
- `consumed_at` (nullable)
- `expires_at`
- `created_at`

### 7.5 auth_nonces
- `id` (uuid pk)
- `wallet_address`
- `nonce`
- `purpose` (`login`, `link_identity`)
- `expires_at`
- `used_at` (nullable)
- `created_at`

## 8. Middleware Enforcement

- `auth.go` verifies JWT signature, expiry, `aud/iss`, and `sid` revocation state.
- `plan.go` gates premium-only endpoints using `plan` claim.
- If token plan is stale, client naturally updates entitlement on next refresh.

## 9. Observability and Audit

Emit structured events:
- `auth.login.success`
- `auth.login.failed`
- `auth.refresh.success`
- `auth.refresh.reuse_detected`
- `auth.session.revoked`
- `account.profile.updated`

Metrics:
- auth success/failure rates by provider
- refresh reuse incidents
- active sessions per MAU
- premium entitlement mismatch rate

## 10. Delivery Plan

Phase A (2-3 days):
- Implement migrations `039-044`.
- Add auth domain models + repositories.

Phase B (2-3 days):
- Implement Discord OAuth and Web3 nonce/verify endpoints.
- Implement JWT issue/verify + refresh rotation.

Phase C (1-2 days):
- Implement account profile/preferences/sessions endpoints.
- Add middleware checks and entitlement gating.

Phase D (1 day):
- Security hardening checks, rate-limit tuning, and integration tests.

## 11. MVP Acceptance Criteria

- User can sign in with Discord or EVM wallet and receive valid access/refresh tokens.
- Refresh rotation works and rejects token reuse.
- User can view/update profile and revoke sessions.
- Premium-gated endpoints respect token plan claims.
- Security controls (nonce TTL/single use, OAuth state, rate limits, hashed token storage) are enforced.
