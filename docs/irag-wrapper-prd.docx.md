   
**dwizzyOS**  
**Indonesian REST API Gateway**  
   
   
**Product Requirements Document**  
API Wrapper — Production-Grade Architecture  
Version 1.0.0  ·  March 2026  
   
| | |  
|-|-|  
| **Field** | **Value** |   
| Document Title | Indonesian REST API Gateway — API Wrapper PRD |   
| Product Code | dwizzyOS / irag-wrapper |   
| Author | Rijal (dwizzyOS Project) |   
| Status | Draft v1.0.0 |   
| Date | March 2026 |   
| Upstream APIs | KanataAPI, Nexure, Ryzumi, Chocomilk, YTDLP |   
| Total Endpoints | 300+ (upstream)  ·  ~180 exposed (wrapper) |   
| Stack | Go (Fiber) + Valkey + TimescaleDB + Docker |   
   
# **Table of Contents**  
   
   
   
   
# **1. Executive Summary**  
   
The Indonesian REST API Gateway (irag-wrapper) is a production-grade Go service that aggregates, normalises, and re-exposes five upstream freemium REST API providers — KanataAPI, Nexure API, Ryzumi API, Chocomilk API, and YTDLP API — behind a single, well-documented, versioned interface.  
   
The wrapper eliminates upstream inconsistencies (divergent parameter names, non-standard error shapes, provider-specific auth), implements intelligent fallback routing, in-memory + persistent caching via Valkey/TimescaleDB, and exposes every feature under a consistent, OpenAPI 3.1-compliant schema accessible from dwizzyOS engine, dwizzyBOT, and any third-party consumer.  
   
## **1.1 Problem Statement**  
- Five upstream APIs each expose different base URLs, auth mechanisms, parameter conventions, and error codes.  
- Several endpoints overlap (e.g., YouTube download on KanataAPI, Nexure, and Ryzumi); consumers must manually pick the best provider.  
- Upstream reliability varies: some endpoints fail silently; some require freemium upgrade; no unified monitoring exists.  
- No shared caching: repeated identical requests hammer upstream servers and waste bandwidth.  
- No observability: latency, error rates, and provider health are invisible to the rest of dwizzyOS.  
   
## **1.2 Solution Overview**  
- Single base URL with versioned path prefix (/v1/).  
- Automatic primary/fallback provider resolution per logical endpoint category.  
- Unified JSON response envelope across all endpoints.  
- Valkey (Redis-compatible) L1 cache + TimescaleDB L2 cache for hot/warm data.  
- Prometheus metrics + structured JSON logs for full observability.  
- $0/month infrastructure — runs on existing homelab Mini PC within Docker Compose stack.  
   
# **2. Upstream API Providers**  
   
All five upstream providers are consumed exclusively by the wrapper. Consumers of irag-wrapper never call upstream APIs directly.  
   
| | | | | | | |  
|-|-|-|-|-|-|-|  
| **Provider** | **Base URL** | **Version** | **Auth** | **Endpoints** | **Reliability** | **Tier** |   
| KanataAPI | https://api.kanata.web.id | 2.1.0 | None | 40+ | ⭐⭐⭐⭐⭐ Highest | **🆓 Free** |   
| Nexure API | https://api.ammaricano.my.id | 1.0.0 | None | 78 | ⭐⭐⭐⭐⭐ High | **🆓 Free** |   
| Ryzumi API | https://api.ryzumi.net | 9.0.0 | None (free tier) | 115 | ⭐⭐⭐ Medium | **💎 Freemium** |   
| Chocomilk API | https://chocomilk.amira.us.kg | v1.3.24 | None | 30+ | ⭐⭐⭐ Medium | **🆓 Free** |   
| YTDLP API | https://ytdlpyton.nvlgroup.my.id | 4.0.0 | X-API-Key header | 50+ | ⭐⭐ Lower | **🔑 Key Required** |   
   
## **2.1 KanataAPI — Primary Provider**  
**Technology: **Elixir / Phoenix  
**License: **MIT  
**Auth: **None required — all endpoints free  
**Strengths: **YouTube downloads, BMKG real-time data, Islamic content (Quran, Sholat, Hadith), consistent 200 responses  
**Known Failures: **/instagram/fetch (ECONNRESET), /facebook/fetch (400), /reddit/fetch (400), /mediafire/fetch (400)  
**Wrapper Role: **PRIMARY for: YouTube, BMKG, Quran, Sholat, News, Translate, KBBI, TempMail, AI Image  
   
## **2.2 Nexure API — Universal Fallback & AI Primary**  
**Technology: **Node.js  
**License: **MIT  
**Auth: **None required  
**Strengths: **19 AI endpoints (GPT, Gemini, DeepSeek, Groq, Webpilot), best universal downloader (AIO), Instagram/Spotify/GDrive  
**Wrapper Role: **PRIMARY for: AI text, AIO downloader, Instagram, Spotify, GDrive, Scribd, SoundCloud, Bstation, Stalk — FALLBACK for: YouTube, TikTok, Pinterest  
   
## **2.3 Ryzumi API — Search & Niche Tools**  
**Version: **9.0.0 (115 endpoints — largest provider)  
**Auth: **None (free tier); Donator plan for AI/advanced tools  
**Strengths: **17 search endpoints (Google, YouTube, Spotify, Pinterest, Lyrics, BMKG), 8 stalk endpoints, image tools, Otakudesu/Komiku  
**Limitations: **AI and some tool endpoints require paid 'Donator' plan; reliability 3/5  
**Wrapper Role: **PRIMARY for: Search (Google, YouTube search, Spotify search, Pinterest search, Lyrics), Stalk (GitHub, ML), Anime (Ryzumi Otakudesu), Image tools  
   
## **2.4 Chocomilk API — Novel & Niche**  
**Version: **v1.3.24  
**Tagline: **Sweet and smooth integration — Freemium  
**Strengths: **Novel content (search, chapters, genre, hot), Tidal LOSSLESS audio, NSFW detection, Capcut downloader, Twitter downloader  
**Wrapper Role: **PRIMARY for: Novel, Tidal, Twitter/X downloader, Capcut, Deezer, ISRC metadata — FALLBACK for: Facebook, Pinterest  
   
## **2.5 YTDLP API — Premium YouTube & Spotify**  
**Base URL: **https://ytdlpyton.nvlgroup.my.id  
**Auth: **X-API-Key header required  
**Free Limits: **720p max video, 100MB max, 10 RPM  
**Strengths: **YouTube playlist download, subtitle download, Spotify full playlist ZIP, Apple Music, SunoAI music generation, Grow A Garden live stock  
**Wrapper Role: **PRIMARY for: YouTube playlist DL, Subtitle DL, Apple Music, SunoAI, Grow A Garden game data — FALLBACK for: Spotify, Mediafire  
   
# **3. Architecture Overview**  
   
## **3.1 High-Level Design**  
The wrapper is implemented as a single Go binary using the Fiber web framework. It sits between all consumers (dwizzyOS engine, dwizzyBOT Discord bot, and external clients) and the five upstream APIs.  
   
Request flow: Consumer → irag-wrapper → [Valkey L1 Cache check] → [TimescaleDB L2 check] → Upstream API call (primary → fallback) → Normalize → Cache → Return unified envelope.  
   
## **3.2 Component Map**  
| | | |  
|-|-|-|  
| **Layer** | **Technology** | **Role** |   
| HTTP Server | Go / Fiber v2 | Route handling, middleware pipeline, graceful shutdown |   
| Routing | Fiber Router + Groups | Versioned /v1/ namespace, category grouping |   
| Cache L1 (hot) | Valkey (Redis-compatible) | In-memory TTL cache for high-frequency responses |   
| Cache L2 (warm) | TimescaleDB | Persistent cache + request logs + metric snapshots |   
| Provider Manager | Go internal package | Provider registry, health checks, fallback chain resolution |   
| HTTP Client | Go net/http + retries | Upstream calls with timeout, retry, exponential backoff |   
| Response Normalizer | Go internal package | Maps upstream shapes → unified envelope |   
| Config | YAML + env vars | Provider keys, TTLs, feature flags |   
| Observability | Prometheus + Zap logger | Metrics endpoint /metrics, structured JSON logs |   
| Container | Docker Compose | Co-located with dwizzyOS engine stack |   
   
## **3.3 Unified Response Envelope**  
Every response from irag-wrapper — regardless of upstream provider — returns the following JSON structure:  
   
| | | | |  
|-|-|-|-|  
| **Field** | **Type** | **Always Present** | **Description** |   
| ok | boolean | Yes | true = success, false = error |   
| code | integer | Yes | HTTP-equivalent status code |   
| data | any | On success | Normalized payload specific to endpoint |   
| error | object | On error | { message: string, upstream: string, details?: any } |   
| meta | object | Yes | { provider: string, latency_ms: int, cached: bool, cache_ttl: int } |   
| timestamp | string | Yes | ISO 8601 UTC timestamp |   
   
## **3.4 Caching Strategy**  
| | | | |  
|-|-|-|-|  
| **Data Category** | **L1 TTL (Valkey)** | **L2 TTL (TimescaleDB)** | **Notes** |   
| Real-time (earthquake, weather) | 60s | 5m | BMKG data — frequent polling |   
| Semi-static (Quran, Hadith, KBBI) | 1h | 24h | Never changes; long TTL |   
| Search results | 5m | 1h | Same query → same result |   
| Download links | 30m | 2h | Links expire; moderate TTL |   
| AI text responses | 0 (no cache) | — | Non-deterministic; skip cache |   
| AI image generation | 10m | 1h | Same prompt → same image |   
| News / TV listings | 10m | 1h | Changes hourly |   
| Anime/Manga content | 15m | 3h | Episode list changes rarely |   
| Stalk / profile data | 5m | 30m | Social profiles can update |   
| Game data (Grow A Garden stock) | 30s | 5m | Live stock tracker |   
   
## **3.5 Fallback Chain Design**  
Each logical endpoint category defines an ordered provider chain. The wrapper tries each provider in order until one succeeds. Failed providers are temporarily marked unhealthy and skipped for a configurable backoff window (default: 60s).  
   
| | | | |  
|-|-|-|-|  
| **Category** | **Primary** | **Fallback 1** | **Fallback 2** |   
| YouTube Download | KanataAPI | Nexure API | Ryzumi API |   
| YouTube Info | KanataAPI | Nexure API | — |   
| TikTok Download | Nexure API | KanataAPI | Ryzumi API |   
| Instagram Download | Nexure API | Ryzumi API | Chocomilk |   
| Spotify Download | Nexure API | Ryzumi API | YTDLP API |   
| Pinterest Download | KanataAPI | Nexure API | Ryzumi API |   
| Facebook Download | Nexure API | Chocomilk | Ryzumi API |   
| Twitter/X Download | Chocomilk | Ryzumi API | — |   
| SoundCloud Download | Nexure API | Ryzumi API | YTDLP API |   
| GDrive Download | Nexure API | Ryzumi API | — |   
| Search (YouTube) | Ryzumi API | Nexure API | — |   
| Search (Spotify) | Ryzumi API | Nexure API | — |   
| AI Text (GPT) | Nexure API | Ryzumi API | — |   
| AI Image Gen | Nexure API | Ryzumi API | KanataAPI |   
| BMKG Weather | KanataAPI | Nexure API | Ryzumi API |   
| Anime (Otakudesu) | Nexure API | KanataAPI | — |   
| Manga (Komiku) | KanataAPI | Nexure API | — |   
   
# **4. Complete Endpoint Specification**  
   
All endpoints are prefixed with /v1. Every endpoint returns the unified response envelope defined in §3.3. Parameters marked * are required.  
   
## **4.1 Downloader Endpoints**  
   
### **4.1.1 Universal / AIO Downloader**  
| | | | |  
|-|-|-|-|  
| **Method** | **Path** | **Primary Provider** | **Status** |   
| GET | /v1/download/aio | Nexure API | **✅ Working** |   
| GET | /v1/download/aio | Ryzumi API (fallback) | **✅ Working** |   
   
| | | | |  
|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Values / Notes** |   
| url* | string (query) | Yes | Any supported platform URL (Instagram, TikTok, Twitter, Facebook, Threads, etc.) |   
   
### **4.1.2 YouTube**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/download/youtube/info | Video metadata | KanataAPI | **✅ Working** |   
| GET | /v1/download/youtube/video | Download MP4 | KanataAPI | **✅ Working** |   
| GET | /v1/download/youtube/audio | Download MP3 | KanataAPI | **✅ Working** |   
| GET | /v1/download/youtube/playlist | Download playlist | YTDLP API | **🔑 Key Required** |   
| GET | /v1/download/youtube/subtitle | Download with subtitle | YTDLP API | **🔑 Key Required** |   
| GET | /v1/download/youtube/search | Search videos | Ryzumi API | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint(s)** | **Values** |   
| url* | string | Yes | info, video, audio, playlist, subtitle | Full YouTube video or playlist URL |   
| quality | string | No | video | 144, 240, 360, 480, 720, 1080 (default: 720) |   
| q* | string | Yes | search | Free-text search query |   
| lang | string | No | subtitle | Subtitle language code (e.g. en, id) |   
   
### **4.1.3 TikTok / Douyin**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/download/tiktok | Download video/photos (no watermark) | Nexure API | **✅ Working** |   
| GET | /v1/download/tiktok/hd | HD video | YTDLP API | **🔑 Key Required** |   
| GET | /v1/download/douyin | Douyin (Chinese TikTok) | Ryzumi API | **✅ Working** |   
| GET | /v1/search/tiktok | Search TikTok videos | Chocomilk | **✅ Working** |   
   
| | | | |  
|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Values / Notes** |   
| url* | string | Yes | TikTok or Douyin video URL |   
| q* | string | Yes (search only) | Search query text |   
   
### **4.1.4 Instagram**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/download/instagram | Posts, Reels, Videos | Nexure API | **✅ Working** |   
| GET | /v1/download/instagram/story | Stories | Nexure API | **✅ Working** |   
| GET | /v1/stalk/instagram | Profile info | Nexure API | **✅ Working** |   
   
| | | | |  
|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Values / Notes** |   
| url* | string | Yes | Instagram post, reel, or story URL |   
| username* | string | Yes (stalk only) | Instagram username without @ |   
   
### **4.1.5 Spotify**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/download/spotify | Download track MP3 | Nexure API | **✅ Working** |   
| GET | /v1/download/spotify/playlist | Download playlist MP3 | YTDLP API | **🔑 Key Required** |   
| GET | /v1/search/spotify | Search tracks | Ryzumi API | **✅ Working** |   
   
| | | | |  
|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Values / Notes** |   
| url* | string | Yes | Spotify track or playlist URL |   
| q* | string | Yes (search only) | Track or artist name |   
   
### **4.1.6 Multi-Platform Downloaders**  
| | | | | | |  
|-|-|-|-|-|-|  
| **Method** | **Path** | **Platform** | **Primary** | **Fallback** | **Status** |   
| GET | /v1/download/facebook | Facebook video/reels | Nexure API | Chocomilk | **✅ Working** |   
| GET | /v1/download/twitter | Twitter/X video & photos | Chocomilk | Ryzumi API | **✅ Working** |   
| GET | /v1/download/threads | Threads media | Nexure API | Ryzumi API | **✅ Working** |   
| GET | /v1/download/pinterest | Pinterest image/video | KanataAPI | Nexure API | **✅ Working** |   
| GET | /v1/download/soundcloud | SoundCloud track | Nexure API | Ryzumi API | **✅ Working** |   
| GET | /v1/download/gdrive | Google Drive file | Nexure API | Ryzumi API | **✅ Working** |   
| GET | /v1/download/bilibili | Bilibili/BStation video | Ryzumi API | Nexure API | **✅ Working** |   
| GET | /v1/download/tidal | Tidal LOSSLESS audio | Chocomilk | — | **✅ Working** |   
| GET | /v1/download/deezer | Deezer audio | Chocomilk | — | **✅ Working** |   
| GET | /v1/download/capcut | CapCut video | Chocomilk | — | **✅ Working** |   
| GET | /v1/download/scribd | Scribd document | Nexure API | — | **✅ Working** |   
| GET | /v1/download/mediafire | Mediafire file | YTDLP API | Ryzumi API | **⚠️ Intermittent** |   
| GET | /v1/download/mega | MEGA file | Ryzumi API | — | **✅ Working** |   
| GET | /v1/download/gdrive | Google Drive | Nexure API | Ryzumi API | **✅ Working** |   
| GET | /v1/download/terabox | TeraBox file | Ryzumi API | — | **✅ Working** |   
| GET | /v1/download/pixeldrain | Pixeldrain file | Ryzumi API | — | **✅ Working** |   
| GET | /v1/download/krakenfiles | KrakenFiles | Ryzumi API | — | **✅ Working** |   
| GET | /v1/download/danbooru | Danbooru image | Ryzumi API | — | **✅ Working** |   
| GET | /v1/download/reddit | Reddit video | Nexure (AIO) | — | **✅ Working** |   
   
All multi-platform download endpoints accept a single url* (string, required) query parameter unless noted otherwise.  
   
### **4.1.7 Apple Music & SunoAI**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/download/applemusic | Apple Music track | YTDLP API | **🔑 Key Required** |   
| GET | /v1/generate/music | Generate music (SunoAI) | YTDLP API | **🔑 Key Required** |   
   
| | | | |  
|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Values / Notes** |   
| url* | string | Yes | /v1/download/applemusic — Apple Music track URL |   
| prompt* | string | Yes | /v1/generate/music — Text description of music to generate |   
   
## **4.2 Search Endpoints**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/search/youtube | Search YouTube videos | Ryzumi API | **✅ Working** |   
| GET | /v1/search/spotify | Search Spotify tracks | Ryzumi API | **✅ Working** |   
| GET | /v1/search/pinterest | Search Pinterest images | Ryzumi API | **✅ Working** |   
| GET | /v1/search/google | Google web search | Ryzumi API | **✅ Working** |   
| GET | /v1/search/google/image | Google image search | Ryzumi API | **✅ Working** |   
| GET | /v1/search/lyrics | Song lyrics search | Ryzumi API | **✅ Working** |   
| GET | /v1/search/tiktok | Search TikTok videos | Chocomilk | **✅ Working** |   
| GET | /v1/search/tidal | Search Tidal tracks | Chocomilk | **✅ Working** |   
| GET | /v1/search/anime | Search anime (Otakudesu) | Nexure API | **✅ Working** |   
| GET | /v1/search/manga | Search manga (Komiku) | KanataAPI | **✅ Working** |   
| GET | /v1/search/novel | Search novel | Chocomilk | **✅ Working** |   
| GET | /v1/search/bstation | Search Bilibili/BStation | Nexure API | **✅ Working** |   
| GET | /v1/search/cookpad | Search Cookpad recipes | Nexure API | **✅ Working** |   
| GET | /v1/search/wallpaper | Search wallpapers (Minwall) | Nexure API | **✅ Working** |   
| GET | /v1/search/pddikti | Search Indonesian university | Nexure API | **✅ Working** |   
| GET | /v1/search/film | Search films (NontonFilm) | KanataAPI | **✅ Working** |   
| GET | /v1/search/drama | Search Dramabox drama | Nexure API | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Applies To** | **Values / Notes** |   
| q* | string | Yes | All search endpoints | Search query text |   
| page | integer | No | Paginated endpoints | Page number (default: 1) |   
| limit | integer | No | Where supported | Results per page (default: 10) |   
   
## **4.3 AI / LLM Endpoints**  
   
### **4.3.1 Text Generation**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Model / Provider** | **Session Support** | **Status** |   
| GET | /v1/ai/text/gpt | ChatGPT (Nexure) | No | **✅ Working** |   
| GET | /v1/ai/text/gpt-v2 | ChatGPT V2 session (Nexure) | Yes | **✅ Working** |   
| GET | /v1/ai/text/claila | Claila (gpt-4.1-mini, gpt-5-mini) | No | **✅ Working** |   
| GET | /v1/ai/text/copilot | Microsoft Copilot (Nexure) | No | **✅ Working** |   
| GET | /v1/ai/text/gemini | Google Gemini (Nexure) | Yes | **✅ Working** |   
| GET | /v1/ai/text/deepseek | DeepSeek — coding focus (Nexure) | Yes | **✅ Working** |   
| GET | /v1/ai/text/groq | Groq — multiple models (Nexure) | Yes | **✅ Working** |   
| GET | /v1/ai/text/meta | Meta LLaMA (Nexure) | No | **✅ Working** |   
| GET | /v1/ai/text/perplexity | Perplexity (Nexure) | No | **✅ Working** |   
| GET | /v1/ai/text/pollinations | Pollinations AI (Nexure) | Yes | **✅ Working** |   
| GET | /v1/ai/text/qwen | Qwen (Nexure + Ryzumi) | Yes | **✅ Working** |   
| GET | /v1/ai/text/z-ai | Z-AI GLM-4.5 (Nexure) | No | **✅ Working** |   
| GET | /v1/ai/text/webpilot | Webpilot — live web search (Nexure) | No | **✅ Working** |   
| GET | /v1/ai/text/ai4chat | AI4Chat (Nexure) | No | **✅ Working** |   
| GET | /v1/ai/text/chatgpt-ryz | ChatGPT (Ryzumi) | Yes | **✅ Working** |   
| GET | /v1/ai/text/deepseek-ryz | DeepSeek (Ryzumi) | Yes | **✅ Working** |   
| GET | /v1/ai/text/gemini-ryz | Gemini (Ryzumi) | Yes | **✅ Working** |   
| GET | /v1/ai/text/mistral | Mistral (Ryzumi) | Yes | **✅ Working** |   
| GET | /v1/ai/text/chocomilk-gpt | ChatGPT gpt-4o-mini (Chocomilk) | No | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoints** | **Values / Notes** |   
| ask* | string | Yes | All text AI | User message / question |   
| model | string | No | copilot, claila, groq, pollinations/image | See model values per endpoint below |   
| style | string | No | deepseek, gpt-v2, groq | System prompt / persona style |   
| temperature | number | No | deepseek | 0.0–2.0 creativity control |   
| think | boolean | No | deepseek | true / false — enable chain-of-thought |   
| session | string | No | gpt-v2, gemini, deepseek, groq, qwen, pollinations | Session ID for multi-turn conversation |   
| imgUrl | string | No | gemini, pollinations, ryzumi gemini | URL of image to include in prompt |   
| imageUrl | string | No | ryzumi chatgpt, ryzumi gemini | URL of image (Ryzumi variant param name) |   
   
### **Model Values per Endpoint**  
| | |  
|-|-|  
| **Endpoint** | **model param values** |   
| /v1/ai/text/copilot | default · think-deeper · gpt-5 |   
| /v1/ai/text/claila | gpt-4.1-mini · gpt-5-mini |   
| /v1/ai/text/groq | llama-3.3-70b-versatile · llama-3.1-8b-instant · qwen/qwen3-32b · meta-llama/llama-4-maverick-17b-128e-instruct · moonshotai/kimi-k2-instruct · groq/compound · groq/compound-mini · openai/gpt-oss-120b · allam-2-7b · whisper-large-v3-turbo |   
| /v1/ai/text/pollinations | (no model param for text — use /v1/ai/image/pollinations for image models) |   
   
### **4.3.2 Image Generation**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Model / Provider** | **Response Type** | **Status** |   
| GET | /v1/ai/image/animagine3 | Animagine XL 3 (Nexure) | image/png | **✅ Working** |   
| GET | /v1/ai/image/animagine4 | Animagine XL 4 (Nexure) | image/png | **✅ Working** |   
| GET | /v1/ai/image/deepimg | DeepImg multi-style (Nexure) | image/png | **✅ Working** |   
| GET | /v1/ai/image/flux-schnell | Flux Schnell (Nexure + Ryzumi) | image/png | **✅ Working** |   
| GET | /v1/ai/image/flux-diffusion | Flux Diffusion (Ryzumi) | image/png | **✅ Working** |   
| GET | /v1/ai/image/pollinations | Pollinations multi-model (Nexure) | image/png | **✅ Working** |   
| GET | /v1/ai/image/text2img | Multi-model (Ryzumi) | image/png | **✅ Working** |   
| GET | /v1/ai/image/kanata | Pollinations via Kanata (JSON URL) | application/json | **✅ Working** |   
| GET | /v1/ai/image/kanata/direct | Direct PNG via Kanata | image/png | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoints** | **Values** |   
| prompt* | string | Yes | All image gen | Text prompt describing the image |   
| style | string | No | deepimg | default · ghibli · cyberpunk · anime · portrait · chibi · pixel art · oil painting · 3d |   
| size | string | No | deepimg | 1:1 · 3:2 · 2:3 |   
| model | string | No | pollinations/image | FLUX · Turbo · GPTImage · DALL-E 3 · Stability AI |   
| model | string | No | text2img | flux · turbo · nanobanana · zimage |   
| width | integer | No | text2img, kanata | Pixel width (e.g. 512, 768, 1024) |   
| height | integer | No | text2img, kanata | Pixel height |   
| seed | integer | No | text2img | Random seed for reproducibility |   
| enhance | boolean | No | text2img | true / false — AI prompt enhancement |   
| negative_prompt | string | No | text2img | Elements to exclude from image |   
| quality | string | No | text2img | standard · high |   
| transparent | boolean | No | text2img | true / false — transparent background |   
| aspectRatio | string | No | text2img | 16:9 · 4:3 · 1:1 · 9:16 |   
   
### **4.3.3 AI Image Processing**  
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/ai/process/toanime | Convert photo to anime style | Nexure API / Ryzumi | **✅ Working** |   
| GET | /v1/ai/process/colorize | Colorize black & white photo | Ryzumi API | **✅ Working** |   
| GET | /v1/ai/process/faceswap | Face swap between two images | Ryzumi API | **✅ Working** |   
| GET | /v1/ai/process/upscale | Upscale image (1x–4x) | Ryzumi API | **✅ Working** |   
| GET | /v1/ai/process/enhance | Enhance photo quality (Remini) | Ryzumi API | **✅ Working** |   
| GET | /v1/ai/process/removebg | Remove background | Nexure API / Ryzumi | **✅ Working** |   
| GET | /v1/ai/process/waifu2x | Waifu2x anime upscale | Ryzumi API | **✅ Working** |   
| GET | /v1/ai/process/image2txt | Generate prompt from image | Ryzumi API | **✅ Working** |   
| GET | /v1/ai/process/tololi | Convert image to loli style | Chocomilk | **✅ Working** |   
| GET | /v1/ai/process/enhance2x | 2x AI enhancement (Chocomilk) | Chocomilk | **✅ Working** |   
| GET | /v1/ai/process/nanobanana | Edit with Nano Banana (Google) | Chocomilk | **✅ Working** |   
| GET | /v1/ai/process/nsfw-check | NSFW content detection | Nexure / Chocomilk | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoints** | **Values** |   
| url* | string | Yes | Most image process endpoints | Public URL of input image |   
| style* | string | Yes | toanime | anime · loli |   
| scale* | integer | Yes | upscale | 1 · 2 · 3 · 4 |   
| original* | string | Yes | faceswap | URL of base image |   
| face* | string | Yes | faceswap | URL of face source image |   
   
## **4.4 BMKG — Indonesian Weather & Earthquake**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/bmkg/earthquake | Latest earthquakes (M 5.0+) | KanataAPI | **✅ Working** |   
| GET | /v1/bmkg/earthquake/felt | Felt earthquake reports | KanataAPI | **✅ Working** |   
| GET | /v1/bmkg/weather | Provincial weather forecast | KanataAPI | **✅ Working** |   
| GET | /v1/bmkg/weather/village | Village-level weather (ADM4) | KanataAPI | **✅ Working** |   
| GET | /v1/bmkg/provinces | List of available provinces | KanataAPI | **✅ Working** |   
| GET | /v1/bmkg/region/search | Search ADM4 region code | KanataAPI | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint** | **Values / Notes** |   
| provinsi* | string | Yes | /v1/bmkg/weather | Province slug e.g. jakarta · jawa-timur · bali · kalimantan-selatan |   
| adm4* | string | Yes | /v1/bmkg/weather/village | ADM4 code e.g. 31.71.03.1001 (get from /v1/bmkg/region/search) |   
| q* | string | Yes | /v1/bmkg/region/search | Village or district name |   
   
## **4.5 Islamic Content**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/islamic/quran | List all 114 surahs | KanataAPI | **✅ Working** |   
| GET | /v1/islamic/quran/{nomor} | Surah detail + ayat + audio | KanataAPI | **✅ Working** |   
| GET | /v1/islamic/sholat/city/{nama} | Search city for prayer schedule | KanataAPI | **✅ Working** |   
| GET | /v1/islamic/sholat/{id_kota} | Prayer schedule by city ID | KanataAPI | **✅ Working** |   
| GET | /v1/islamic/hadith/{collection}/{n} | Hadith by collection & number | KanataAPI | **✅ Working** |   
| GET | /v1/islamic/khutbah | List of Friday sermon materials | KanataAPI | **✅ Working** |   
| GET | /v1/islamic/khutbah/detail | Full sermon text | KanataAPI | **✅ Working** |   
| GET | /v1/islamic/tafsir | Quran tafsir commentary | YTDLP API | **✅ Working** |   
| GET | /v1/islamic/topegon | Latin to Pegon script converter | YTDLP API | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint** | **Values** |   
| nomor* | integer | Yes | /v1/islamic/quran/{nomor} | 1–114 (surah number) |   
| nama* | string | Yes | /v1/islamic/sholat/city/{nama} | City/kabupaten name |   
| id_kota* | string | Yes | /v1/islamic/sholat/{id_kota} | City ID from search |   
| date | string | No | /v1/islamic/sholat/{id_kota} | Date YYYY-MM-DD (default: today) |   
| collection* | string | Yes | /v1/islamic/hadith/{collection}/{n} | bukhari · muslim · abu-dawud · tirmidhi · nasai · ibnu-majah |   
| n* | integer | Yes | /v1/islamic/hadith/{collection}/{n} | Hadith number |   
| url* | string | Yes | /v1/islamic/khutbah/detail | Khutbah URL from /khutbah list |   
   
## **4.6 Anime & Streaming**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/anime/home | Ongoing & updated anime list | Nexure (Otakudesu) | **✅ Working** |   
| GET | /v1/anime/schedule | Weekly airing schedule | Nexure (Otakudesu) | **✅ Working** |   
| GET | /v1/anime/genres | All genre list | Nexure (Otakudesu) | **✅ Working** |   
| GET | /v1/anime/genre/{genre} | Anime by genre | Nexure (Otakudesu) | **✅ Working** |   
| GET | /v1/anime/search | Search anime | KanataAPI | **✅ Working** |   
| GET | /v1/anime/detail/{slug} | Anime detail + episode list | Nexure (Otakudesu) | **✅ Working** |   
| GET | /v1/anime/episode/{slug} | Episode + mirror + download | Nexure (Otakudesu) | **✅ Working** |   
| GET | /v1/anime/batch/{slug} | Batch download links | KanataAPI | **✅ Working** |   
| GET | /v1/anime/full/{slug} | Full data in one call | Nexure | **✅ Working** |   
| GET | /v1/anime/nonce | Nonce for iframe embed | Nexure | **✅ Working** |   
| GET | /v1/anime/iframe | Get iframe embed URL | Nexure | **✅ Working** |   
| GET | /v1/donghua/home | Donghua (Chinese anime) home | KanataAPI (Anichin) | **✅ Working** |   
| GET | /v1/weebs/info | Anime info (Ryzumi) | Ryzumi API | **✅ Working** |   
| GET | /v1/weebs/character | Anime character info (Ryzumi) | Ryzumi API | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint(s)** | **Values** |   
| slug* | string | Yes | detail, episode, batch, full | Anime slug from search/home |   
| genre* | string | Yes | /v1/anime/genre/{genre} | Genre slug e.g. action · romance · fantasy |   
| q* | string | Yes | /v1/anime/search | Anime title search query |   
| url* | string | Yes | /v1/anime/iframe | Episode embed URL |   
| name* | string | Yes | /v1/weebs/info, character | Anime or character name |   
   
## **4.7 Manga & Novel**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/manga/search | Search manga (Komiku) | KanataAPI | **✅ Working** |   
| GET | /v1/manga/detail/{slug} | Manga detail | Nexure | **✅ Working** |   
| GET | /v1/manga/chapter/{slug} | Chapter page images | KanataAPI | **✅ Working** |   
| GET | /v1/manga/latest | Latest manga updates | Nexure | **✅ Working** |   
| GET | /v1/novel/home | Novel home page | Chocomilk | **✅ Working** |   
| GET | /v1/novel/hot | Hot search novels | Chocomilk | **✅ Working** |   
| GET | /v1/novel/search | Search novels | Chocomilk | **✅ Working** |   
| GET | /v1/novel/genre | Browse by genre | Chocomilk | **✅ Working** |   
| GET | /v1/novel/chapters | Novel chapter content | Chocomilk | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint(s)** | **Values** |   
| q* | string | Yes | search endpoints | Search query |   
| slug* | string | Yes | detail, chapter | Content slug from search |   
| url* | string | Yes | /v1/novel/chapters | Chapter URL from novel detail |   
| page | integer | No | paginated endpoints | Page number (default: 1) |   
   
## **4.8 Film, Drama & Streaming**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/film/search | Search films (NontonFilm) | KanataAPI | **✅ Working** |   
| GET | /v1/film/stream | Get streaming URL | KanataAPI | **✅ Working** |   
| GET | /v1/film/detail/{slug} | Movie/series detail | KanataAPI | **✅ Working** |   
| GET | /v1/drama/home | Dramabox home listing | Nexure API | **✅ Working** |   
| GET | /v1/drama/search | Search drama | Nexure API | **✅ Working** |   
| GET | /v1/lk21 | LK21 film list/home | Nexure API | **✅ Working** |   
| GET | /v1/lk21/episode/{slug} | LK21 episode streaming | Nexure API | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint(s)** | **Values** |   
| q* | string | Yes | search endpoints | Search query |   
| id* | string | Yes | /v1/film/stream | Film ID from search |   
| slug* | string | Yes | detail, lk21/episode | Content slug |   
| type | string | No | /v1/film/detail/{slug} | movie · series |   
| page | integer | No | paginated | Page number |   
   
## **4.9 Tools & Utilities**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/tools/translate | Google Translate | KanataAPI | **✅ Working** |   
| GET | /v1/tools/kbbi | Indonesian dictionary (KBBI) | KanataAPI | **✅ Working** |   
| GET | /v1/tools/ipinfo/{ip} | IP geolocation | KanataAPI | **✅ Working** |   
| GET | /v1/tools/weather | Global weather (OpenMeteo) | Nexure API | **✅ Working** |   
| GET | /v1/tools/carbon | Generate code snippet image | KanataAPI | **✅ Working** |   
| GET | /v1/tools/qr | Generate QR code | Nexure API | **✅ Working** |   
| GET | /v1/tools/screenshot | Website screenshot | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/nsfw | NSFW image check | Chocomilk | **✅ Working** |   
| GET | /v1/tools/isrc | Track metadata by ISRC code | Chocomilk | **✅ Working** |   
| GET | /v1/tools/cekresi | Indonesian package tracking | Nexure API | **✅ Working** |   
| GET | /v1/tools/pln | PLN electricity bill check | Nexure API | **✅ Working** |   
| GET | /v1/tools/pajak/jabar | West Java vehicle tax check | Nexure API | **✅ Working** |   
| GET | /v1/tools/kurs | Currency exchange rates | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/gsmarena | Phone specifications | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/distance | Distance between Indonesian cities | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/shorturl | URL shortener | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/listbank | List Indonesian banks | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/cekbank | Check bank account validity | YTDLP API | **🔑 Key Required** |   
| POST | /v1/tools/removebg | Remove image background | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/brat | Brat meme generator | Nexure API | **✅ Working** |   
| GET | /v1/tools/brat/animated | Animated brat meme | Nexure API | **✅ Working** |   
| GET | /v1/tools/iphonechat | iPhone chat screenshot generator | YTDLP API | **🔑 Key Required** |   
| GET | /v1/tools/satudata | Indonesian government datasets | KanataAPI | **⚠️ Intermittent** |   
| GET | /v1/tools/tv | Currently airing TV shows | KanataAPI | **✅ Working** |   
| GET | /v1/tools/tempmail | Create temporary email | KanataAPI | **✅ Working** |   
| GET | /v1/tools/cctv | CCTV stream directory | Nexure (BSW) | **✅ Working** |   
| GET | /v1/tools/cctv/search | Search CCTV by location | Nexure (BSW) | **✅ Working** |   
| GET | /v1/tools/cctv/{id} | CCTV stream detail | Nexure (BSW) | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint(s)** | **Values / Notes** |   
| text* | string | Yes | translate | Text to translate |   
| to* | string | Yes | translate | Target language code e.g. id · en · ja · ar |   
| from | string | No | translate | Source language code (default: auto-detect) |   
| q* | string | Yes | kbbi, weather | Word (kbbi) or city name (weather) |   
| ip* | string | Yes | /v1/tools/ipinfo/{ip} | IP address (path param) |   
| code* | string | Yes | carbon | Source code to render |   
| url* | string | Yes | screenshot, nsfw | Target URL or image URL |   
| text* | string | Yes | brat, brat/animated | Brat meme text content |   
| q* | string | Yes | satudata, cctv/search | Search query |   
| resi* | string | Yes | cekresi | Package tracking number |   
| id_pel* | string | Yes | pln | PLN customer ID |   
| plat* | string | Yes | pajak/jabar | Vehicle plate number (Jawa Barat) |   
| no_pol* | string | No | pajak/jabar | Alternative vehicle plate param |   
| isrc* | string | Yes | isrc | ISRC code e.g. USAT21900000 |   
| from* | string | Yes | distance | Origin city name |   
| to* | string | Yes | distance | Destination city name |   
| long_url* | string | Yes | shorturl | URL to shorten |   
| bank* | string | Yes | cekbank | Bank code e.g. BCA · BNI · BRI |   
| no_rek* | string | Yes | cekbank | Account number |   
| device* | string | Yes | gsmarena | Phone model name |   
   
## **4.10 Stalk / Profile Lookup**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/stalk/instagram | Instagram profile info | Nexure API | **✅ Working** |   
| GET | /v1/stalk/github | GitHub profile stats | Ryzumi API | **✅ Working** |   
| GET | /v1/stalk/mobile-legends | Mobile Legends player stats | Ryzumi API | **✅ Working** |   
| GET | /v1/stalk/free-fire | Free Fire player profile | Ryzumi API | **✅ Working** |   
| GET | /v1/stalk/valorant | Valorant player card | Ryzumi API | **✅ Working** |   
| GET | /v1/stalk/clash-of-clans | Clash of Clans account | Ryzumi API | **✅ Working** |   
| GET | /v1/stalk/clash-royale | Clash Royale account | Ryzumi API | **✅ Working** |   
| GET | /v1/stalk/npm | NPM package info | Ryzumi API | **✅ Working** |   
   
| | | | | |  
|-|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Endpoint(s)** | **Values** |   
| username* | string | Yes | instagram, github, npm | Username or package name |   
| id* | string | Yes | mobile-legends, free-fire | Player ID |   
| server* | string | Yes | mobile-legends | Server region e.g. 101 |   
| name* | string | Yes | valorant, clash-of-clans, clash-royale | Player name or tag |   
| tag* | string | No | valorant | Valorant tag (e.g. #EUW) |   
| token* | string | Yes | clash-of-clans, clash-royale | Clash API token |   
   
## **4.11 Grow A Garden (Game Data)**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/game/growagarden/crops | Crops database | YTDLP API / Ryzumi | **✅ Working** |   
| GET | /v1/game/growagarden/pets | Pets database | YTDLP API / Ryzumi | **✅ Working** |   
| GET | /v1/game/growagarden/gear | Gear database | YTDLP API / Ryzumi | **✅ Working** |   
| GET | /v1/game/growagarden/eggs | Eggs database | YTDLP API / Ryzumi | **✅ Working** |   
| GET | /v1/game/growagarden/cosmetics | Cosmetics database | YTDLP API / Ryzumi | **✅ Working** |   
| GET | /v1/game/growagarden/events | Events database | YTDLP API / Ryzumi | **✅ Working** |   
| GET | /v1/game/growagarden/stock | Live stock tracker | Ryzumi / Nexure | **✅ Working** |   
No parameters required for any Grow A Garden endpoint. Cache TTL: 30s for /stock, 1h for static databases.  
   
## **4.12 News & Media**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| GET | /v1/news/top | Top Indonesian news stories | KanataAPI | **✅ Working** |   
| GET | /v1/news/cnn | CNN Indonesia news feed | Nexure API | **✅ Working** |   
| GET | /v1/media/tv | Currently airing TV shows | KanataAPI | **✅ Working** |   
No parameters required. Cache TTL: 10 minutes.  
   
## **4.13 File Uploader**  
   
| | | | | |  
|-|-|-|-|-|  
| **Method** | **Path** | **Description** | **Primary** | **Status** |   
| POST | /v1/upload | Upload to NexureCDN | Nexure API | **✅ Working** |   
| POST | /v1/upload/kanata | Upload to KanataAPI CDN | KanataAPI | **✅ Working** |   
| POST | /v1/upload/ryzumi | Upload to RyzumiCDN | Ryzumi API | **✅ Working** |   
   
| | | | |  
|-|-|-|-|  
| **Parameter** | **Type** | **Required** | **Values / Notes** |   
| file* | binary (multipart/form-data) | Yes | Nexure: max 10MB · Ryzumi: max 100MB · Kanata: varies |   
   
## **4.14 Wrapper Meta Endpoints**  
   
| | | | |  
|-|-|-|-|  
| **Method** | **Path** | **Description** | **Auth** |   
| GET | /health | Liveness probe | None |   
| GET | /metrics | Prometheus metrics | Internal only |   
| GET | /v1/providers | Provider registry & health status | None |   
| GET | /v1/providers/{id} | Individual provider health & latency stats | None |   
| GET | /v1/cache/stats | Cache hit rates, size, eviction stats | Internal only |   
| DELETE | /v1/cache/{key} | Manual cache eviction by key | Internal only |   
   
# **5. Error Handling & Status Codes**  
   
The wrapper translates all upstream errors into the unified error envelope. Consumers never see raw upstream error messages.  
   
| | | | | |  
|-|-|-|-|-|  
| **HTTP Code** | **ok field** | **Meaning** | **Common Causes** | **Wrapper Action** |   
| 200 | true | Success | — | Return normalized data |   
| 400 | false | Bad Request | Missing required param, invalid URL | Validate before upstream call |   
| 404 | false | Not Found | Invalid slug/ID, content removed | Exhaust fallback chain → 404 |   
| 429 | false | Rate Limited | Upstream rate limit exceeded | Retry with backoff, try next provider |   
| 500 | false | Server Error | Upstream internal error | Retry once, try fallback provider |   
| 502 | false | Bad Gateway | Upstream unreachable | Mark provider unhealthy, try fallback |   
| 503 | false | Service Unavailable | All providers in fallback chain failed | Return 503 with provider states |   
| 504 | false | Gateway Timeout | Upstream took > configured timeout | Timeout, try next fallback provider |   
   
## **5.1 Error Response Shape**  
| | | |  
|-|-|-|  
| **Field** | **Type** | **Example Value** |   
| ok | false | false |   
| code | integer | 503 |   
| error.message | string | All providers failed for category: youtube-download |   
| error.upstream | string | KanataAPI, Nexure API, Ryzumi API |   
| error.details | object | { KanataAPI: 'timeout', Nexure: '500', Ryzumi: '429' } |   
| meta.provider | string | none |   
| meta.latency_ms | integer | 3420 |   
| meta.cached | boolean | false |   
| timestamp | string | 2026-03-18T10:00:00Z |   
   
# **6. Non-Functional Requirements**  
   
## **6.1 Performance**  
| | | |  
|-|-|-|  
| **Metric** | **Target** | **Notes** |   
| Cache hit response time | < 10ms | Valkey L1 cache |   
| Warm upstream response | < 500ms p95 | Single upstream call with no retry |   
| Cold chain (all fallbacks) | < 5s | Max 3 providers × ~1.5s timeout each |   
| Throughput | > 500 RPS | Per Fiber benchmarks on homelab Mini PC |   
| Concurrency | 10,000 goroutines | Go's native concurrency model |   
   
## **6.2 Reliability**  
| | |  
|-|-|  
| **Requirement** | **Detail** |   
| Uptime target | 99.5% (wrapper service — upstream APIs are out of scope) |   
| Graceful shutdown | Drain in-flight requests on SIGTERM with 30s timeout |   
| Health check interval | 30s per provider, circuit breaker opens after 3 consecutive failures |   
| Circuit breaker reset | 60s backoff window before retrying failed provider |   
| Request timeout | Per provider: configurable, default 3s; chain total max 10s |   
| Retry policy | 1 retry on 5xx or timeout before moving to next provider |   
   
## **6.3 Observability**  
- Prometheus metrics: request_total (by endpoint, provider, status_code), request_duration_seconds (histogram), cache_hit_total, provider_health_gauge  
- Structured JSON logs (Zap): request ID, endpoint, provider used, latency, status, upstream error if any  
- TimescaleDB request log table: timestamp, endpoint, provider, latency_ms, status, cached, error  
- Grafana dashboard (optional): provider health heatmap, cache hit rate chart, error rate by category  
   
## **6.4 Security**  
- No consumer API key required in default homelab deployment (internal network only)  
- Optional: X-API-Key header for external-facing deployments  
- YTDLP API key stored in Docker secret / env var — never exposed in responses or logs  
- Rate limiting: 100 RPM per consumer IP (configurable), returns 429  
- Input validation: all query params sanitized, URL params validated before upstream call  
- No PII stored: only endpoint, latency, status in logs — no user data  
   
## **6.5 Infrastructure ($0/month Target)**  
| | | |  
|-|-|-|  
| **Component** | **Resource** | **Notes** |   
| irag-wrapper binary | Mini PC (homelab) — 1–2 vCPU, 512MB RAM | Docker Compose service |   
| Valkey | Shared with dwizzyOS — 256MB max | co-located |   
| TimescaleDB | Shared with dwizzyOS — request log + cache | co-located |   
| CDN / Storage | $0 — no external CDN needed | upstream APIs serve media directly |   
| External cost | YTDLP API key only (if premium tier needed) | Free tier: 10 RPM, 720p |   
   
# **7. Implementation Roadmap**  
   
## **Phase 1 — Core Foundation (Week 1–2)**  
1. Initialize Go module with Fiber v2, configure folder structure: /cmd, /internal/provider, /internal/handler, /internal/cache, /internal/normalizer  
2. Implement provider registry with health check loop and circuit breaker  
3. Implement Valkey L1 cache client with TTL support  
4. Build unified response envelope middleware  
5. Implement KanataAPI client (highest stability → validate everything against it first)  
6. Launch: YouTube info, BMKG earthquake/weather, Quran, translate, KBBI — all with KanataAPI  
   
## **Phase 2 — Nexure Integration & AI (Week 3)**  
7. Implement Nexure API client with session management for multi-turn AI  
8. Launch: All AI text endpoints (19 models), AIO downloader, Instagram, Spotify, GDrive, Stalk  
9. Implement fallback chain for YouTube (KanataAPI → Nexure)  
10. Implement TimescaleDB L2 cache + request logging  
   
## **Phase 3 — Ryzumi, Chocomilk, YTDLP (Week 4)**  
11. Implement Ryzumi API client (largest surface: 115 endpoints, freemium model)  
12. Implement Chocomilk API client (novel, Tidal, Twitter, niche downloaders)  
13. Implement YTDLP API client with X-API-Key auth  
14. Launch: Search endpoints (Google, Spotify search, Pinterest search, Lyrics), all stalk endpoints, game data (Grow A Garden), novel content  
15. Complete fallback chains for all downloader categories  
   
## **Phase 4 — Anime, Media, Tools Completion (Week 5)**  
16. Launch: Full anime/manga/film/drama surface  
17. Launch: All tools & utilities endpoints  
18. Launch: CCTV directory (BSW/Nexure)  
19. OpenAPI 3.1 spec auto-generation from Fiber routes  
20. Integration testing against all 180+ wrapper endpoints  
   
## **Phase 5 — Observability & Hardening (Week 6)**  
21. Prometheus metrics endpoint + Grafana dashboard  
22. Load testing (k6): validate 500 RPS target  
23. Circuit breaker tuning based on real provider failure patterns  
24. Documentation: README, endpoint catalog, consumer quick-start  
25. Docker Compose integration with dwizzyOS engine stack  
   
# **8. Appendix — Complete Endpoint Count Reference**  
   
| | | | |  
|-|-|-|-|  
| **Category** | **Wrapper Endpoints** | **Upstream Endpoints Used** | **Status** |   
| Downloader (Universal/AIO) | 2 | Nexure AIO + Ryzumi AIO | **✅ Working** |   
| YouTube | 6 | KanataAPI ×7, Nexure ×2, YTDLP ×7 | **✅ Working** |   
| TikTok / Douyin | 4 | Nexure, KanataAPI, Ryzumi, YTDLP | **✅ Working** |   
| Instagram | 3 | Nexure ×3 | **✅ Working** |   
| Spotify | 3 | Nexure, Ryzumi, YTDLP | **✅ Working** |   
| Multi-platform Downloaders | 19 | Chocomilk, Nexure, Ryzumi, YTDLP | **✅ Working** |   
| Apple Music / SunoAI | 2 | YTDLP API | **🔑 Key Required** |   
| Search | 17 | Ryzumi ×10, Nexure ×7, KanataAPI ×2 | **✅ Working** |   
| AI Text Generation | 19 | Nexure ×14, Ryzumi ×5, Chocomilk ×1 | **✅ Working** |   
| AI Image Generation | 9 | Nexure ×5, Ryzumi ×3, KanataAPI ×2 | **✅ Working** |   
| AI Image Processing | 12 | Ryzumi ×8, Nexure ×3, Chocomilk ×3 | **✅ Working** |   
| BMKG Weather & Earthquake | 6 | KanataAPI ×6 | **✅ Working** |   
| Islamic Content | 9 | KanataAPI ×7, YTDLP ×2 | **✅ Working** |   
| Anime & Streaming | 14 | Nexure ×9, KanataAPI ×4, Ryzumi ×2 | **✅ Working** |   
| Manga & Novel | 9 | KanataAPI ×4, Nexure ×3, Chocomilk ×5 | **✅ Working** |   
| Film, Drama & LK21 | 7 | KanataAPI ×5, Nexure ×4 | **✅ Working** |   
| Tools & Utilities | 28 | All providers | **✅ Working** |   
| Stalk / Profile Lookup | 8 | Nexure ×2, Ryzumi ×6 | **✅ Working** |   
| Grow A Garden Game Data | 7 | Ryzumi ×7, YTDLP ×7, Nexure ×1 | **✅ Working** |   
| News & Media | 3 | KanataAPI ×2, Nexure ×1 | **✅ Working** |   
| File Uploader | 3 | Nexure, KanataAPI, Ryzumi | **✅ Working** |   
| Wrapper Meta | 6 | Internal | **✅ Working** |   
| TOTAL | ~187 | 300+ upstream endpoints | **✅ Full Coverage** |   
   
   
dwizzyOS — Indonesian REST API Gateway PRD v1.0.0  
© 2026 Rijal — Internal dwizzyOS Project Document  
