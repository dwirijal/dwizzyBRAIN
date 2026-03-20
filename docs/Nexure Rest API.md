# Nexure Rest API

**Version:** `1.0.0`  
**Specification:** `OAS 3.0`  
**License:** MIT License  
**Tagline:** Free Rest API for everyone without limit.

## Informasi Umum

- **Website:** Nexure API - Website
- **Contact:** Send email to Nexure API
- **Servers:**
  - `https://api.ammaricano.my.id` — Production server
  - `http://localhost:3004` — Development server

> Catatan: dokumentasi ini dirapikan dari spesifikasi yang Anda kirim. Contoh response yang URL-nya sangat panjang dipersingkat dengan `...` agar lebih mudah dibaca. Jumlah endpoint yang terdokumentasi di file ini: **78 endpoint**.

## Ringkasan Grup Endpoint

| Grup | Jumlah Endpoint |
|---|---:|
| Uploader | 1 |
| AI | 19 |
| BSW | 3 |
| Download | 13 |
| Dramabox | 2 |
| Image | 3 |
| Info | 3 |
| Komiku | 3 |
| Search | 8 |
| Stalk | 2 |
| Tools | 7 |
| Otakudesu | 11 |
| lk21 | 2 |
| Misc | 1 |

## Daftar Endpoint Lengkap

| # | Grup | Method | Endpoint |
|---:|---|---|---|
| 1 | Uploader | `POST` | `/api/upload` |
| 2 | AI | `GET` | `/api/ai/ai4chat` |
| 3 | AI | `GET` | `/api/ai/animagine-xl-3` |
| 4 | AI | `GET` | `/api/ai/animagine-xl-4` |
| 5 | AI | `GET` | `/api/ai/claila` |
| 6 | AI | `GET` | `/api/ai/copilot` |
| 7 | AI | `GET` | `/api/ai/deepimg` |
| 8 | AI | `GET` | `/api/ai/deepseek` |
| 9 | AI | `GET` | `/api/ai/flux-schnell` |
| 10 | AI | `GET` | `/api/ai/gemini` |
| 11 | AI | `GET` | `/api/ai/gpt` |
| 12 | AI | `GET` | `/api/ai/v2/gpt` |
| 13 | AI | `GET` | `/api/ai/groq` |
| 14 | AI | `GET` | `/api/ai/meta` |
| 15 | AI | `GET` | `/api/ai/perplexity` |
| 16 | AI | `GET` | `/api/ai/pollinations` |
| 17 | AI | `GET` | `/api/ai/pollinations/image` |
| 18 | AI | `GET` | `/api/ai/qwen` |
| 19 | AI | `GET` | `/api/ai/webpilot` |
| 20 | AI | `GET` | `/api/ai/z-ai` |
| 21 | BSW | `GET` | `/api/bsw/cctv/all` |
| 22 | BSW | `GET` | `/api/bsw/cctv/search` |
| 23 | BSW | `GET` | `/api/bsw/cctv/detail/{id}` |
| 24 | Download | `GET` | `/api/download/aio` |
| 25 | Download | `GET` | `/api/download/bstation` |
| 26 | Download | `GET` | `/api/download/facebook` |
| 27 | Download | `GET` | `/api/download/gdrive` |
| 28 | Download | `GET` | `/api/download/ig-story` |
| 29 | Download | `GET` | `/api/download/instagram` |
| 30 | Download | `GET` | `/api/download/pinterest` |
| 31 | Download | `GET` | `/api/download/scribd` |
| 32 | Download | `GET` | `/api/download/soundcloud` |
| 33 | Download | `GET` | `/api/download/spotify` |
| 34 | Download | `GET` | `/api/download/threads` |
| 35 | Download | `GET` | `/api/download/tiktok` |
| 36 | Download | `GET` | `/api/download/youtube` |
| 37 | Dramabox | `GET` | `/api/dramabox` |
| 38 | Dramabox | `GET` | `/api/dramabox/search` |
| 39 | Image | `GET` | `/api/image/brat` |
| 40 | Image | `GET` | `/api/image/brat/animated` |
| 41 | Image | `GET` | `/api/image/qr` |
| 42 | Info | `GET` | `/api/info/cnn` |
| 43 | Info | `GET` | `/api/info/growagarden` |
| 44 | Info | `GET` | `/api/info/weather` |
| 45 | Komiku | `GET` | `/api/komiku/latest` |
| 46 | Komiku | `GET` | `/api/komiku/chapter/{slug}` |
| 47 | Komiku | `GET` | `/api/komiku/{slug}` |
| 48 | Search | `GET` | `/api/search/bstation` |
| 49 | Search | `GET` | `/api/search/pinterest` |
| 50 | Search | `GET` | `/api/search/cookpad` |
| 51 | Search | `GET` | `/api/search/lyrics` |
| 52 | Search | `GET` | `/api/search/minwall` |
| 53 | Search | `GET` | `/api/search/pddikti` |
| 54 | Search | `GET` | `/api/search/spotify` |
| 55 | Search | `GET` | `/api/search/youtube` |
| 56 | Stalk | `GET` | `/api/stalk/instagram` |
| 57 | Stalk | `GET` | `/api/stalk/ml` |
| 58 | Tools | `GET` | `/api/tools/cek-pajak/jabar` |
| 59 | Tools | `GET` | `/api/tools/cekresi` |
| 60 | Tools | `GET` | `/api/tools/cf-token` |
| 61 | Tools | `GET` | `/api/tools/nsfw-check` |
| 62 | Tools | `GET` | `/api/tools/pln` |
| 63 | Tools | `GET` | `/api/tools/ssweb` |
| 64 | Tools | `GET` | `/api/tools/upscale` |
| 65 | Otakudesu | `GET` | `/api/otakudesu/animebygenre` |
| 66 | Otakudesu | `GET` | `/api/otakudesu/batch/{slug}` |
| 67 | Otakudesu | `GET` | `/api/otakudesu/detail/{slug}` |
| 68 | Otakudesu | `GET` | `/api/otakudesu/episode/{slug}` |
| 69 | Otakudesu | `GET` | `/api/otakudesu/genre` |
| 70 | Otakudesu | `GET` | `/api/otakudesu/getiframe` |
| 71 | Otakudesu | `GET` | `/api/otakudesu/lengkap/{slug}` |
| 72 | Otakudesu | `GET` | `/api/otakudesu/nonce` |
| 73 | Otakudesu | `GET` | `/api/otakudesu` |
| 74 | Otakudesu | `GET` | `/api/otakudesu/schedule` |
| 75 | Otakudesu | `GET` | `/api/otakudesu/search` |
| 76 | lk21 | `GET` | `/api/lk21/episode/{slug}` |
| 77 | lk21 | `GET` | `/api/lk21` |
| 78 | Misc | `GET` | `/api/misc/server-info` |

## Uploader

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `POST` | `/api/upload` | Nexure Uploader | `application/json` | 200, 400, 500 |

### Uploader.1 `POST /api/upload` — Nexure Uploader

| Field | Value |
|---|---|
| Description | Upload file ke NexureCDN dan mengembalikan URL file. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `file` | form-data | `string($binary)` | Ya | Maks. 10MB | File yang akan diunggah. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"name":"0e768d11f42afd00.jpg","original":"example.jpg","type":"image/jpeg","size":62340,"url":"http://api.ammaricano.my.id/file/0e768d11f42afd00.jpg"},"creator":"Nexure Network"}
```

## AI

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/ai/ai4chat` | AI4Chat | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/animagine-xl-3` | Animagine XL 3 | `image/png` | 200, 400, 500 |
| `GET` | `/api/ai/animagine-xl-4` | Animagine XL 4 | `image/png` | 200, 400, 500 |
| `GET` | `/api/ai/claila` | Claila AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/copilot` | Copilot AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/deepimg` | Deepimg | `image/png` | 200, 400, 500 |
| `GET` | `/api/ai/deepseek` | DeepSeek AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/flux-schnell` | Flux Schnell | `image/png` | 200, 400, 500 |
| `GET` | `/api/ai/gemini` | Gemini AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/gpt` | ChatGPT | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/v2/gpt` | ChatGPT V2 | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/groq` | Groq AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/meta` | Meta AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/perplexity` | Perplexity | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/pollinations` | Pollinations AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/pollinations/image` | Pollinations Image | `image/png` | 200, 400, 500 |
| `GET` | `/api/ai/qwen` | Qwen | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/webpilot` | Webpilot AI | `application/json` | 200, 400, 500 |
| `GET` | `/api/ai/z-ai` | Z-AI | `application/json` | 200, 400, 500 |

### AI.1 `GET /api/ai/ai4chat` — AI4Chat

| Field | Value |
|---|---|
| Description | Kirim pertanyaan ke AI4Chat. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"Paris.","creator":"Nexure Network"}
```

### AI.2 `GET /api/ai/animagine-xl-3` — Animagine XL 3

| Field | Value |
|---|---|
| Description | Generate image dari prompt. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `prompt` | query | `string` | Ya | bebas | Prompt gambar. |

**Contoh Response**

```text
Binary image (PNG)
```

### AI.3 `GET /api/ai/animagine-xl-4` — Animagine XL 4

| Field | Value |
|---|---|
| Description | Generate image dari prompt. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `prompt` | query | `string` | Ya | bebas | Prompt gambar. |

**Contoh Response**

```text
Binary image (PNG)
```

### AI.4 `GET /api/ai/claila` — Claila AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Claila AI. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pesan untuk AI. |
| `model` | query | `any` | Tidak | gpt-4.1-mini, gpt-5-mini | Model Claila. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"The capital of France is Paris.","creator":"Nexure Network"}
```

### AI.5 `GET /api/ai/copilot` — Copilot AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Copilot AI. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pesan untuk AI. |
| `model` | query | `any` | Tidak | default, think-deeper, gpt-5 | Mode/model Copilot. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"text":"The capital of France is Paris.","suggestions":{"title":"Paris - Wikipedia","icon":"https://services.bingapis.com/favicon?url=en.wikipedia.org","url":"https://en.wikipedia.org/wiki/Paris"}},"creator":"Nexure Network"}
```

### AI.6 `GET /api/ai/deepimg` — Deepimg

| Field | Value |
|---|---|
| Description | Generate image dari prompt dengan style dan rasio opsional. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `prompt` | query | `string` | Ya | bebas | Prompt gambar. |
| `style` | query | `any` | Tidak | default, ghibli, cyberpunk, anime, portrait, chibi, pixel art, oil painting, 3d | Style gambar. |
| `size` | query | `any` | Tidak | 1:1, 3:2, 2:3 | Rasio gambar. |

**Contoh Response**

```text
Binary image (PNG)
```

### AI.7 `GET /api/ai/deepseek` — DeepSeek AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan DeepSeek, mendukung style, temperature, think, dan session. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pesan untuk AI. |
| `style` | query | `string` | Tidak | bebas | Konteks/style tambahan. |
| `temperature` | query | `any` | Tidak | number | Kontrol kreativitas respons. |
| `think` | query | `boolean` | Tidak | true, false | Aktifkan mode think. |
| `session` | query | `string` | Tidak | bebas | Session ID. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"Ibukota Prancis adalah Paris.","sessionId":"e8d99500d8379341","think":false,"creator":"Nexure Network"}
```

### AI.8 `GET /api/ai/flux-schnell` — Flux Schnell

| Field | Value |
|---|---|
| Description | Generate image dari prompt. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `prompt` | query | `string` | Ya | bebas | Prompt gambar. |

**Contoh Response**

```text
Binary image (PNG)
```

### AI.9 `GET /api/ai/gemini` — Gemini AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Gemini; dapat menerima URL gambar dan session. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |
| `imgUrl` | query | `string` | Tidak | URL gambar | URL gambar opsional. |
| `session` | query | `string` | Tidak | bebas | Session ID. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"Ini adalah gambar seorang gadis anime...","creator":"Nexure Network"}
```

### AI.10 `GET /api/ai/gpt` — ChatGPT

| Field | Value |
|---|---|
| Description | Tanya jawab dengan ChatGPT. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"The capital of France is Paris.","creator":"Nexure Network"}
```

### AI.11 `GET /api/ai/v2/gpt` — ChatGPT V2

| Field | Value |
|---|---|
| Description | Tanya jawab dengan ChatGPT V2. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |
| `style` | query | `string` | Tidak | bebas | Style tambahan. |
| `session` | query | `string` | Tidak | bebas | Session ID. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"text":"Ibu kota Prancis adalah Paris.","sessionId":"gpt-v2-03630a0ad9363ba8"},"creator":"Nexure Network"}
```

### AI.12 `GET /api/ai/groq` — Groq AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Groq AI. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |
| `style` | query | `string` | Tidak | bebas | Style tambahan. |
| `model` | query | `any` | Tidak | moonshotai/kimi-k2-instruct, playai-tts, meta-llama/llama-prompt-guard-2-22m, whisper-large-v3-turbo, llama-3.1-8b-instant, groq/compound, meta-llama/llama-guard-4-12b, openai/gpt-oss-120b, meta-llama/llama-4-maverick-17b-128e-instruct, allam-2-7b, llama-3.3-70b-versatile, moonshotai/kimi-k2-instruct-0905, groq/compound-mini, qwen/qwen3-32b | Model Groq. |
| `session` | query | `string` | Tidak | bebas | Session ID. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"text":"Ibu kota Prancis adalah Paris.","sessionId":"groq-45b3e8db1c39"},"creator":"Nexure Network"}
```

### AI.13 `GET /api/ai/meta` — Meta AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Meta AI. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"The capital of France is Paris.","creator":"Nexure Network"}
```

### AI.14 `GET /api/ai/perplexity` — Perplexity

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Perplexity. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"text":"The capital of France is Paris."},"creator":"Nexure Network"}
```

### AI.15 `GET /api/ai/pollinations` — Pollinations AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Pollinations AI; mendukung input gambar dan session. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |
| `imgUrl` | query | `string` | Tidak | URL gambar | URL gambar opsional. |
| `session` | query | `string` | Tidak | bebas | Session ID. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"Ini adalah gambar seorang gadis anime...","creator":"Nexure Network"}
```

### AI.16 `GET /api/ai/pollinations/image` — Pollinations Image

| Field | Value |
|---|---|
| Description | Generate image dengan Pollinations. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `prompt` | query | `string` | Ya | bebas | Prompt gambar. |
| `model` | query | `any` | Tidak | FLUX, Turbo (AI NSFW), GPTImage, DALL-E 3 (OpenAI), Stability AI | Model generator gambar. |

**Contoh Response**

```text
Binary image (PNG)
```

### AI.17 `GET /api/ai/qwen` — Qwen

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Qwen; mendukung model, type, dan session. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |
| `model` | query | `any` | Tidak | qwen3-max-2025-09-23, qwen3-vl-plus, qwen3-coder-plus, qwen3-vl-32b, qwen3-vl-30b-a3b, qwen3-omni-flash-2025-12-01, qwen-plus-2025-09-11, qwen-plus-2025-07-28, qwen3-30b-a3b, qwen3-coder-30b-a3b-instruct, qwen-max-latest, qwen-plus-2025-01-25, qwq-32b, qwen-turbo-2025-02-11, qwen2.5-omni-7b, qvq-72b-preview-0310, qwen2.5-vl-32b-instruct, qwen2.5-14b-instruct-1m, qwen2.5-coder-32b-instruct, qwen2.5-72b-instruct | Model Qwen. |
| `type` | query | `any` | Tidak | t2t, search | Tipe respons. |
| `session` | query | `string` | Tidak | bebas | Session ID. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"response":{"reasoning":"","content":"The capital of France is Paris.","web_search":[]},"sessionId":"3b5fc012-1102-4d51-a810-226dc75c4934"},"creator":"Nexure Network"}
```

### AI.18 `GET /api/ai/webpilot` — Webpilot AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Webpilot AI. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"text":"Berita terkini di Indonesia...","sources":[]},"creator":"Nexure Network"}
```

### AI.19 `GET /api/ai/z-ai` — Z-AI

| Field | Value |
|---|---|
| Description | Tanya jawab dengan Z-AI. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `ask` | query | `string` | Ya | bebas | Pertanyaan/input teks. |
| `model` | query | `any` | Tidak | glm-4.6, glm-4.6v, glm-4.5v, glm-4.5, glm-4.5-air, glm-4-32b, z1-rumination, z1-32b, chatglm, 0808-360b-dr | Model Z-AI. |
| `search` | query | `boolean` | Tidak | true, false | Aktifkan pencarian. |
| `deepthink` | query | `boolean` | Tidak | true, false | Aktifkan deep think. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"The capital of France is Paris.","creator":"Nexure Network"}
```

## BSW

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/bsw/cctv/all` | BSW CCTV All | `application/json` | 200, 400, 500 |
| `GET` | `/api/bsw/cctv/search` | BSW CCTV Search | `application/json` | 200, 400, 500 |
| `GET` | `/api/bsw/cctv/detail/{id}` | BSW CCTV Detail | `application/json` | 200, 400, 500 |

### BSW.1 `GET /api/bsw/cctv/all` — BSW CCTV All

| Field | Value |
|---|---|
| Description | Ambil semua CCTV dari BSW. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"success":true,"code":200,"result":[{}],"creator":"Nexure Network"}
```

### BSW.2 `GET /api/bsw/cctv/search` — BSW CCTV Search

| Field | Value |
|---|---|
| Description | Cari CCTV dari BSW. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword pencarian CCTV. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{},"creator":"Nexure Network"}
```

### BSW.3 `GET /api/bsw/cctv/detail/{id}` — BSW CCTV Detail

| Field | Value |
|---|---|
| Description | Ambil detail CCTV dan stream URL. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `id` | path | `number` | Ya | ID CCTV | ID CCTV. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{},"creator":"Nexure Network"}
```

## Download

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/download/aio` | Downloader AIO | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/bstation` | Download Bstation | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/facebook` | Download Facebook | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/gdrive` | Download Google Drive | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/ig-story` | Download Instagram Story | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/instagram` | Download Instagram Media | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/pinterest` | Download Pinterest | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/scribd` | Download Scribd | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/soundcloud` | Download SoundCloud | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/spotify` | Download Spotify | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/threads` | Download Threads | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/tiktok` | Download TikTok | `application/json` | 200, 400, 500 |
| `GET` | `/api/download/youtube` | Download YouTube | `application/json` | 200, 400, 500 |

### Download.1 `GET /api/download/aio` — Downloader AIO

| Field | Value |
|---|---|
| Description | Downloader media all-in-one dari berbagai platform. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL media | URL target. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{},"creator":"Nexure Network"}
```

### Download.2 `GET /api/download/bstation` — Download Bstation

| Field | Value |
|---|---|
| Description | Unduh media dari Bstation. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL Bstation | URL media Bstation. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"title":"ATTACK ON TITAN","locate":"id","description":"...","type":"video/mp4","cover":"https://...","views":"820 Ditonton","like":"11","comments":"0","favorites":"Favorit Saya","downloads":"0","media":{"video":[{"quality":"480p","codecs":"avc1.64001F","size":3639371,"mime":"video/mp4","url":"https://..."}],"audio":[{"size":417795,"url":"https://..."}]}},"creator":"ARCH Network"}
```

### Download.3 `GET /api/download/facebook` — Download Facebook

| Field | Value |
|---|---|
| Description | Unduh video Facebook. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL video Facebook | URL video Facebook. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"hd":"https://...","sd":"https://..."},"creator":"Nexure Network"}
```

### Download.4 `GET /api/download/gdrive` — Download Google Drive

| Field | Value |
|---|---|
| Description | Ambil direct download URL file Google Drive. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL Google Drive | URL file Google Drive. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"downloadUrl":"https://drive.usercontent.google.com/download?...","fileName":"armour stone brantford.jpg","fileSize":"210.96 KB","mimetype":"image/jpeg"},"creator":"Nexure Network"}
```

### Download.5 `GET /api/download/ig-story` — Download Instagram Story

| Field | Value |
|---|---|
| Description | Unduh story Instagram berdasarkan URL user. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL/user Instagram | Target story Instagram. |

**Contoh Response**

```json
{"success":true,"code":200,"result":["https://dl.snapcdn.app/download?..."],"creator":"Nexure Network"}
```

### Download.6 `GET /api/download/instagram` — Download Instagram Media

| Field | Value |
|---|---|
| Description | Unduh media Instagram (post/reel/IGTV). |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL post/reel/IGTV | URL media Instagram. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"image":[],"video":[{"url":"https://..."}],"thumbnail":"https://..."},"creator":"Nexure Network"}
```

### Download.7 `GET /api/download/pinterest` — Download Pinterest

| Field | Value |
|---|---|
| Description | Unduh media Pinterest dan metadata pin. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL pin Pinterest | URL pin Pinterest. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"id":"1548181179937932","title":"Nayeon TWICE Official Fanclub ONCE 4th Generation","description":" ","created_at":"Wed, 26 Feb 2025 02:18:03 +0000","dominant_color":"#dbe4e8","link":"string","category":"string","media_urls":[{"type":"image","quality":"original","width":728,"height":1030,"url":"https://i.pinimg.com/originals/...jpg","size":"728x1030"}],"statistics":{"saves":2562,"comments":0,"reactions":{"additionalProp1":369},"total_reactions":0,"views":0},"source":{"name":"Uploaded by user","url":"string"},"board":{"id":"1548249811324972","name":"Guardado rápido","url":"https://pinterest.com/..."},"uploader":{"id":"1548318530367477","username":"esthelamtz35","full_name":"Blanca Esthela Martinez"},"metadata":{"article":"string","product":{"price":"string","currency":"USD"}},"is_promoted":false,"is_downloadable":true,"is_playable":false,"is_repin":true,"is_video":false,"privacy_level":"public","tags":["string"],"hashtags":["#TWICE"],"native_creator":{"username":"rafystarzz"},"sponsor":"string","visual_search_objects":[{}]},"creator":"Nexure Network"}
```

### Download.8 `GET /api/download/scribd` — Download Scribd

| Field | Value |
|---|---|
| Description | Ambil URL unduhan dokumen Scribd. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL dokumen Scribd | URL dokumen. |

**Contoh Response**

```json
{"success":true,"code":200,"result":"https://dlscrib.online/files/TUGAS-KLIPING.pdf","creator":"Nexure Network"}
```

### Download.9 `GET /api/download/soundcloud` — Download SoundCloud

| Field | Value |
|---|---|
| Description | Unduh track SoundCloud. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL track SoundCloud | URL track. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"title":"Juicy Luicy - Lampu Kuning (Dolbie & Zeth Edit)","image":"https://i1.sndcdn.com/...jpg","username":"Dolbie","playbackCount":107029,"likesCount":1437,"commentsCount":7,"displayDate":"2024-06-30T20:29:19Z","url":"https://cf-media.sndcdn.com/...mp3"},"creator":"Nexure Network"}
```

### Download.10 `GET /api/download/spotify` — Download Spotify

| Field | Value |
|---|---|
| Description | Unduh track Spotify. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL track Spotify | URL track. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"title":"Can You Feel the Love Tonight","url":"https://open.spotify.com/track/1EyvonxK8OGHRV4WDRdArc","artist":"Boyce Avenue, Connie Talbot","image":"https://i.scdn.co/image/ab67616d0000b273c340f6f23cf176a5518acb13","duration":"00:04:10","download":"https://cdn-spotify.zm.io.vn/download/..."},"creator":"Nexure Network"}
```

### Download.11 `GET /api/download/threads` — Download Threads

| Field | Value |
|---|---|
| Description | Unduh media dari post Threads. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL post Threads | URL post. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"image_urls":["https://cdn.threadsphotodownloader.com/...jpg"],"video_urls":["string"]},"creator":"Nexure Network"}
```

### Download.12 `GET /api/download/tiktok` — Download TikTok

| Field | Value |
|---|---|
| Description | Unduh video/audio TikTok. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL video TikTok | URL video. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"type":"video","desc":"#siopaolo ...","author":"@sio.maya","authorimg":"https://...","thumbnail":"https://...","media":{"video":"https://...","video_hd":"https://..."},"video_wm":"https://...","audio":"https://...mp3"},"creator":"Nexure Network"}
```

### Download.13 `GET /api/download/youtube` — Download YouTube

| Field | Value |
|---|---|
| Description | Unduh video/audio YouTube. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL video YouTube | URL video. |
| `format` | query | `any` | Tidak | 144, 240, 360, 480, 720, 1080, mp3 | Format/quality unduhan. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"title":"Kita Semua Keturunan Dua Orang Ini","type":"video","format":"720","thumbnail":"https://i.ytimg.com/vi/EGjA4NR9UMc/maxresdefault.jpg","download":"https://cdn300.savetube.su/download-direct/video/720/...","id":"EGjA4NR9UMc","key":"...","duration":236,"quality":"720","downloaded":true},"creator":"Nexure Network"}
```

## Dramabox

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/dramabox` | Dramabox Home | `application/json` | 200 |
| `GET` | `/api/dramabox/search` | Dramabox Search | `application/json` | 200, 400, 500 |

### Dramabox.1 `GET /api/dramabox` — Dramabox Home

| Field | Value |
|---|---|
| Description | Ambil daftar latest dan trending dari Dramabox. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"success":true,"code":200,"latest":[{"title":"string","book_id":"string","image":"string","views":"string","episodes":"string"}],"trending":[{"rank":"string","title":"string","book_id":"string","image":"string"}],"creator":"Nexure Network"}
```

### Dramabox.2 `GET /api/dramabox/search` — Dramabox Search

| Field | Value |
|---|---|
| Description | Cari drama berdasarkan judul. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | judul drama | Keyword pencarian. |

**Contoh Response**

```json
{"success":true,"code":200,"result":[{}],"creator":"Nexure Network"}
```

## Image

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/image/brat` | Brat Image | `image/png` | 200, 400, 500 |
| `GET` | `/api/image/brat/animated` | Brat Animated | `image/gif` | 200, 400, 500 |
| `GET` | `/api/image/qr` | QR Code | `image/png` | 200, 400, 500 |

### Image.1 `GET /api/image/brat` — Brat Image

| Field | Value |
|---|---|
| Description | Generate brat image dari teks. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `text` | query | `string` | Ya | bebas | Teks yang dirender. |

**Contoh Response**

```text
Binary image (PNG)
```

### Image.2 `GET /api/image/brat/animated` — Brat Animated

| Field | Value |
|---|---|
| Description | Generate brat image animasi. |
| Response Type | `image/gif` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `text` | query | `string` | Ya | bebas | Teks yang dirender. |

**Contoh Response**

```text
Binary image (GIF)
```

### Image.3 `GET /api/image/qr` — QR Code

| Field | Value |
|---|---|
| Description | Generate QR code. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `text` | query | `string` | Ya | bebas | Konten QR. |
| `frame` | query | `any` | Tidak | qrcg-scan-me-bottom-frame, qrcg-scan-me-bubble-frame, qrcg-scan-me-bottom-tooltip-frame | Frame QR opsional. |

**Contoh Response**

```text
Binary image (PNG)
```

## Info

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/info/cnn` | CNN Indonesia News | `application/json` | 200, 400, 500 |
| `GET` | `/api/info/growagarden` | Grow a Garden Stock | `application/json` | 200, 400, 500 |
| `GET` | `/api/info/weather` | Weather | `application/json` | 200 |

### Info.1 `GET /api/info/cnn` — CNN Indonesia News

| Field | Value |
|---|---|
| Description | Ambil berita CNN Indonesia. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"success":true,"code":200,"result":[{"title":"Kondisi Aceh Tamiang: Warga Tidur di Jalan, Bau Bangkai Mulai Tercium","url":"https://www.cnnindonesia.com/...","thumb":"https://...jpeg","detail":{"title":"Kondisi Aceh Tamiang...","description":"Kondisi Kabupaten Aceh Tamiang...","image":"https://...jpeg","publishedAt":"Banda Aceh, CNN Indonesia","text":"Kondisi Kabupaten Aceh Tamiang..."}}],"creator":"Nexure Network"}
```

### Info.2 `GET /api/info/growagarden` — Grow a Garden Stock

| Field | Value |
|---|---|
| Description | Ambil info stok Grow a Garden. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"success":true,"code":200,"result":{"type":"initial_data","result":{"seeds":[{"name":"Carrot","quantity":7,"available":true,"category":"SEEDS","type":"seed","roleId":"1376319170052100284","lastUpdated":"2025-09-05T09:45:24.813Z"}]}},"creator":"Nexure Network"}
```

### Info.3 `GET /api/info/weather` — Weather

| Field | Value |
|---|---|
| Description | Ambil info cuaca dari AccuWeather berdasarkan nama kota. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `city` | query | `string` | Ya | nama kota | Kota target. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

## Komiku

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/komiku/latest` | Komiku Latest | `application/json` | 200, 500 |
| `GET` | `/api/komiku/chapter/{slug}` | Komiku Chapter | `application/json` | 200, 404, 500 |
| `GET` | `/api/komiku/{slug}` | Komiku Detail | `application/json` | 200, 400, 500 |

### Komiku.1 `GET /api/komiku/latest` — Komiku Latest

| Field | Value |
|---|---|
| Description | Ambil daftar manga terbaru dengan filter opsional. |
| Response Type | `application/json` |
| HTTP Codes | 200, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `page` | query | `integer` | Tidak | 1..n | Nomor halaman. |
| `search` | query | `string` | Tidak | keyword | Cari manga. |
| `tag` | query | `string` | Tidak | hot, new, trending, dll. | Filter tag. |
| `genre` | query | `string` | Tidak | nama genre | Filter genre. |

**Contoh Response**

```json
{"next_page":"http://localhost:3004/api/komiku/latest?page=2&tag=hot","prev_page":null,"data":[{"title":"Solo Leveling","description":"Sung Jin-Woo, a weak hunter becomes overpowered...","latest_chapter":"Chapter 201","thumbnail":"https://komiku.org/wp-content/uploads/1234.jpg","param":"solo-leveling","detail_url":"http://localhost:3004/api/komiku/solo-leveling"}]}
```

### Komiku.2 `GET /api/komiku/chapter/{slug}` — Komiku Chapter

| Field | Value |
|---|---|
| Description | Ambil semua URL gambar pada chapter manga. |
| Response Type | `application/json` |
| HTTP Codes | 200, 404, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `slug` | path | `string` | Ya | slug chapter | Slug chapter Komiku. |

**Contoh Response**

```json
{"data":["https://cdn.komiku.co.id/12345/chapter1-1.jpg","https://cdn.komiku.co.id/12345/chapter1-2.jpg"]}
```

### Komiku.3 `GET /api/komiku/{slug}` — Komiku Detail

| Field | Value |
|---|---|
| Description | Ambil detail manga, chapter, dan similar manga. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `slug` | path | `string` | Ya | slug manga | Slug manga. |

**Contoh Response**

```json
{"title":"Regressing as the Reincarnated Bastard of the Sword Clan","thumbnail":"https://komiku.id/wp-content/uploads/2023/05/sample.jpg","genre":["Action","Adventure","Fantasy"],"synopsis":"A reincarnated warrior returns to reclaim his lost destiny...","chapters":[{"chapter":"Chapter 12","param":"chapter-12","release":"2 days ago","detail_url":"http://localhost:3004/api/komiku/chapter/chapter-12"}],"similars":[{"title":"Another Similar Manga","thumbnail":"https://komiku.id/wp-content/uploads/2023/04/another.jpg","synopsis":"A tale of a young hero rising once more...","param":"similar-manga-slug","detail_url":"http://localhost:3004/api/komiku/similar-manga-slug"}]}
```

## Search

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/search/bstation` | Search Bstation | `application/json` | 200, 400, 500 |
| `GET` | `/api/search/pinterest` | Search Pinterest | `application/json` | 200, 400, 500 |
| `GET` | `/api/search/cookpad` | Search Cookpad | `application/json` | 200, 400, 500 |
| `GET` | `/api/search/lyrics` | Search Lyrics | `application/json` | 200, 400, 500 |
| `GET` | `/api/search/minwall` | Search Minwall | `application/json` | 200, 400, 500 |
| `GET` | `/api/search/pddikti` | Search PDDIKTI | `application/json` | 200, 400, 500 |
| `GET` | `/api/search/spotify` | Search Spotify | `application/json` | 200, 400, 500 |
| `GET` | `/api/search/youtube` | Search YouTube | `application/json` | 200, 400, 500 |

### Search.1 `GET /api/search/bstation` — Search Bstation

| Field | Value |
|---|---|
| Description | Cari media Bstation. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword pencarian. |

**Contoh Response**

```json
{"success":true,"code":200,"result":[{"title":"ATTACK ON TITAN","locate":"id","description":"string","type":"video/mp4","cover":"string","views":"1.2K Ditonton","like":"string","comments":"string","favorites":"string","downloads":"string","media":{"video":[{"quality":"480p","codecs":"string","size":3639371,"mime":"video/mp4","url":"string"}],"audio":[{"size":417795,"url":"string"}]}}]}
```

### Search.2 `GET /api/search/pinterest` — Search Pinterest

| Field | Value |
|---|---|
| Description | Cari gambar di Pinterest. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword pencarian gambar. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"query":"moon","total":10,"pins":[{"id":"1337074887773653","title":"","description":"How To Paint The Moon For Beginners","pin_url":"https://pinterest.com/pin/1337074887773653","media":{"images":{"orig":{"url":"string"},"small":{"url":"string"},"medium":{"url":"string"},"large":{"url":"string"}},"video":null},"uploader":{"username":"violet0hood","full_name":"Violet","profile_url":"https://pinterest.com/violet0hood"}}]},"creator":"Nexure Network"}
```

### Search.3 `GET /api/search/cookpad` — Search Cookpad

| Field | Value |
|---|---|
| Description | Cari resep Cookpad. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword resep. |

**Contoh Response**

```json
{"success":true,"code":200,"result":[{"id":"15228910","title":"Ayam Kecap","imageUrl":"https://img-global.cpcdn.com/recipes/...jpg","author":"20 menit","prepTime":"Ayam Kecap","servings":"3-4 orang","ingredients":[["dada ayam fillet","potong dadu","air","saus tiram","kecap manis"]],"url":"https://cookpad.com/id/resep/15228910"}],"creator":"Nexure Network"}
```

### Search.4 `GET /api/search/lyrics` — Search Lyrics

| Field | Value |
|---|---|
| Description | Cari lirik lagu. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Judul, artis, atau lirik. |

**Contoh Response**

```json
{"success":true,"code":200,"result":[{"id":0,"name":"string","trackName":"string","artistName":"string","albumName":"string","duration":0,"instrumental":true,"plainLyrics":"string","syncedLyrics":"[00:13.20] Some synced lyric line"}],"creator":"Nexure Network"}
```

### Search.5 `GET /api/search/minwall` — Search Minwall

| Field | Value |
|---|---|
| Description | Cari wallpaper dari Minimal Wallpaper. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword wallpaper. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"wallpapers":[{"id":4861,"isPremium":true,"catId":20,"name":"Mountain","image":"https://minimal.4everwallpaper.in/images/2025-05-15%2016:14:01.forever.jpeg","views":64,"downloads":6,"sets":5,"status":0,"type":"image"}]},"creator":"Nexure Network"}
```

### Search.6 `GET /api/search/pddikti` — Search PDDIKTI

| Field | Value |
|---|---|
| Description | Cari data PDDIKTI berdasarkan jenis pencarian. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | NIM / nama mahasiswa | Keyword pencarian. |
| `type` | query | `string` | Ya | mahasiswa, dosen, prodi, pt | Tipe pencarian. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"mahasiswa":[{"id":"kC-w99K68tuHRKw17CgkbuoW8Nu9vkKhaFoHRMeyMHhktkUSOikzIad-yEzb8aEvu1BtKA==","nama":"HANIF NAUFAL - NAUFAL","nim":"023132026","nama_pt":"UNIVERSITAS TRISAKTI","sinkatan_pt":"","nama_prodi":"AKUNTANSI"}]},"creator":"Nexure Network"}
```

### Search.7 `GET /api/search/spotify` — Search Spotify

| Field | Value |
|---|---|
| Description | Cari lagu/album/playlist/artist di Spotify. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword pencarian. |
| `type` | query | `any` | Tidak | track, album, playlist, artist | Tipe pencarian. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Search.8 `GET /api/search/youtube` — Search YouTube

| Field | Value |
|---|---|
| Description | Cari video atau lagu di YouTube. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword pencarian. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

## Stalk

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/stalk/instagram` | Stalk Instagram | `application/json` | 200 |
| `GET` | `/api/stalk/ml` | Stalk Mobile Legends | `application/json` | Tidak dicantumkan |

### Stalk.1 `GET /api/stalk/instagram` — Stalk Instagram

| Field | Value |
|---|---|
| Description | Ambil profil, story, dan posting terbaru akun Instagram. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `username` | query | `string` | Ya | username Instagram | Username target. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"profile_info":{"username":"ammaricano","full_name":"Ammar | IT enthusiast","biography":"","follower_count":668,"following_count":269,"media_count":1,"profile_pic_url":"https://ig.socialapi-v2.workers.dev/?q=...","profile_pic_url_hd":"https://ig.socialapi-v2.workers.dev/?q=...","is_verified":false,"is_private":false,"external_url":"https://ammaricano.my.id","category":"Software"},"stories":[],"latest_posts":[{"mediaUrls":[{"url":"https://ig.socialapi-v2.workers.dev/?q=...","type":"image"}],"postType":"image","caption":"🌊 I\u0027M READY FOR MPLS 2025 🌊","like_count":40,"comment_count":14,"taken_at_date":"2025-07-12T17:51:01+00:00"}]},"creator":"Nexure Network"}
```

### Stalk.2 `GET /api/stalk/ml` — Stalk Mobile Legends

| Field | Value |
|---|---|
| Description | Ambil informasi user Mobile Legends berdasarkan userId dan zoneId. |
| Response Type | `application/json` |
| HTTP Codes | Tidak dicantumkan |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `userId` | query | `string` | Ya | ML userId | User ID Mobile Legends. |
| `zoneId` | query | `string` | Ya | ML zoneId | Zone ID Mobile Legends. |

**Catatan:** Response example/schema tidak tercantum pada potongan spesifikasi yang diberikan.

**Contoh Response**

```text
Contoh response tidak ditampilkan pada sumber yang diberikan.
```

## Tools

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/tools/cek-pajak/jabar` | Cek Pajak Jabar | `application/json` | 200, 400, 500 |
| `GET` | `/api/tools/cekresi` | Cek Resi | `application/json` | 200, 400, 500 |
| `GET` | `/api/tools/cf-token` | CF Token | `application/json` | 200, 400, 500 |
| `GET` | `/api/tools/nsfw-check` | NSFW Check | `application/json` | 200, 400, 500 |
| `GET` | `/api/tools/pln` | PLN Bill Check | `application/json` | 200, 400, 500 |
| `GET` | `/api/tools/ssweb` | Screenshot Website | `image/png` | 200, 400, 500 |
| `GET` | `/api/tools/upscale` | Upscale Image | `image/png` | 200, 400, 500 |

### Tools.1 `GET /api/tools/cek-pajak/jabar` — Cek Pajak Jabar

| Field | Value |
|---|---|
| Description | Cek status pajak kendaraan Jawa Barat. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `plat` | query | `string` | Ya | nomor polisi | Nomor plat kendaraan. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{},"creator":"Nexure Network"}
```

### Tools.2 `GET /api/tools/cekresi` — Cek Resi

| Field | Value |
|---|---|
| Description | Cek status paket berdasarkan nomor resi dan ekspedisi. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `noresi` | query | `string` | Ya | nomor resi | Nomor resi paket. |
| `ekspedisi` | query | `any` | Tidak | shopee-express, ninja, lion-parcel, pos-indonesia, tiki, acommerce, gtl-goto-logistics, paxel, sap-express, indah-logistik-cargo, lazada-express-lex, lazada-logistics, janio-asia, jet-express, pcp-express, pt-ncs, nss-express, grab-express, rcl-red-carpet-logistics, qrim-express, ark-xpress, standard-express-lwe, luar-negeri-bea-cukai | Kode kurir. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"message":"Selamat, paket Shopee Express (SPX) Anda ... status Delivered.","data":{"resi":"SPXID054330680586","ekspedisi":"Shopee Express (SPX)","ekspedisiCode":"SPX","status":"Delivered","tanggalKirim":"21/06/2025 14:36","customerService":"1500702","lastPosition":"[MH Tebing Tinggi Riau 2 Hub] Your parcel has been delivered [Riduwan] (Delivered)","shareLink":"https://cekresi.com/?r=w&noresi=SPXID054330680586","history":[{"tanggal":"29/06/2025 17:36","keterangan":"[MH Tebing Tinggi Riau 2 Hub] Your parcel has been delivered [Riduwan]"}]}},"creator":"Nexure Network"}
```

### Tools.3 `GET /api/tools/cf-token` — CF Token

| Field | Value |
|---|---|
| Description | Endpoint solver captcha/token sebagaimana tercantum pada spesifikasi sumber. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL target | URL target. |
| `sitekey` | query | `string` | Tidak | mulai dengan 0x4AAA... | Sitekey Turnstile. |

**Catatan:** Didokumentasikan sesuai spesifikasi sumber yang Anda kirim.

**Contoh Response**

```json
{"success":true,"code":200,"result":{"token":"0.Ykx_R9bqlyAyEJ_3R75kBd...","ver":"v2","act":"turnstile-min","elapsed":"9.14s"},"creator":"Nexure Network"}
```

### Tools.4 `GET /api/tools/nsfw-check` — NSFW Check

| Field | Value |
|---|---|
| Description | Cek apakah gambar terdeteksi NSFW. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `imgUrl` | query | `string` | Ya | URL gambar | URL gambar yang dicek. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"isNsfw":false,"labelName":"Not Porn","labelId":"label_n2ian8w116lhxuyk","confidence":0.9806925139692273,"percentage":"98.07%","message":"This image is not detected as NSFW. It is safe to use."},"creator":"Nexure Network"}
```

### Tools.5 `GET /api/tools/pln` — PLN Bill Check

| Field | Value |
|---|---|
| Description | Cek tagihan listrik PLN. |
| Response Type | `application/json` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `id` | query | `integer` | Ya | 12 digit numerik | ID pelanggan PLN. |

**Contoh Response**

```json
{"success":true,"code":200,"result":{"status":"paid","message":"The bill for customer ID 520522604488 has already been paid."},"creator":"Nexure Network"}
```

### Tools.6 `GET /api/tools/ssweb` — Screenshot Website

| Field | Value |
|---|---|
| Description | Ambil screenshot website. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `url` | query | `string` | Ya | URL website | Website target. |

**Contoh Response**

```text
Binary image (PNG)
```

### Tools.7 `GET /api/tools/upscale` — Upscale Image

| Field | Value |
|---|---|
| Description | Upscale gambar dari URL. |
| Response Type | `image/png` |
| HTTP Codes | 200, 400, 500 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `imgUrl` | query | `string` | Ya | URL gambar | Gambar yang di-upscale. |

**Contoh Response**

```text
Binary image (PNG)
```

## Otakudesu

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/otakudesu/animebygenre` | Anime by Genre | `application/json` | 200 |
| `GET` | `/api/otakudesu/batch/{slug}` | Batch Download | `application/json` | 200 |
| `GET` | `/api/otakudesu/detail/{slug}` | Anime Detail | `application/json` | 200 |
| `GET` | `/api/otakudesu/episode/{slug}` | Episode Detail | `application/json` | 200 |
| `GET` | `/api/otakudesu/genre` | Genre List | `application/json` | 200 |
| `GET` | `/api/otakudesu/getiframe` | Get Iframe | `application/json` | 200 |
| `GET` | `/api/otakudesu/lengkap/{slug}` | Complete Download | `application/json` | 200 |
| `GET` | `/api/otakudesu/nonce` | Get Nonce | `application/json` | 200 |
| `GET` | `/api/otakudesu` | Anime List | `application/json` | 200 |
| `GET` | `/api/otakudesu/schedule` | Schedule | `application/json` | 200 |
| `GET` | `/api/otakudesu/search` | Search Anime | `application/json` | 200 |

### Otakudesu.1 `GET /api/otakudesu/animebygenre` — Anime by Genre

| Field | Value |
|---|---|
| Description | Ambil daftar anime berdasarkan genre. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `genre` | query | `string` | Ya | genre anime | Genre anime. |
| `page` | query | `number` | Tidak | 1..n | Nomor halaman. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.2 `GET /api/otakudesu/batch/{slug}` — Batch Download

| Field | Value |
|---|---|
| Description | Ambil link batch download anime. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `slug` | path | `string` | Ya | slug batch | Slug batch. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.3 `GET /api/otakudesu/detail/{slug}` — Anime Detail

| Field | Value |
|---|---|
| Description | Ambil detail anime berdasarkan slug. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `slug` | path | `string` | Ya | slug anime | Slug anime. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.4 `GET /api/otakudesu/episode/{slug}` — Episode Detail

| Field | Value |
|---|---|
| Description | Ambil detail stream dan download episode. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `slug` | path | `string` | Ya | slug episode | Slug episode. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.5 `GET /api/otakudesu/genre` — Genre List

| Field | Value |
|---|---|
| Description | Ambil daftar genre anime. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.6 `GET /api/otakudesu/getiframe` — Get Iframe

| Field | Value |
|---|---|
| Description | Ambil data iframe 360p/480p/720p. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `content` | query | `string` | Ya | base64-encoded content | Konten yang sudah di-base64. |
| `nonce` | query | `string` | Ya | token nonce | Nonce/token. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.7 `GET /api/otakudesu/lengkap/{slug}` — Complete Download

| Field | Value |
|---|---|
| Description | Ambil link download anime lengkap. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `slug` | path | `string` | Ya | slug lengkap | Slug complete download. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.8 `GET /api/otakudesu/nonce` — Get Nonce

| Field | Value |
|---|---|
| Description | Ambil nonce token dari situs. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.9 `GET /api/otakudesu` — Anime List

| Field | Value |
|---|---|
| Description | Ambil daftar anime ongoing/complete. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `type` | query | `any` | Tidak | ongoing, complete | Tipe daftar anime. |
| `page` | query | `number` | Tidak | 1..n | Nomor halaman. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.10 `GET /api/otakudesu/schedule` — Schedule

| Field | Value |
|---|---|
| Description | Ambil jadwal rilis anime. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### Otakudesu.11 `GET /api/otakudesu/search` — Search Anime

| Field | Value |
|---|---|
| Description | Cari anime berdasarkan query. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `query` | query | `string` | Ya | bebas | Keyword pencarian anime. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

## lk21

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/lk21/episode/{slug}` | LK21 Episode Detail | `application/json` | 200 |
| `GET` | `/api/lk21` | LK21 Film List | `application/json` | 200 |

### lk21.1 `GET /api/lk21/episode/{slug}` — LK21 Episode Detail

| Field | Value |
|---|---|
| Description | Ambil detail stream dan download film/episode berdasarkan slug. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `slug` | path | `string` | Ya | slug episode | Slug episode. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

### lk21.2 `GET /api/lk21` — LK21 Film List

| Field | Value |
|---|---|
| Description | Ambil daftar film latest/populer/series. |
| Response Type | `application/json` |
| HTTP Codes | 200 |

**Parameter**

| Nama | Lokasi | Tipe | Wajib | Available Values | Keterangan |
|---|---|---|---|---|---|
| `type` | query | `any` | Tidak | latest, top-movie-today, latest-series, populer | Tipe daftar film. |
| `page` | query | `number` | Tidak | 1..n | Nomor halaman. |

**Contoh Response**

```json
{"success":true,"code":200,"creator":"Nexure Network"}
```

## Misc

| Method | Endpoint | Judul | Response Type | HTTP Codes |
|---|---|---|---|---|
| `GET` | `/api/misc/server-info` | Server Info | `application/json` | 200, 500 |

### Misc.1 `GET /api/misc/server-info` — Server Info

| Field | Value |
|---|---|
| Description | Ambil informasi server. |
| Response Type | `application/json` |
| HTTP Codes | 200, 500 |

**Parameter**

_Tidak ada parameter._

**Contoh Response**

```json
{"os":{"platform":"string","release":"string","type":"string"},"cpu":{"model":"string","cores":0},"ram":{"totalMemory":"string","freeMemory":"string"},"storage":{"drive":"string","free":"string","total":"string"}}
```

## Catatan Tambahan

- Mayoritas endpoint JSON mengikuti envelope umum berikut:

```json
{
  "success": true,
  "code": 200,
  "result": {},
  "creator": "Nexure Network"
}
```

- Beberapa endpoint mengembalikan **binary response** seperti `image/png` atau `image/gif`.
- Beberapa schema pada spesifikasi sumber memang sangat longgar, misalnya `result: {}` atau hanya menampilkan `success`, `code`, dan `creator`.
- Endpoint `/api/stalk/ml` tidak menampilkan contoh response pada potongan spesifikasi yang Anda berikan, jadi bagian itu saya tandai apa adanya.
