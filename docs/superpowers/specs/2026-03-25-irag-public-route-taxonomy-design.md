# IRAG Public Route Taxonomy Design

Date: 2026-03-25
Status: Approved for taxonomy baseline
Scope: `api.dwizzy.my.id` public route layout for wrapped IRAG-backed service families inside `dwizzyBRAIN`

## Summary

`dwizzyBRAIN` is the unified public gateway for `dwizzyOS` at `api.dwizzy.my.id`.

IRAG-backed capabilities are exposed as top-level public route families without an `irag` prefix. Public routes are organized by user-facing capability, while provider selection and fallback remain internal to the adapter layer.

This keeps the public contract stable even if upstream providers, fallback order, or internal implementations change.

## Goals

- Make `api.dwizzy.my.id` the only public entrypoint for wrapped IRAG-style services.
- Expose routes by capability, not by upstream provider name.
- Keep `content`, `auth`, `account`, and `billing` clearly separated from wrapped utility families.
- Preserve room for provider fallback, caching, and premium gating without changing public URLs.
- Create a route taxonomy that can scale as more wrapped services are added later.

## Non-Goals

- This document does not define every endpoint parameter or response schema.
- This document does not implement route handlers.
- This document does not change canonical `content` API contracts.
- This document does not reintroduce `market`, `arbitrage`, `spread`, or `snapshot` as active gateway families.

## Design Principles

1. Public routes are grouped by capability.
2. Public routes do not mention provider names such as `kanata`, `nexure`, `ryzumi`, `chocomilk`, or `ytdlp`.
3. Fallback order is an internal adapter concern.
4. Canonical platform data stays separate from wrapped upstream surfaces.
5. Every wrapped family returns the same gateway metadata shape.

## Top-Level Namespace

The public namespace at `api.dwizzy.my.id` is split into two broad classes:

- Platform-core families
  - `content`
  - `auth`
  - `account`
  - `billing`
  - `internal`
- Wrapped service families
  - `ai`
  - `download`
  - `search`
  - `tools`
  - `stalk`
  - `bmkg`
  - `islamic`
  - `anime`
  - `manga`
  - `novel`
  - `film`
  - `drama`
  - `news`
  - `media`
  - `upload`
  - `game`
  - `misc`

No public route uses `/v1/irag/*`.

## Platform-Core Families

These are owned by `dwizzyBRAIN` as first-class platform surfaces:

- `/v1/content/*`
  - canonical content APIs backed by Neon and source adapters
- `/v1/auth/*`
  - auth-facing gateway endpoints aligned with `auth.dwizzy.my.id`
- `/v1/account/*`
  - user-scoped account, entitlement, API key, and usage surfaces
- `/v1/billing/*`
  - subscription and premium billing surfaces
- `/v1/internal/*`
  - private service-to-service routes only

These families are not considered IRAG wrappers even if they reuse adapter or cache patterns.

## Wrapped Service Families

These families expose upstream-backed capabilities without leaking provider implementation details.

### AI

- `/v1/ai/text/*`
  - text-generation and conversational models
- `/v1/ai/image/*`
  - text-to-image and image generation surfaces
- `/v1/ai/process/*`
  - image-to-image and content-processing surfaces

### Download and Search

- `/v1/download/*`
  - downloader surfaces by source or media type
- `/v1/search/*`
  - query-based discovery surfaces

Notes:

- `youtube`, `spotify`, `tiktok`, and similar sources stay nested under `download` or `search` when the capability is primarily download or search.
- We do not create a duplicate top-level `/v1/youtube/*` family while `/v1/download/youtube/*` and `/v1/search/youtube` already cover that surface.

### Utility and Lookup

- `/v1/tools/*`
  - generic tools and utility endpoints
- `/v1/stalk/*`
  - social/profile lookup surfaces
- `/v1/bmkg/*`
  - weather and earthquake capability family
- `/v1/islamic/*`
  - Quran, prayer, and Islamic content utilities
- `/v1/misc/*`
  - small utility surfaces that do not justify a dedicated family

### Entertainment and Media

- `/v1/anime/*`
  - semantic anime content wrappers
- `/v1/manga/*`
  - semantic manga content wrappers
- `/v1/novel/*`
  - novel browsing and chapter wrappers
- `/v1/film/*`
  - movie and film wrappers
- `/v1/drama/*`
  - drama wrappers
- `/v1/news/*`
  - upstream-backed news wrappers distinct from canonical platform news if needed
- `/v1/media/*`
  - general media or TV-style wrappers
- `/v1/upload/*`
  - provider-backed upload or file-host surfaces
- `/v1/game/*`
  - game-related wrapped surfaces

## Taxonomy Rules

### Rule 1: Capability First

Use the public route that best describes what the user is trying to do, not which provider happens to serve it.

Examples:

- `GET /v1/download/instagram`
- `GET /v1/search/google`
- `GET /v1/ai/text/gpt`
- `GET /v1/tools/translate`

Not:

- `GET /v1/nexure/instagram`
- `GET /v1/kanata/google`

### Rule 2: Keep Canonical and Wrapped Surfaces Separate

If a domain becomes canonical platform data, it belongs under its own platform family.

Examples:

- canonical manhwa lives under `/v1/content/manhwa/*`
- wrapped manga search can still live under `/v1/search/manga`

### Rule 3: No Duplicate Top-Level Families Without Clear Product Value

Do not create both a top-level family and a subfamily for the same capability unless the product contract genuinely needs both.

Examples:

- keep `/v1/download/youtube/*` and `/v1/search/youtube`
- do not also add `/v1/youtube/*` by default

### Rule 4: Provider Aliases Are Internal

Provider-specific path translation belongs in adapter code, not in the public API contract.

Internal examples:

- Kanata adapter
- Nexure adapter
- Ryzumi adapter
- Chocomilk adapter
- YTDLP adapter

### Rule 5: Future Wrapped Services Rejoin at the Top-Level Family

If `market` returns later as a wrapped external service, it re-enters as its own top-level public family:

- `/v1/market/quote`
- `/v1/market/ticker`
- `/v1/market/orderbook`
- `/v1/market/candles`

But it remains a thin facade and not an internal market engine inside `dwizzyBRAIN`.

## Response Contract for Wrapped Families

Every wrapped family should expose a consistent gateway envelope:

```json
{
  "data": {},
  "meta": {
    "providers_used": ["nexure", "ryzumi"],
    "partial": false,
    "fallback_used": true,
    "cache_status": "hit"
  },
  "error": null
}
```

Minimum metadata fields:

- `providers_used`
- `partial`
- `fallback_used`
- `cache_status`

## Internal Implementation Boundary

Public route families map to internal adapter groups, not directly to provider names in the URL space.

Suggested internal layout:

- `gateway/wrapped/ai`
- `gateway/wrapped/download`
- `gateway/wrapped/search`
- `gateway/wrapped/tools`
- `gateway/wrapped/stalk`
- `gateway/wrapped/bmkg`
- `gateway/wrapped/islamic`
- `gateway/wrapped/anime`
- `gateway/wrapped/manga`
- `gateway/wrapped/novel`
- `gateway/wrapped/film`
- `gateway/wrapped/drama`
- `gateway/wrapped/news`
- `gateway/wrapped/media`
- `gateway/wrapped/upload`
- `gateway/wrapped/game`
- `gateway/wrapped/misc`

Each adapter group can maintain its own:

- route registry
- provider chain
- cache policy
- query normalization
- response normalization

## Initial Endpoint Family Matrix

| Family | Public Prefix | Nature | Persistence |
| --- | --- | --- | --- |
| Content | `/v1/content/*` | canonical platform | Neon |
| Auth | `/v1/auth/*` | platform core | auth authority |
| Account | `/v1/account/*` | platform core | Supabase |
| Billing | `/v1/billing/*` | platform core | Supabase |
| AI Text | `/v1/ai/text/*` | wrapped | no canonical persistence |
| AI Image | `/v1/ai/image/*` | wrapped | optional short cache |
| AI Process | `/v1/ai/process/*` | wrapped | optional short cache |
| Download | `/v1/download/*` | wrapped | optional short cache |
| Search | `/v1/search/*` | wrapped | optional short cache |
| Tools | `/v1/tools/*` | wrapped | optional short cache |
| Stalk | `/v1/stalk/*` | wrapped | optional short cache |
| BMKG | `/v1/bmkg/*` | wrapped | short cache |
| Islamic | `/v1/islamic/*` | wrapped | short/medium cache |
| Anime | `/v1/anime/*` | wrapped | optional cache |
| Manga | `/v1/manga/*` | wrapped | optional cache |
| Novel | `/v1/novel/*` | wrapped | optional cache |
| Film | `/v1/film/*` | wrapped | optional cache |
| Drama | `/v1/drama/*` | wrapped | optional cache |
| News | `/v1/news/*` | wrapped or hybrid | optional cache |
| Media | `/v1/media/*` | wrapped | optional cache |
| Upload | `/v1/upload/*` | wrapped | no canonical persistence |
| Game | `/v1/game/*` | wrapped | optional short cache |
| Misc | `/v1/misc/*` | wrapped | optional short cache |
| Internal | `/v1/internal/*` | private | internal only |

## Recommendation

Adopt capability-first public route families now and keep provider naming completely internal.

This gives `api.dwizzy.my.id` a stable taxonomy for all wrapped services, keeps the contract clean for client apps, and makes future service replacement possible without breaking public URLs.
