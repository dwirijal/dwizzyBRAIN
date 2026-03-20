
# YTDLP API

Dokumentasi ringkas dan lengkap berdasarkan cuplikan OpenAPI yang Anda kirim.

## Informasi Umum

| Item | Nilai |
|---|---|
| Nama API | YTDLP API |
| Versi | `4.0.0` |
| OpenAPI | `OAS 3.1` |
| OpenAPI JSON | `/openapi.json` |
| Deskripsi | YouTube dan Spotify Downloader API |
| Catatan proyek | `simple yt dl by nauval` |
| Base URL yang terlihat pada contoh | `https://ytdlpyton.nvlgroup.my.id` |
| Total endpoint pada cuplikan | **99 endpoint** |

## Catatan Umum

- Hampir semua endpoint menerima header opsional `X-API-Key` dengan tipe `string | null`.
- Sebagian besar endpoint pada cuplikan mendefinisikan response `200` sebagai `application/json` dengan schema sederhana `"string"`.
- Banyak endpoint FastAPI di cuplikan memiliki response error validasi standar `422` dengan schema `HTTPValidationError`.
- Beberapa endpoint **hanya menampilkan judul endpoint** tanpa detail parameter/response yang lengkap. Endpoint seperti ini saya tandai sebagai **detail tidak ditampilkan pada cuplikan**.
- Saya **tidak mengarang schema** yang tidak muncul pada source.

## Schema Error Validasi Umum (`422`)

```json
{
  "detail": [
    {
      "loc": ["string", 0],
      "msg": "string",
      "type": "string",
      "input": "string",
      "ctx": {}
    }
  ]
}
```

---

## Default

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/` | Root Endpoint. Menampilkan `index.html`. | Header: `X-API-Key` *(optional, string \| null)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/buyrole` | Buy Role Page. Menampilkan `buy.html`. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/apidocs` | API docs. Menampilkan halaman API docs v3. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/download/file/{filename}` | Mengunduh file hasil. | Path: `filename` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## YouTube

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/search/` | Pencarian Video YouTube dengan Thumbnail. | Query: `query` *(required, string)* — kata kunci pencarian<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/info/` | Informasi lengkap video/playlist YouTube. | Query: `url` *(required, string)*<br>Query: `limit` *(integer)*<br>Header: `X-API-Key` *(optional)* | `limit` default `50`, maksimal `50` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/download/` | Unduhan video YouTube (non-blocking). | Query: `url` *(required, string)*<br>Query: `resolution` *(integer)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `resolution` default `720`<br>`mode` default `url` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/download/ytindo` | Unduhan video YouTube via Proxy Indonesia. **Deprecated**. | Query: `url` *(required, string)*<br>Query: `resolution` *(integer)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `resolution` default `720`<br>`mode` default `url` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/download/ytsub` | Unduh video dengan subtitle digabung (non-blocking). | Query: `url` *(required, string)*<br>Query: `resolution` *(integer)*<br>Query: `lang` *(string)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `resolution` default `720`<br>`lang` default `id`<br>`mode` default `url` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/download/ytpost` | Ambil gambar kualitas tinggi dari post komunitas YouTube. | Query: `url` *(required, string)* — URL post komunitas<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `mode` = `url` / `buffer`<br>default `url` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/download/audio` | Download audio. | Query: `url` *(required, string)*<br>Query: `mode` *(string)*<br>Query: `bitrate` *(string)*<br>Header: `X-API-Key` *(optional)* | `mode` default `url`<br>`bitrate` default `128k` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/download/playlist` | Unduh playlist YouTube (video+audio atau audio saja, tanpa FFmpeg). | Query: `url` *(required, string)*<br>Query: `resolution` *(string)*<br>Query: `max_videos` *(integer)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `resolution` contoh `360`, `720`, atau `audio`<br>`max_videos` default `10`, maksimal `10`<br>`mode` = `url` / `buffer`, default `url` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Spotify

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/spotify/search` | Cari lagu di Spotify (dengan client ID). | Query: `query` *(required, string)* — judul lagu atau artis<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/spotify/info` | Info lengkap Spotify URL (track/album/playlist/show/radio). | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | URL dapat berupa track, album, playlist, show, radio | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/spotify/download/audio` | Unduh audio dari Spotify track/show/episode (via YouTube). | **Detail parameter/response tidak ditampilkan pada cuplikan.** | - | - | **Tidak ditampilkan pada cuplikan.** |
| GET | `/spotify/download/playlist` | Unduh playlist Spotify jadi MP3 (via YouTube). | Query: `url` *(required, string)*<br>Query: `limit` *(integer)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `limit` `1–50`, default `10`<br>`mode` saat ini hanya `url` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/spotify/fullplaylist` | Unduh full playlist Spotify (MP3) dengan opsi ZIP/GDrive. | Query: `url` *(required, string)*<br>Query: `limit` *(integer)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `limit` default `10`<br>`mode` = `url`, `zip`<br>default `zip` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Downloader

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/douyin` | Download video atau foto dari Douyin (TikTok China). | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/tiktok` | Download video TikTok (via Tikwm API). | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/threads/download` | Threads Download Route. | Query: `url` *(required, string)* — link Threads yang ingin didownload<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/downloader/soundcloud` | Download SoundCloud. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/downloader/soundcloud/playlist` | Download SoundCloud Playlist. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/sfile` | Download file dari `sfile.mobi`. | Query: `url` *(required, string)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `mode` = `url` / `buffer`<br>default `url` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/downloader/ssyoutube` | Download Ssyoutube. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/downloader/mediafire` | Download MediaFire. | Query: `url` *(required, string)* — URL Mediafire<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/Instagram` | Download Instagram. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/downloader/igstory` | Instagram Story. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/downloader/tiktokhd` | TikTok HD Downloader. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/shopee/video` | Get Shopee Video Metadata. | Query: `url` *(required, string)* — link Shopee Video<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/nhentai` | Download metadata dan gambar dari nhentai. | Query: `link` *(required, string)* — URL galeri nhentai<br>Header: `X-API-Key` *(optional)* | contoh: `https://nhentai.net/g/573590/` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/aplemusic` | Download dari `aplmusicdownloader.net`. | Query: `url` *(required, string)* — URL lagu Apple Music<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/facebook` | Facebook Downloader. | Query: `url` *(required, string)* — URL Facebook video<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Stats

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/stats` | Statistik Server. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Pinterest

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/pinterest/search` | Cari Pin di Pinterest. | **Detail parameter/response tidak ditampilkan pada cuplikan.** | - | - | **Tidak ditampilkan pada cuplikan.** |
| GET | `/pinterest/download` | Download media dari Pin Pinterest. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Utility

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/jarak` | Hitung jarak antar kota via `distancecalculator.net`. | Query: `dari` *(required, string)*<br>Query: `ke` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/screenshot` | Ambil screenshot halaman web. | Query: `url` *(required, string)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `mode` default `desktop` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/tobase64` | Konversi file ke Base64. | Header: `X-API-Key` *(optional)* | - | **Schema section only**: object `{ "file": "binary" }` (`application/octet-stream`) | **Detail response code tidak ditampilkan pada cuplikan.** |
| GET | `/cekpln` | Cek Tagihan PLN. | Query: `id` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/removebg` | Hapus background dari gambar. | Query: `image_url` *(string, optional jika pakai file)*<br>Query: `mode` *(string)*<br>Header: `X-API-Key` *(optional)* | `mode` = `file` / `url`<br>default `file` | `multipart/form-data` → field `file` *(string/file, optional jika pakai URL)* | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/shorturl` | Persingkat URL (internal). | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/s/{code}` | Redirect dari short URL. | **Detail parameter/response tidak ditampilkan pada cuplikan.** | - | - | **Tidak ditampilkan pada cuplikan.** |
| GET | `/gsmarena` | Gsmarena Specs. | Query: `q` *(required, string)* — nama HP<br>Header: `X-API-Key` *(optional)* | contoh: `iPhone 15 Pro`, `Galaxy S24` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/kurs` | Cek nilai tukar mata uang. | Query: `dari` *(string)*<br>Query: `ke` *(string)*<br>Query: `jumlah` *(number)*<br>Header: `X-API-Key` *(optional)* | `dari` default `USD`<br>`ke` default `IDR`<br>`jumlah` default `1` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/nsfw/check` | Cek NSFW dari gambar via URL. | Query: `url` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/subdofinder` | Subdofinder. | Query: `domain` *(required, string)*<br>Header: `X-API-Key` *(optional)* | contoh: `example.com` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/utility/upscale` | Upscale gambar via Pixelcut. | Header: `X-API-Key` *(optional)* | - | **Schema section only**: object `{ "image": "binary" }` (`application/octet-stream`) | **Detail response code tidak ditampilkan pada cuplikan.** |
| GET | `/listbank` | List Bank. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/cekbank` | Cek Bank. | Query: `bank_code` *(required, string)*<br>Query: `account_number` *(required, string)*<br>Header: `X-API-Key` *(optional)* | contoh bank code: `dana` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## AI

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/ai/powerbrain` | Chat dengan PowerBrain AI. | Query: `question` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/ai/felo` | Cari jawaban dari Felo AI. | Query: `text` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/ai/beago` | Chat dengan Beago AI. | Query: `text` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/blackbox` | Chat completion via Blackbox.ai. | Query: `text` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/ai/imagen-exoml` | Generate gambar dari Imagen Exomlapi. | Query: `prompt` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/ai/gemini` | Gemini Chat API. | Query: `text` *(string)*<br>Query: `model` *(string)*<br>Header: `X-API-Key` *(optional)* | Nilai model tidak dirinci pada cuplikan | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/ai/deepseek` | Deepseek Chat. | Header: `X-API-Key` *(optional)* | - | `application/json` → `{ "input": "string", "session_id": "string", "think": false }` | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/ai/arting` | Generate Arting Image. | Query: `prompt` *(required, string)*<br>Query: `negative_prompt` *(string, optional)*<br>Header: `X-API-Key` *(optional)* | `negative_prompt` default string kosong | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/ai/deepai-chat` | DeepAI Chat. | Query: `prompt` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/sunoai` | Generate Suno. | Query: `prompt` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Topup

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/topup/roles` | List semua role, harga, dan limit. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/topup/createkupon` | Create Kupon. | Query: `keykhusus` *(required, string)*<br>Query: `nama` *(required, string)*<br>Query: `tipe` *(required, string)*<br>Query: `jumlah` *(required, integer)*<br>Query: `maks` *(required, integer)*<br>Header: `X-API-Key` *(optional)* | Nilai `tipe` tidak dirinci | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/topup/qris` | Topup Qris. | Query: `ip` *(required, string)*<br>Query: `role` *(required, string)*<br>Query: `wa` *(required, string)*<br>Query: `idpay` *(string)*<br>Query: `kupon` *(string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/topup/check/{idpay}` | Check Payment. | Path: `idpay` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/topup/otpvc` | Kirim OTP ke WA admin. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/topup/createvoucher` | Buat kode voucher manual. | Query: `otp` *(required, string)*<br>Query: `role` *(required, string)*<br>Query: `days` *(required, integer)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/topup/claimvoucher/{voucher}` | Claim Voucher. | Path: `voucher` *(required, string)*<br>Query: `ip` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/topup/upgrade-role` | Upgrade/Downgrade role berdasarkan IP dan ID pembayaran. | Query: `ip` *(required, string)*<br>Query: `role_lama` *(required, string)*<br>Query: `role_baru` *(required, string)*<br>Query: `idpay` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/role/check` | Check user role based on IP address. | Query: `ip` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/checkme` | Cek role, batasan, dan waktu kadaluarsa berdasarkan IP/API key. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"` *(cuplikan real response ada, lihat contoh di bawah)*<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/topup/change-ip` | Ganti IP berdasarkan ID pembayaran & IP lama. | Query: `ip_lama` *(required, string)*<br>Query: `ip_baru` *(required, string)*<br>Query: `idpay` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

### Contoh Response Nyata `/checkme`

```json
{
  "auth_by": "ip",
  "auth_value": "182.8.66.216",
  "role": "petualang_gratis",
  "expired": null,
  "expired_in": null,
  "limits": {
    "resolution": 720,
    "max_size_mb": 100,
    "rpm": 10
  }
}
```

---

## PDDIKTI

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/pddikti` | Cari dan dekripsi data dari PDDIKTI. | Query: `q` *(required, string)*<br>Query: `tipe` *(string)* — filter `mahasiswa`, `prodi`, `dosen`, dll<br>Header: `X-API-Key` *(optional)* | Nilai `tipe` tidak dibatasi eksplisit pada cuplikan | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Anime

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/donghua` | Donghua Handler. | Query: `action` *(required, string)*<br>Query: `q` *(required, string)*<br>Header: `X-API-Key` *(optional)* | `action` = `search` \| `episode` \| `video` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Stalker

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/ml/stalk` | Stalk ML. | Query: `user_id` *(required, string)*<br>Query: `zone_id` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Islami

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/quran` | Ambil data ayah atau surah dari API Al-Qur'an. | Query: `ayah` *(string)*<br>Query: `surah` *(integer)*<br>Query: `edition` *(string)*<br>Header: `X-API-Key` *(optional)* | `edition` default `quran-simple` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/tafsir` | Cari Tafsir Al-Qur'an dalam Bahasa Indonesia. | Query: `surah` *(required, integer)*<br>Header: `X-API-Key` *(optional)* | contoh: `1` untuk Al-Fatihah | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/hadits` | Cari Hadits dari kitab yang dipilih. | Query: `book` *(string)*<br>Query: `hadith_id` *(integer)*<br>Header: `X-API-Key` *(optional)* | `book` default `bukhari`<br>`hadith_id` default `1` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/topegon` | Ubah tulisan latin menjadi pegon. | Query: `text` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Quran

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/quran/surah` | API All Surah. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/quran/surah/{number}` | API Surah By Number. | Path: `number` *(required, integer)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/quran/surah/byname` | API Surah By Name. | Query: `name` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/quran/audio` | Get Audio URL. | Query: `qori_id` *(required, integer)*<br>Query: `surah_id` *(required, integer)*<br>Query: `verse_id` *(required, integer)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Maker

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/maker/brat` | Generate gambar teks BRAT dari `bratgenerator.com`. | Query: `text` *(required, string)*<br>Query: `background` *(string)*<br>Query: `color` *(string)*<br>Header: `X-API-Key` *(optional)* | `background` dan `color` berupa warna contoh `#000000` / `#FFFFFF` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/maker/bratvid` | Buat video animasi teks BRAT 500x440 secara async. | Query: `text` *(required, string)*<br>Query: `background` *(string)*<br>Query: `color` *(string)*<br>Header: `X-API-Key` *(optional)* | warna contoh `#000000`, `#FFFFFF` | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/maker/iqc` | Buat gambar iPhone fake-chat. | Query: `text` *(required, string)*<br>Query: `user` *(string)*<br>Query: `jam` *(string)*<br>Query: `profile_url` *(string)*<br>Header: `X-API-Key` *(optional)* | `user` default `Anonymous`<br>`jam` default WIB sekarang | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## GrowAGarden

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| GET | `/growagarden/crops` | Get Crops. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/growagarden/pets` | Get Pets. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/growagarden/gear` | Get Gear. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/growagarden/eggs` | Get Eggs. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/growagarden/cosmetics` | Get Cosmetics. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/growagarden/events` | Get Events. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/growagarden/stock` | Stock. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Payment Gateway (`pg`)

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| POST | `/pg/register` | Daftarkan user Payment Gateway (hanya admin, kirim WA ke admin & user, branding Midtrans). | Query: `keyadmin` *(required, string)*<br>Query: `username` *(required, string)*<br>Query: `wa` *(required, string)*<br>Query: `email` *(string \| null)*<br>Query: `pin` *(required, string)*<br>Header: `X-API-Key` *(optional)* | `pin` = PIN 4 digit | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/pg/create` | PG Create. | Query: `username` *(required, string)*<br>Query: `nama_tagihan` *(required, string)*<br>Query: `nama_barang` *(required, string)*<br>Query: `nominal` *(required, integer)*<br>Query: `wa` *(required, string)*<br>Query: `email` *(string \| null)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/pg/cek/{idpay}` | PG Cek. | Path: `idpay` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| GET | `/pg/me` | PG Me. Menampilkan info user PG. | Query: `username` *(required, string)*<br>Query: `pin` *(required, string)*<br>Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/pg/withdraw` | PG Withdraw. Tarik saldo cair dari transaksi PG. | Query: `username` *(required, string)*<br>Query: `pin` *(required, string)*<br>Query: `metode` *(required, string)*<br>Query: `nomor_tujuan` *(required, string)*<br>Header: `X-API-Key` *(optional)* | biaya admin Rp 3.000, minimal saldo 13.000 | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/pg/acc/wd` | PG Acc Wd. | **Detail parameter/response tidak ditampilkan pada cuplikan.** | - | - | **Tidak ditampilkan pada cuplikan.** |

---

## Payment

| Method | Endpoint | Deskripsi | Parameter | Available values / default | Request body | Response type & code |
|---|---|---|---|---|---|---|
| POST | `/mustika/create` | Mustika Create Pay. Membuat pembayaran baru via Mustika Payment. | Query: `user` *(required, string)*<br>Query: `amount` *(required, integer)*<br>Header: `X-API-Key` *(optional)* | `amount` dalam IDR | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |
| POST | `/callback` | Mustika Callback. Endpoint callback untuk menerima notifikasi dari Mustika Payment. | Header: `X-API-Key` *(optional)* | - | - | `200` `application/json` → `"string"`<br>`422` `application/json` → `HTTPValidationError` |

---

## Lampiran Schema Request Body dari Bagian `Schemas`

### `Body_deepseek_chat_ai_deepseek_post`

```json
{
  "input": "string",
  "session_id": "string",
  "think": false
}
```

### `Body_remove_bg_removebg_post`

```json
{
  "file": "binary"
}
```

### `Body_to_base64_tobase64_post`

```json
{
  "file": "binary"
}
```

### `Body_upscale_image_utility_upscale_post`

```json
{
  "image": "binary"
}
```

## Endpoint yang Detailnya Tidak Lengkap pada Cuplikan

Berikut endpoint yang muncul pada source, tetapi detail parameter/response-nya tidak lengkap pada cuplikan yang Anda kirim:

- `GET /spotify/download/audio`
- `GET /pinterest/search`
- `POST /tobase64`
- `GET /s/{code}`
- `POST /utility/upscale`
- `POST /pg/acc/wd`

## Ringkasan

- Total endpoint yang teridentifikasi dari cuplikan: **99 endpoint**
- OpenAPI version: **3.1**
- Banyak endpoint hanya mengembalikan schema generik `"string"` pada dokumentasi Swagger yang diberikan.
- Semua bagian yang tidak tampil pada source saya tandai apa adanya agar dokumentasi tetap akurat.
