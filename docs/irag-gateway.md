# IRAG Gateway

The IRAG gateway is the backend wrapper service for the Indonesian REST API surface described in `irag-wrapper-prd.docx.md`.

## Implemented

- Standalone Go binary in `irag/cmd/irag`
- Shared `GET /healthz` and `GET /metrics`
- `/v1/*` gateway routing with category-based provider fallback
- L1 cache support via Valkey
- Request logging into `irag_request_log`
- Unified response envelope for JSON responses
- Raw passthrough for binary/media responses

## Route Groups

- `/v1/ai/text/*`
- `/v1/ai/image/*`
- `/v1/ai/process/*`
- `/v1/i2i/*`
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

`search` and `stalk` now use provider-aware route translators and canonical query aliases taken from the report matrix, not raw pass-throughs.
- `/v1/game/*`
- `/v1/news/*`
- `/v1/media/*`
- `/v1/upload/*`

## Environment

The service uses built-in provider base URLs by default. Optional env overrides are available if upstream hosts ever change:

- `IRAG_KANATA_URL`
- `IRAG_NEXURE_URL`
- `IRAG_RYZUMI_URL`
- `IRAG_CHOCOMILK_URL`
- `IRAG_YTDLP_URL`

Optional config:

- `IRAG_TIMEOUT_MS`
- `IRAG_CACHE_ENABLED`
- `IRAG_LOG_ENABLED`
- `IRAG_ALLOWED_ORIGINS`
- `IRAG_DEFAULT_CACHE_TTL`

## Notes

- Provider fallback is category-driven and circuit-breaker protected.
- The gateway is intentionally thin: upstream-specific transforms can be extended per route group without changing the public contract.
- The remaining PRD gaps and execution order are tracked in [IRAG Wrapper Blueprint](./irag-wrapper-implementation-blueprint.md).
- The route-by-route status matrix is tracked in [IRAG Gap Checklist](./irag-wrapper-gap-checklist.md).
- AI text routes normalize the public `ask` parameter and map it to provider-native fields such as `ask`, `text`, or `prompt`, depending on the upstream adapter.
- Direct report aliases under `/v1/ai/*` now map to the same provider surfaces as the semantic route families, including `ai4chat`, `copilot`, `deepseek`, `gpt`, `v2/gpt`, `perplexity`, `webpilot`, `z-ai`, `animagine-xl-3`, `animagine-xl-4`, `deepimg`, `flux-schnell`, and `pollinations` variants.
- AI image routes use provider-specific upstreams and may fall back from Nexure to Kanata when Nexure rejects a prompt or model shape.
- Exact report aliases `/v1/ai/generate` and `/v1/ai/image` now map to Kanata's JSON and PNG image-generation surfaces.
- AI process routes and the `/v1/i2i/*` alias map to provider-native image-editing surfaces such as `toanime`, `colorize`, `faceswap`, `upscaler`, `remini`, `removebg`, `waifu2x`, `image2txt`, `tololi`, `enhance`, `nanobanana`, and `nsfw-check`.
- Chocomilk's YouTube surfaces now map to first-class `/v1/youtube/*` routes for search, play, info, and download, with YTDLP and Nexure as fallbacks where useful.
- The report's Chocomilk LLM surface `/v1/llm/chatgpt/completions` now maps to Chocomilk first, with Nexure and Ryzumi as fallbacks, and normalizes `prompt` / `ask` / `text` into provider-friendly chat payloads.
- Search routes now include long-tail Ryzumi surfaces such as `bilibili`, `bmkg`, `chord`, `harga-emas`, `jadwal-sholat`, `kurs-bca`, `lens`, `mahasiswa`, `pixiv`, and `weather`, with canonical query aliases preserved.
- Downloader routes now apply provider-specific path and query rewrites for AIO and the main media matrix, including Twitter/X, SoundCloud playlist, Bstation/Bilibili, Videy, Sfile, Shopee video, Nhentai, and longer timeout windows for playlist/subtitle/mediafire-style jobs.
- Utility routes now include a local `POST /v1/tobase64` file-to-base64 converter and a report-aligned `POST /v1/utility/upscale` alias that rewrites to Nexure's `/api/tools/upscale`, with Ryzumi as fallback.
- Islamic routes split by capability: `quran` / `tafsir` / `topegon` prefer YTDLP first, while prayer schedule and khutbah stay on Kanata first.
- Tool routes now translate to provider-native shapes for `translate`, `kbbi`, `ipinfo`, `weather`, `cekresi`, `qr`, `ssweb`, `carbon`, `pln`, `pajak/jabar`, `removebg`, `brat`, `brat/animated`, `shorturl`, `listbank`, `cekbank`, `gsmarena`, `distance`, `iphonechat`, `whois`, `check-hosting`, `hargapangan`, `mc-lookup`, `qris-converter`, `turnstile-bypass`, `turnstile/sitekey`, `yt-transcript`, `currency-converter`, `subdofinder`, `isrc`, and `nsfw`.
- Misc IP whitelist checks now map to Ryzumi via `/v1/misc/ip-whitelist-check`.
- BMKG routes now map canonically to Kanata for `earthquake`, `earthquake/felt`, `weather`, `weather/village`, `provinces`, and `region/search`, with the documented `provinsi`, `adm4`, and `q` parameters preserved or normalized as needed.
- Anime, manga, film, drama, news, and LK21 routes now translate to Otakudesu, Komiku, NontonFilm, LK21, and CNN/Top News provider surfaces based on the report matrix.
- Direct report aliases `/v1/otakudesu/*` and `/v1/komiku/*` now map to the same Otakudesu/Komiku provider surfaces as the semantic `/v1/anime/*` and `/v1/manga/*` routes.
- Novel routes now translate to Chocomilk's home, hot-search, search, genre, and chapters surfaces, with `/v1/novel` aliasing the home page.
- Media and upload routes now map to Kanata TV and provider-specific upload surfaces (`Nexure`, `Kanata`, `Ryzumi`) with explicit route tests.
- Grow A Garden routes now split by sub-surface: catalog endpoints prefer Ryzumi, while stock prefers Nexure first and falls back to Ryzumi.
- BSW CCTV, Dramabox, and server-info now map to Nexure report surfaces via `/v1/tools/cctv*`, `/v1/dramabox*`, and `/v1/misc/server-info`.
- The live status matrix separates gateway correctness from upstream policy blocks; many `403` / `1010` failures are provider-side and not route bugs.
