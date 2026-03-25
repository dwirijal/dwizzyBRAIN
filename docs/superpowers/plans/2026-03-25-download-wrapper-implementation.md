# Download Wrapper Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose the `download` family at `api.dwizzy.my.id` through `dwizzyBRAIN` by reusing the existing IRAG download translation, fallback, cache, and upstream proxy logic.

**Architecture:** Keep `dwizzyBRAIN` as the unified public gateway and treat `download` as the first wrapped service family. Reuse the existing `irag` package as the adapter engine, but add a download-only facade so the main API can mount `/v1/download/*` without mounting all IRAG routes. Public responses must use minimized provider codes (`n`, `r`, `k`, `y`, `c`) and capability-first routes only.

**Tech Stack:** Go 1.24, stdlib `net/http`, existing `irag` package, existing `api/handler` patterns, OpenAPI JSON contract, `go test`, Docker image build.

---

## File Structure

### Existing files to modify

- `irag/service.go`
  - extract reusable route execution path for download-only serving
  - stop leaking full provider names in public metadata for wrapped family responses
- `irag/download.go`
  - remain the canonical upstream path/query mapper for download routes
- `api/cmd/api/main.go`
  - construct the download wrapper service from IRAG config
- `api/router.go`
  - register the download handler in the unified gateway
- `api/index.go`
  - advertise `/v1/download/*` on the root index
- `api/openapi.go`
  - bump the public contract version
- `api/openapi.json`
  - document the `download` route family
- `api/openapi_test.go`
  - assert the new download routes exist in the contract

### New files to create

- `irag/provider_codes.go`
  - map internal provider names to short public codes
- `irag/provider_codes_test.go`
  - contract tests for public provider code mapping
- `irag/download_family.go`
  - download-only facade over the existing IRAG service
- `irag/download_family_test.go`
  - tests for route gating, fallback metadata, and provider-code exposure
- `api/download/service.go`
  - small wrapper service that owns the IRAG download facade from the gateway side
- `api/download/service_test.go`
  - tests for gateway-side service construction and delegation
- `api/handler/download.go`
  - HTTP handler for `/v1/download/{path...}`
- `api/handler/download_test.go`
  - route and response tests for the gateway handler

### Files intentionally untouched in this slice

- `api/content/*`
- `api/news/*` aside from shared contract/version effects
- any future wrapped families such as `search`, `tools`, `anime`, `manga`

### Notes for implementation

- The repo worktree is already dirty with unrelated purge changes. Stage only task-specific files during each commit.
- Do not reintroduce `/v1/irag/*`.
- Do not expose verbose provider names in download response metadata.

## Task 1: Add Public Provider Code Mapping

**Files:**
- Create: `irag/provider_codes.go`
- Create: `irag/provider_codes_test.go`
- Modify: `irag/service.go`
- Test: `irag/provider_codes_test.go`

- [ ] **Step 1: Write the failing provider-code tests**

Add tests that assert:

```go
func TestPublicProviderCode(t *testing.T) {
    tests := map[string]string{
        "nexure":    "n",
        "ryzumi":    "r",
        "kanata":    "k",
        "ytdlp":     "y",
        "chocomilk": "c",
        "unknown":   "",
    }
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./irag -run TestPublicProviderCode -count=1`

Expected: FAIL because the mapping helper does not exist yet.

- [ ] **Step 3: Implement the minimal provider-code helper**

Add `publicProviderCode` and `publicProviderCodes` helpers in `irag/provider_codes.go`.

Implementation target:

```go
func publicProviderCode(name string) string
func publicProviderCodes(names []string) []string
```

- [ ] **Step 4: Update IRAG response writing to use public provider codes where appropriate**

Change the wrapped public metadata path so it can emit short provider codes instead of full provider names.

Implementation target inside response/meta handling:

```go
meta := map[string]any{
    "providers_used": publicProviderCodes(resp.FallbackChain),
    "partial": false,
    "fallback_used": len(resp.FallbackChain) > 1,
    "cache_status": "miss",
}
```

- [ ] **Step 5: Run the focused tests and verify they pass**

Run: `go test ./irag -run 'TestPublicProviderCode' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add irag/provider_codes.go irag/provider_codes_test.go irag/service.go
git commit -m "feat: add public provider code mapping"
```

## Task 2: Extract a Download-Only IRAG Facade

**Files:**
- Create: `irag/download_family.go`
- Create: `irag/download_family_test.go`
- Modify: `irag/service.go`
- Modify: `irag/download.go`
- Test: `irag/download_family_test.go`

- [ ] **Step 1: Write the failing download-family tests**

Cover:

- only `/v1/download/*` is accepted
- unknown download route returns `404`
- successful download response returns short provider codes
- fallback chain is exposed as short codes

Example test shape:

```go
func TestDownloadFamilyRejectsNonDownloadRoute(t *testing.T) {}
func TestDownloadFamilyUsesProviderCodesInMeta(t *testing.T) {}
func TestDownloadFamilyMarksFallbackUsed(t *testing.T) {}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./irag -run 'TestDownloadFamily' -count=1`

Expected: FAIL because the download facade does not exist.

- [ ] **Step 3: Implement the download-only facade**

Create a dedicated type, for example:

```go
type DownloadFamily struct {
    service *Service
}

func NewDownloadFamily(service *Service) *DownloadFamily
func (f *DownloadFamily) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

Rules:

- only serve `/v1/download/*`
- reuse `routeSpecForPath`, `downloadChain`, cache logic, and proxy logic
- do not mount any non-download family
- normalize public metadata to the gateway-oriented shape

- [ ] **Step 4: Keep `irag/download.go` as the single source of truth for upstream mapping**

Do not duplicate provider rewrite logic into gateway code. If a helper split is needed, do it inside `irag` only.

- [ ] **Step 5: Run focused tests and the broader IRAG suite**

Run:

- `go test ./irag -run 'TestDownloadFamily' -count=1`
- `go test ./irag -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add irag/download_family.go irag/download_family_test.go irag/service.go irag/download.go
git commit -m "feat: extract download-only irag facade"
```

## Task 3: Add a Gateway-Side Download Service

**Files:**
- Create: `api/download/service.go`
- Create: `api/download/service_test.go`
- Modify: `api/cmd/api/main.go`
- Test: `api/download/service_test.go`

- [ ] **Step 1: Write the failing service tests**

Cover:

- service builds from IRAG config
- disabled/misconfigured IRAG returns unavailable state
- service delegates to the IRAG download family

Example test shape:

```go
func TestNewServiceBuildsDownloadFamily(t *testing.T) {}
func TestServiceUnavailableWithoutConfiguredIRAG(t *testing.T) {}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./api/download -count=1`

Expected: FAIL because the package does not exist.

- [ ] **Step 3: Implement the gateway-side service**

Keep this package thin. Suggested interface:

```go
type Service struct {
    family http.Handler
}

func NewService(cfg irag.Config, cache irag.Cache, logs *irag.LogStore) *Service
func (s *Service) Enabled() bool
func (s *Service) ServeHTTP(w http.ResponseWriter, r *http.Request)
```

Implementation notes:

- do not duplicate IRAG routing logic
- use `irag.NewService(...)` plus `irag.NewDownloadFamily(...)`
- return `503` behavior through the handler if not enabled

- [ ] **Step 4: Wire the service in `api/cmd/api/main.go`**

Create the IRAG config once for this slice and instantiate the download service from it.

- [ ] **Step 5: Run focused tests**

Run:

- `go test ./api/download -count=1`
- `go test ./api/cmd/api -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add api/download/service.go api/download/service_test.go api/cmd/api/main.go
git commit -m "feat: add gateway download service"
```

## Task 4: Add the Public Download Handler and Router Wiring

**Files:**
- Create: `api/handler/download.go`
- Create: `api/handler/download_test.go`
- Modify: `api/router.go`
- Modify: `api/index.go`
- Test: `api/handler/download_test.go`

- [ ] **Step 1: Write the failing handler tests**

Cover:

- `/v1/download/{path...}` delegates to the service
- non-configured service returns `503`
- request path and query are preserved

Example:

```go
func TestDownloadHandlerDelegatesToService(t *testing.T) {}
func TestDownloadHandlerReturnsServiceUnavailableWhenDisabled(t *testing.T) {}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run: `go test ./api/handler -run 'TestDownloadHandler' -count=1`

Expected: FAIL because the handler does not exist.

- [ ] **Step 3: Implement the download handler**

Suggested shape:

```go
type downloadReader interface {
    Enabled() bool
    ServeHTTP(w http.ResponseWriter, r *http.Request)
}

type DownloadHandler struct {
    service downloadReader
}

func NewDownloadHandler(service downloadReader) *DownloadHandler
func (h *DownloadHandler) Register(mux *http.ServeMux)
```

Use a catch-all route:

```go
mux.Handle("GET /v1/download/{path...}", http.HandlerFunc(h.proxy))
```

- [ ] **Step 4: Wire the handler into `api/router.go` and expose it in the root index**

Add the download handler parameter to `api.NewRouter(...)` and include `/v1/download/*` in the root document.

- [ ] **Step 5: Run focused tests and package-level API tests**

Run:

- `go test ./api/handler -run 'TestDownloadHandler' -count=1`
- `go test ./api -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add api/handler/download.go api/handler/download_test.go api/router.go api/index.go
git commit -m "feat: expose download family in gateway router"
```

## Task 5: Update OpenAPI Contract for the Download Family

**Files:**
- Modify: `api/openapi.json`
- Modify: `api/openapi.go`
- Modify: `api/openapi_test.go`
- Test: `api/openapi_test.go`

- [ ] **Step 1: Write the failing contract assertions**

Add path assertions for at least:

- `/v1/download/aio`
- `/v1/download/youtube/info`
- `/v1/download/youtube/video`
- `/v1/download/youtube/audio`
- `/v1/download/youtube/playlist`
- `/v1/download/youtube/subtitle`
- `/v1/download/instagram`
- `/v1/download/tiktok`
- `/v1/download/spotify`

- [ ] **Step 2: Run the focused OpenAPI test and verify it fails**

Run: `go test ./api -run 'TestOpenAPISpecContainsGatewayPaths|TestOpenAPIRoute' -count=1`

Expected: FAIL because the download routes are not documented yet.

- [ ] **Step 3: Update the contract**

Rules:

- document capability-first download routes only
- do not document provider names in paths
- response examples should use provider codes like `["n","r"]`

- [ ] **Step 4: Bump the OpenAPI contract version**

Update `api/openapi.go` to a new version after the route family is added.

- [ ] **Step 5: Run the focused OpenAPI tests**

Run: `go test ./api -run 'TestOpenAPISpecContainsGatewayPaths|TestOpenAPIRoute' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add api/openapi.json api/openapi.go api/openapi_test.go
git commit -m "docs: add download family to gateway contract"
```

## Task 6: Run End-to-End Verification

**Files:**
- Modify: none unless failures require follow-up
- Test: existing packages and Docker build

- [ ] **Step 1: Run targeted package tests**

Run:

- `go test ./irag -count=1`
- `go test ./api/download ./api/handler ./api/cmd/api ./api -count=1`

Expected: PASS.

- [ ] **Step 2: Run the full Go test suite**

Run: `go test ./... -count=1`

Expected: PASS.

- [ ] **Step 3: Build the API image**

Run: `docker build -f Dockerfile.api -t dwizzybrain-api:test .`

Expected: successful image build.

- [ ] **Step 4: Smoke test one live download route locally if environment is configured**

Example:

```bash
API_PORT=18080 go run ./api/cmd/api
curl "http://127.0.0.1:18080/v1/download/youtube/info?url=https://youtube.com/watch?v=dQw4w9WgXcQ"
```

Expected:

- `200` or provider-side `4xx/5xx` with gateway envelope
- `meta.providers_used` uses short codes only
- no provider names in the public JSON body

- [ ] **Step 5: Commit verification-only fixes if required**

```bash
git add <exact-files>
git commit -m "test: finalize download family verification"
```

## Execution Notes

- Implement `download` only. Do not add `search`, `tools`, or other wrapped families in this pass.
- Keep the internal provider names intact for logs and adapter internals.
- Keep public provider identifiers short in response metadata only.
- Avoid reusing the old standalone IRAG envelope verbatim if it leaks verbose provider names.
- If route conflicts appear later with other families, the next family should reuse the same pattern: thin gateway handler plus internal adapter facade.
