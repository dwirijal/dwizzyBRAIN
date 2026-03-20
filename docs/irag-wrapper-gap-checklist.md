# IRAG Wrapper Gap Checklist

This checklist translates `docs/0_RestApiReport.md` into implementation work for `irag/`.

Status legend:
- `done` = live in the IRAG gateway and verified locally
- `partial` = route exists, but upstream behavior or coverage is incomplete
- `missing` = documented in upstream reports, but not implemented in the gateway

## 1. AI Text

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/ai/text/gemini` | `ytdlp -> nexure -> ryzumi -> chocomilk` | `ask` | `model`, `session` | done | Uses YTDLP first with `text` rewrite and `gemma-3-27b-it` fallback model. |
| `/v1/ai/text/powerbrain` | `ytdlp -> nexure -> ryzumi -> chocomilk` | `ask` | - | done | Verified live through YTDLP. |
| `/v1/ai/text/felo` | `ytdlp -> nexure -> ryzumi -> chocomilk` | `ask` | - | done | Verified live through YTDLP. |
| `/v1/ai/text/deepai-chat` | `ytdlp -> nexure -> ryzumi -> chocomilk` | `ask` | - | done | Verified live through YTDLP. |
| `/v1/ai/text/gpt` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | done | Returns 200 through Nexure. |
| `/v1/ai/text/gpt-v2` | `nexure -> ryzumi -> chocomilk` | `ask` | `session` | done | Returns 200 through Nexure. |
| `/v1/ai/text/copilot` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | done | Returns 200 through Nexure. |
| `/v1/ai/text/deepseek` | `nexure -> ryzumi -> chocomilk` | `ask` | `think`, `session` | done | Returns 200 through Nexure. |
| `/v1/ai/text/perplexity` | `nexure -> ryzumi -> chocomilk` | `ask` | - | done | Returns 200 through Nexure. |
| `/v1/ai/text/ai4chat` | `nexure -> ryzumi -> chocomilk` | `ask` | - | done | Returns 200 through Nexure. |
| `/v1/ai/text/z-ai` | `nexure -> ryzumi -> chocomilk` | `ask` | `model`, `search`, `deepthink` | done | Returns 200 through Nexure. |
| `/v1/ai/text/webpilot` | `nexure -> ryzumi -> chocomilk` | `ask` | - | done | Returns 200 through Nexure. |
| `/v1/ai/text/claila` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | partial | Route exists, but current sample requests still get `403` upstream. |
| `/v1/ai/text/meta` | `nexure -> ryzumi -> chocomilk` | `ask` | - | partial | Route exists, but current sample requests still get `403` upstream. |
| `/v1/ai/text/qwen` | `nexure -> ryzumi -> chocomilk` | `ask` | `model`, `type`, `session` | partial | Route exists with a valid default `qwen3-coder-plus` model, but current sample requests still get `403` upstream. |
| `/v1/ai/text/chatgpt-ryz` | `ryzumi -> nexure -> chocomilk` | `ask` | `session` | partial | Route exists, but current sample requests still get `403` upstream. |
| `/v1/ai/text/deepseek-ryz` | `ryzumi -> nexure -> chocomilk` | `ask` | `session` | partial | Route exists, but current sample requests still get `403` upstream. |
| `/v1/ai/text/gemini-ryz` | `ryzumi -> nexure -> chocomilk` | `ask` | `session` | partial | Route exists, but current sample requests still get `403` upstream. |
| `/v1/ai/text/mistral` | `ryzumi -> nexure -> chocomilk` | `ask` | `session` | partial | Route exists, but current sample requests still get `403` upstream. |
| `/v1/ai/text/groq` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | partial | Route exists with a valid default `groq/compound` model, but current sample requests still depend on upstream availability. |
| `/v1/ai/ai4chat` | `nexure -> ryzumi -> chocomilk` | `ask` | - | done | Direct report alias now maps to the same Nexure AI4Chat surface as the semantic text route. |
| `/v1/ai/claila` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | partial | Direct report alias exists, but current sample requests still get `403` upstream. |
| `/v1/ai/copilot` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | done | Direct report alias now maps to the same Copilot surface as the semantic text route. |
| `/v1/ai/deepseek` | `nexure -> ryzumi -> chocomilk` | `ask` | `think`, `session` | done | Direct report alias now maps to the same DeepSeek surface as the semantic text route. |
| `/v1/ai/gemini` | `ytdlp -> nexure -> ryzumi -> chocomilk` | `ask` | `model`, `session` | done | Direct report alias now maps to the same Gemini surface as the semantic text route. |
| `/v1/ai/gpt` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | done | Direct report alias now maps to the same GPT V1 surface as the semantic text route. |
| `/v1/ai/v2/gpt` | `nexure -> ryzumi -> chocomilk` | `ask` | `session` | done | Direct report alias now maps to the same GPT V2 surface as the semantic text route. |
| `/v1/ai/groq` | `nexure -> ryzumi -> chocomilk` | `ask` | `model` | partial | Direct report alias exists with a valid default `groq/compound` model, but current sample requests still depend on upstream availability. |
| `/v1/ai/meta` | `nexure -> ryzumi -> chocomilk` | `ask` | - | partial | Direct report alias exists, but current sample requests still get `403` upstream. |
| `/v1/ai/perplexity` | `nexure -> ryzumi -> chocomilk` | `ask` | - | done | Direct report alias now maps to the same Perplexity surface as the semantic text route. |
| `/v1/ai/pollinations` | `nexure -> ryzumi -> chocomilk` | `ask` | - | partial | Direct report alias exists; text mode is still upstream-sensitive from this environment. |
| `/v1/ai/qwen` | `nexure -> ryzumi -> chocomilk` | `ask` | `model`, `type`, `session` | partial | Direct report alias exists with a valid default `qwen3-coder-plus` model, but current sample requests still get `403` upstream. |
| `/v1/ai/webpilot` | `nexure -> ryzumi -> chocomilk` | `ask` | - | done | Direct report alias now maps to the same Webpilot surface as the semantic text route. |
| `/v1/ai/z-ai` | `nexure -> ryzumi -> chocomilk` | `ask` | `model`, `search`, `deepthink` | done | Direct report alias now maps to the same Z-AI surface as the semantic text route. |

## 2. AI Image

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/ai/image/deepimg` | `nexure -> ryzumi -> kanata` | `prompt` | `style`, `size` | done | Path rewrite fixed; live smoke returns `200 image/png`. |
| `/v1/ai/image/animagine-xl-3` | `nexure -> ryzumi -> kanata` | `prompt` | - | done | Live smoke returns `200 image/png` via Nexure. |
| `/v1/ai/image/animagine-xl-4` | `nexure -> ryzumi -> kanata` | `prompt` | - | done | Live smoke returns `200 image/png` via Nexure. |
| `/v1/ai/image/flux-schnell` | `nexure -> kanata -> ryzumi` | `prompt` | - | done | Live smoke returns `200` after Kanata fallback on cache-busted prompts. |
| `/v1/ai/image/pollinations` | `nexure -> kanata -> ryzumi` | `prompt` | `model` | done | Live smoke returns `200` after Kanata fallback on cache-busted prompts. |
| `/v1/ai/animagine-xl-3` | `kanata -> nexure -> ryzumi -> chocomilk` | `prompt` | - | done | Direct report alias now maps to the same image-generation surface as `/v1/ai/image/animagine-xl-3`. |
| `/v1/ai/animagine-xl-4` | `kanata -> nexure -> ryzumi -> chocomilk` | `prompt` | - | done | Direct report alias now maps to the same image-generation surface as `/v1/ai/image/animagine-xl-4`. |
| `/v1/ai/deepimg` | `kanata -> nexure -> ryzumi -> chocomilk` | `prompt` | `style`, `size` | done | Direct report alias now maps to the same image-generation surface as `/v1/ai/image/deepimg`. |
| `/v1/ai/flux-schnell` | `kanata -> nexure -> ryzumi -> chocomilk` | `prompt` | - | done | Direct report alias now maps to the same image-generation surface as `/v1/ai/image/flux-schnell`. |
| `/v1/ai/pollinations` | `kanata -> nexure -> ryzumi -> chocomilk` | `prompt` | `model` | done | Direct report alias now maps to the same image-generation surface as `/v1/ai/image/pollinations`. |
| `/v1/ai/pollinations/image` | `kanata -> nexure -> ryzumi -> chocomilk` | `prompt` | `model` | done | Direct report alias now maps to the same image-generation surface as `/v1/ai/image/pollinations/image`. |
| `/v1/ai/generate` | `kanata` | `prompt` | `model` | done | Canonical JSON image-generation alias from the report. |
| `/v1/ai/image` | `kanata` | `prompt` | - | done | Canonical direct PNG image-generation alias from the report. |

## 3. AI Process / I2I

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/ai/process/toanime` | `nexure -> ryzumi -> chocomilk` | `url` | `style`, `image_url`, `imgUrl` | done | Rewrites to provider-native `toanime` surfaces with image URL aliases preserved. |
| `/v1/ai/process/colorize` | `ryzumi -> nexure -> chocomilk` | `url` | `image_url`, `imgUrl` | done | Rewrites to provider-native `colorize` surfaces. |
| `/v1/ai/process/faceswap` | `ryzumi -> nexure -> chocomilk` | `original`, `face` | - | done | Rewrites to provider-native `faceswap` surfaces. |
| `/v1/ai/process/upscale` | `ryzumi -> nexure -> chocomilk` | `url` | `scale`, `image_url`, `imgUrl` | done | Rewrites to provider-native `upscaler` or `upscale` surfaces. |
| `/v1/ai/process/enhance` | `ryzumi -> nexure -> chocomilk` | `url` | `scale`, `image_url`, `imgUrl` | done | Rewrites to Remini / upscale-style surfaces. |
| `/v1/ai/process/removebg` | `ryzumi -> nexure -> chocomilk` | `url` | `image_url`, `imgUrl` | done | Rewrites to provider-native background removal surfaces. |
| `/v1/ai/process/waifu2x` | `ryzumi -> nexure -> chocomilk` | `url` | `image_url`, `imgUrl` | done | Rewrites to provider-native `waifu2x` surfaces. |
| `/v1/ai/process/image2txt` | `ryzumi -> nexure -> chocomilk` | `url` | `image_url`, `imgUrl` | done | Rewrites to provider-native image-to-text surfaces. |
| `/v1/ai/process/tololi` | `chocomilk -> nexure -> ryzumi` | `url` | `image_url`, `imgUrl` | done | Rewrites to Chocomilk `i2i` surface. |
| `/v1/ai/process/enhance2x` | `chocomilk -> nexure -> ryzumi` | `url` | `image_url`, `imgUrl` | done | Rewrites to Chocomilk `i2i/enhance` surface. |
| `/v1/ai/process/nanobanana` | `chocomilk -> nexure -> ryzumi` | `url` | `image_url`, `imgUrl`, `prompt` | done | Rewrites to Chocomilk `i2i/nano-banana` surface. |
| `/v1/ai/process/nsfw-check` | `nexure -> ryzumi -> chocomilk` | `url` | `image_url`, `imgUrl` | done | Rewrites to provider-native NSFW image-analysis surfaces. |
| `/v1/i2i/*` | mirrors `/v1/ai/process/*` | route alias | same as above | done | Alias path is now first-class and resolved through the same adapter. |

## 4. Downloader / Search / Tools

| Surface | Canonical route family | Status | Notes |
|---|---|---|---|
| Download | `/v1/download/*` | partial | Provider-native path rewrites and translation tests are in place for the main matrix, including AIO, YouTube, Instagram, TikTok, Spotify, Facebook, Threads, Pinterest, SoundCloud, GDrive, Bstation, Scribd, Mediafire, Mega, Terabox, Pixeldrain, Krakenfiles, Danbooru, Reddit, Apple Music, Twitter/X, Videy, Sfile, Shopee video, and Nhentai. Live validation confirms `tiktok/hd` and `douyin` via YTDLP. Most other routes still fail upstream with `403` / `1010` or provider paywalls from this environment. |
| Search | `/v1/search/*` | partial | Route family exists and is now provider-aware for `youtube`, `spotify`, `pinterest`, `google`, `google/image`, `lyrics`, `tiktok`, `tidal`, `anime`, `manga`, `film`, `bstation`, `cookpad`, `wallpaper`, `pddikti`, `drama`, `novel`, plus Ryzumi-specific `bilibili`, `bmkg`, `chord`, `harga-emas`, `jadwal-sholat`, `kurs-bca`, `lens`, `mahasiswa`, `pixiv`, and `weather`. Unit tests cover the translation matrix; live smoke is still upstream-blocked from this environment for many providers. |
| Tools | `/v1/tools/*` | partial | Provider-native rewrites are in place for `translate`, `kbbi`, `ipinfo`, `weather`, `cekresi`, `qr`, `ssweb`, `carbon`, `pln`, `pajak/jabar`, `removebg`, `brat`, `brat/animated`, `shorturl`, `listbank`, `cekbank`, `gsmarena`, `distance`, `iphonechat`, `whois`, `check-hosting`, `hargapangan`, `mc-lookup`, `qris-converter`, `turnstile-bypass`, `turnstile/sitekey`, `yt-transcript`, `currency-converter`, `subdofinder`, `isrc`, and `nsfw`. Live probes from this environment confirm `200` for `translate`, `kbbi`, `ipinfo`, `weather`, `qr`, and `pln`. `cekresi`, `pajak/jabar`, and the long-tail tool set now reach the provider with correct shapes, but some still depend on valid domain-specific inputs or provider-side availability from a clean egress IP. |
| Media | `/v1/media/*` | done | `/v1/media/tv` now maps to the Kanata TV surface (`/tv/now`) and is covered by unit tests. |
| Upload | `/v1/upload/*` | done | Canonical upload routes now map to Nexure, Kanata, and Ryzumi upload surfaces with provider-specific tests. |

## 5. BMKG

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/bmkg/earthquake` | `kanata` | - | - | done | Translates to `/bmkg/gempa` and is covered by unit tests. |
| `/v1/bmkg/earthquake/felt` | `kanata` | - | - | done | Translates to `/bmkg/gempa/dirasakan` and is covered by unit tests. |
| `/v1/bmkg/weather` | `kanata` | `provinsi` | `province`, `slug`, `q`, `query` | done | Normalizes province slug and maps to `/bmkg/cuaca`. |
| `/v1/bmkg/weather/village` | `kanata` | `adm4` | `code`, `id`, `wilayah` | done | Normalizes ADM4 code and maps to `/bmkg/cuaca/desa`. |
| `/v1/bmkg/provinces` | `kanata` | - | - | done | Translates to `/bmkg/cuaca/provinces`. |
| `/v1/bmkg/region/search` | `kanata` | `q` | `query`, `text`, `name`, `wilayah` | done | Normalizes search text and maps to `/bmkg/wilayah/search`. |

## 6. Anime / Manga / Film / News

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/weebs/anime-info` | `ryzumi` | `query` | - | done | Maps to Ryzumi anime info lookup. |
| `/v1/weebs/manga-info` | `ryzumi` | `query` | - | done | Maps to Ryzumi manga info lookup. |
| `/v1/weebs/sfw-waifu` | `ryzumi` | `tag` | - | done | Maps to Ryzumi waifu randomizer with the documented tag set. |
| `/v1/weebs/whatanime` | `ryzumi` | `url` | - | done | Maps to Ryzumi anime source finder. |
| `/v1/anime/home` | `nexure -> kanata` | - | - | done | Maps to Otakudesu home. |
| `/v1/anime/schedule` | `nexure -> kanata` | - | - | done | Maps to Otakudesu schedule. |
| `/v1/anime/genres` | `nexure -> kanata` | - | - | done | Maps to Otakudesu genre list. |
| `/v1/anime/genre/{genre}` | `nexure -> kanata` | `genre` | - | done | Normalizes genre slug and maps to `animebygenre`. |
| `/v1/anime/search` | `nexure -> kanata` | `q` | `query`, `title` | done | Maps to Otakudesu search. |
| `/v1/anime/detail/{slug}` | `nexure -> kanata` | `slug` | - | done | Maps to Otakudesu detail. |
| `/v1/anime/episode/{slug}` | `nexure -> kanata` | `slug` | - | done | Maps to Otakudesu episode. |
| `/v1/anime/batch/{slug}` | `kanata -> nexure` | `slug` | - | done | Maps to batch download. |
| `/v1/anime/full/{slug}` | `nexure -> kanata` | `slug` | - | done | Maps to Otakudesu lengkap/full. |
| `/v1/anime/nonce` | `nexure -> kanata` | - | - | done | Maps to Otakudesu nonce. |
| `/v1/anime/iframe` | `nexure -> kanata` | `url` | `embed_url`, `iframe` | done | Maps to Otakudesu iframe getter. |
| `/v1/manga/search` | `kanata -> nexure` | `q` | `query`, `title` | done | Maps to Komiku search. |
| `/v1/manga/detail/{slug}` | `nexure -> kanata` | `slug` | - | done | Maps to Komiku detail. |
| `/v1/manga/chapter/{slug}` | `kanata -> nexure` | `slug` | - | done | Maps to Komiku chapter. |
| `/v1/manga/latest` | `nexure -> kanata` | - | - | done | Maps to Komiku terbaru/latest. |
| `/v1/film/search` | `kanata -> nexure` | `q` | `query`, `title` | done | Maps to NontonFilm search. |
| `/v1/film/stream` | `kanata` | `id` | `slug`, `url` | done | Maps to NontonFilm stream. |
| `/v1/film/detail/{slug}` | `kanata -> nexure` | `slug` | - | done | Maps to NontonFilm detail. |
| `/v1/lk21` | `nexure` | - | - | done | Maps to LK21 home. |
| `/v1/lk21/episode/{slug}` | `nexure` | `slug` | - | done | Maps to LK21 episode. |
| `/v1/news/top` | `kanata -> nexure` | - | - | done | Maps to news top surface. |
| `/v1/news/cnn` | `nexure` | - | - | done | Maps to CNN Indonesia feed. |
| `/v1/stalk/ml` | `ryzumi -> nexure` | `userId` | `zoneId`, `id`, `server`, `user_id`, `zone_id` | done | Alias for Mobile Legends profile lookup. |

## 7. Novel

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/novel/home` | `chocomilk` | - | - | done | Maps to Chocomilk homepage novels. |
| `/v1/novel/hot-search` | `chocomilk` | - | `page` | done | `/v1/novel/hot` is treated as an alias for the Chocomilk hot-search surface. |
| `/v1/novel/search` | `chocomilk` | `q` | `query`, `page` | done | Normalizes the novel search query and keeps page aliases intact. |
| `/v1/novel/genre` | `chocomilk` | `genre` | `q`, `page` | done | Normalizes genre lookups and keeps page aliases intact. |
| `/v1/novel/chapters` | `chocomilk` | `url` | `chapter_url`, `href`, `link` | done | Normalizes chapter URL lookups. |

## 8. Grow A Garden

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/game/growagarden/crops` | `ryzumi -> nexure` | - | `category`, `type` | done | Maps to the Ryzumi Grow A Garden data surface. |
| `/v1/game/growagarden/pets` | `ryzumi -> nexure` | - | `category`, `type` | done | Maps to the Ryzumi Grow A Garden data surface. |
| `/v1/game/growagarden/gear` | `ryzumi -> nexure` | - | `category`, `type` | done | Maps to the Ryzumi Grow A Garden data surface. |
| `/v1/game/growagarden/eggs` | `ryzumi -> nexure` | - | `category`, `type` | done | Maps to the Ryzumi Grow A Garden data surface. |
| `/v1/game/growagarden/cosmetics` | `ryzumi -> nexure` | - | `category`, `type` | done | Maps to the Ryzumi Grow A Garden data surface. |
| `/v1/game/growagarden/events` | `ryzumi -> nexure` | - | `category`, `type` | done | Maps to the Ryzumi Grow A Garden data surface. |
| `/v1/game/growagarden/stock` | `nexure -> ryzumi` | - | - | done | Maps to `api/info/growagarden` first, then Ryzumi tool surface. |

## 9. Misc

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/misc/server-info` | `nexure` | - | - | done | Maps to Nexure server info surface. |
| `/v1/misc/ip-whitelist-check` | `ryzumi` | `ip` | `q`, `query` | done | Maps to Ryzumi IP whitelist / blacklist check surface. |

## 10. YouTube

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/youtube/search` | `chocomilk -> ytdlp -> nexure` | `q` | `query`, `ask`, `text`, `keyword`, `page` | done | Chocomilk YouTube search surface with YTDLP/Nexure fallbacks. |
| `/v1/youtube/play` | `chocomilk -> ytdlp -> nexure` | `q` | `query`, `ask`, `text`, `keyword`, `page` | done | Chocomilk play/search surface normalized to the same YouTube query aliases. |
| `/v1/youtube/info` | `chocomilk -> ytdlp -> nexure` | `url` | `link`, `source`, `target` | done | Chocomilk video info surface with YTDLP fallback. |
| `/v1/youtube/download` | `chocomilk -> ytdlp -> nexure` | `url` | `link`, `source`, `target`, `quality`, `format`, `itag`, `lang` | done | Chocomilk download surface with YTDLP fallback. |

## 11. LLM

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/llm/chatgpt/completions` | `chocomilk -> nexure -> ryzumi` | `prompt` | `ask`, `text`, `model`, `session` | done | Maps to Chocomilk ChatGPT completions first, then Nexure/Ryzumi fallbacks with canonical prompt aliases. |

## 12. Utility

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/tobase64` | `local` | `file` | `content` | done | Local utility that converts an uploaded file or raw payload into base64. `/v1/tools/tobase64` is treated as an alias. |
| `/v1/utility/upscale` | `nexure -> ryzumi` | `imgUrl` | `image_url`, `url` | done | Canonical utility upscaler route from the report. `/v1/tools/upscale` is treated as an alias and rewrites to Nexure `/api/tools/upscale`. |

## 13. Islamic Content

| Canonical route | Provider order | Required params | Optional params | Status | Notes |
|---|---|---|---|---|---|
| `/v1/islamic/quran` | `ytdlp -> kanata` | - | - | partial | Route rewrite is in place and locally verified, but live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/quran/{nomor}` | `ytdlp -> kanata` | `nomor` | - | partial | Route rewrite is in place and locally verified, but live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/tafsir` | `ytdlp -> kanata` | `surah` | - | partial | Route rewrite is in place and locally verified, but live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/topegon` | `ytdlp -> kanata` | `text` | - | partial | Route rewrite is in place and locally verified, but live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/hadith/{collection}/{n}` | `ytdlp -> kanata` | `collection`, `n` | - | partial | Route rewrite is in place and locally verified, but live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/sholat/city/{nama}` | `kanata -> ytdlp` | `nama` | - | partial | Kanata route translation exists; live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/sholat/{id_kota}` | `kanata -> ytdlp` | `id_kota` | `date` | partial | Kanata route translation exists; live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/khutbah` | `kanata -> ytdlp` | - | - | partial | Kanata route translation exists; live smoke still hits upstream `403` / `1010` from this environment. |
| `/v1/islamic/khutbah/detail` | `kanata -> ytdlp` | `url` | - | partial | Kanata route translation exists; live smoke still hits upstream `403` / `1010` from this environment. |

## 14. Missing By Report

These report entries are not yet mapped as first-class IRAG route groups or need a dedicated adapter pass:

- Kanata:
  - direct `/v1/ai/*` aliases are now mapped through the image/text adapters; remaining variations are upstream-policy dependent
- Nexure:
  - `/api/bsw/cctv/*` — implemented via `/v1/tools/cctv*`
  - `/api/dramabox/*` — implemented via `/v1/dramabox*`
  - `/api/komiku/*` — implemented via `/v1/komiku*`
  - `/api/otakudesu/*` — implemented via `/v1/otakudesu*`
  - `/api/lk21/*` — implemented via `/v1/lk21*`
  - `/api/misc/server-info` — implemented via `/v1/misc/server-info`
- Ryzumi:
  - `donator`-only AI/tool/stalk routes should remain fallback-only or premium-gated
- Chocomilk:
  - `/v1/tools/*` image/utility routes — implemented via `/v1/tools/turnstile-bypass`, `/v1/tools/nsfw`, and `/v1/tools/isrc`
- YTDLP:
  - download and topup routes are not part of IRAG scope
- Tool long-tail now implemented in code:
  - `whois`, `check-hosting`, `hargapangan`, `mc-lookup`, `qris-converter`, `turnstile-bypass`, `turnstile/sitekey`, `yt-transcript`, `currency-converter`, `subdofinder`, `nsfw`

## 15. Immediate Next Work

Priority order:
1. Treat the current temp matrix as the source of truth for verified `done` vs `partial` IRAG routes.
2. Build the long-tail downloader matrix for the remaining `partial` routes only.
3. Mark the tool routes above as `done in code` but `upstream-blocked in live smoke` until the providers stop rejecting this egress IP.

## 16. Source Notes

This checklist is derived from:
- `docs/0_RestApiReport.md`
- the live IRAG smoke results in this repository
- the current route mapping in `irag/service.go`
