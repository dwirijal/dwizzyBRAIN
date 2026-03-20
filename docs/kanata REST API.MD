# Kanata REST API

## Metadata

| Field | Value |
|---|---|
| API Name | KanataAPI |
| Version | 2.1.0 |
| Spec | OAS 3.0 |
| OpenAPI Document | `/openapi.json` |
| License | MIT |
| Primary Server | `https://api.kanata.web.id` |
| Production Server | `https://api.kanata.web.id` |
| Notes | API modular berbasis Elixir/Phoenix yang mengintegrasikan scraper media, tools, AI image, BMKG, Islami, TV schedule, dataset pemerintah, uploader, dan utilitas lainnya. |

## Overview

Dokumentasi ini disusun dari spesifikasi yang Anda berikan untuk **KanataAPI**. Formatnya dibuat sama seperti dokumentasi Nexure: mencakup **endpoint, method, parameter, available values, request body, response type, response code, dan contoh code/schema bila tersedia**.

### Important Notes

- Total endpoint yang teridentifikasi dari source: **91 endpoint**.
- Banyak endpoint pada source **tidak menampilkan contoh response/schema lengkap**. Pada kasus seperti itu, dokumentasi ini **tidak mengarang schema**, dan akan menandai response sebagai:
  - `JSON (schema not shown in supplied excerpt)`, atau
  - `Binary image (direct file; exact media type not shown in supplied excerpt)`.
- Endpoint yang secara eksplisit menampilkan request body / media type di source sudah didokumentasikan apa adanya.
- Ada heading **Game** pada source, tetapi **tidak ada endpoint** yang ditampilkan di bawah heading tersebut.

## Category Summary

| Category | Endpoint Count |
|---|---|
| AI | 2 |
| Anichin | 7 |
| Animasu | 8 |
| BMKG | 6 |
| Downloader | 8 |
| GoogleTranslate | 1 |
| Government | 2 |
| Islami | 7 |
| KBBI | 1 |
| Komiku | 7 |
| MovieTube | 5 |
| News | 3 |
| NontonFilm | 3 |
| Otakudesu | 12 |
| TV | 4 |
| TempMail | 3 |
| Tools | 4 |
| Upload | 1 |
| YouTube | 7 |

## Endpoint Index

| No. | Category | Method | Endpoint | Summary |
|---:|---|---|---|---|
| 1 | Downloader | `GET` | `/instagram/fetch` | Fetch Instagram media |
| 2 | Downloader | `GET` | `/threads/fetch` | Fetch Threads data |
| 3 | Downloader | `GET` | `/mediafire/fetch` | Fetch Mediafire download link |
| 4 | Downloader | `GET` | `/pinterest/fetch` | Pinterest Downloader |
| 5 | Downloader | `GET` | `/tiktok2` | Fetch TikTok video data (Method 2 - Fallback) |
| 6 | Downloader | `GET` | `/tiktok/fetch` | Fetch TikTok video data (Method 1) |
| 7 | Downloader | `GET` | `/facebook/fetch` | Fetch Facebook video |
| 8 | Downloader | `GET` | `/reddit/fetch` | Fetch Reddit video |
| 9 | BMKG | `GET` | `/bmkg/cuaca/desa` | Prakiraan Cuaca (Desa/Kelurahan) |
| 10 | BMKG | `GET` | `/bmkg/gempa` | Info Gempa Bumi |
| 11 | BMKG | `GET` | `/bmkg/cuaca/provinces` | List Provinsi Cuaca |
| 12 | BMKG | `GET` | `/bmkg/cuaca` | Prakiraan Cuaca (Provinsi) |
| 13 | BMKG | `GET` | `/bmkg/gempa/dirasakan` | Info Gempa Dirasakan |
| 14 | BMKG | `GET` | `/bmkg/wilayah/search` | Cari Kode Wilayah |
| 15 | Islami | `GET` | `/sholat/kota/{nama}` | Cari ID Kota/Kabupaten |
| 16 | Islami | `GET` | `/khutbah/detail` | Detail Khutbah |
| 17 | Islami | `GET` | `/quran/surat` | Daftar Surah Al-Qur'an |
| 18 | Islami | `GET` | `/sholat/jadwal/{id_kota}` | Dapatkan jadwal sholat |
| 19 | Islami | `GET` | `/hadits/{collection}/{number}` | Ambil data Hadits |
| 20 | Islami | `GET` | `/quran/surat/{nomor}` | Detail Surah (Ayat-ayat) |
| 21 | Islami | `GET` | `/khutbah/list` | Daftar Khutbah |
| 22 | NontonFilm | `GET` | `/nontonfilm/stream` | Get Streaming URL |
| 23 | NontonFilm | `GET` | `/nontonfilm/search` | Search Movies |
| 24 | NontonFilm | `GET` | `/nontonfilm` | Get Latest Movies |
| 25 | Animasu | `GET` | `/animasu/home` | Get Animasu Home Data |
| 26 | Animasu | `GET` | `/animasu/search` | Search Anime |
| 27 | Animasu | `GET` | `/animasu/genres` | Get All Genres |
| 28 | Animasu | `GET` | `/animasu/anime/{slug}` | Get Anime Detail |
| 29 | Animasu | `GET` | `/animasu/schedule` | Get Animasu Release Schedule |
| 30 | Animasu | `GET` | `/animasu/genres/{slug}` | Get Anime by Genre |
| 31 | Animasu | `GET` | `/animasu/movies` | Get Anime Movies |
| 32 | Animasu | `GET` | `/animasu/episode/{slug}` | Get Episode Embed Link |
| 33 | Otakudesu | `GET` | `/otakudesu/episode/{slug}` | Get Episode Detail |
| 34 | Otakudesu | `GET` | `/otakudesu/genres/{slug}` | Get Anime by Genre |
| 35 | Otakudesu | `GET` | `/otakudesu/schedule` | Get Release Schedule |
| 36 | Otakudesu | `GET` | `/otakudesu/anime-list` | Get Full Anime List (A-Z) |
| 37 | Otakudesu | `GET` | `/otakudesu/ongoing` | Get Ongoing Anime List |
| 38 | Otakudesu | `GET` | `/otakudesu/anime/{slug}` | Get Anime Detail |
| 39 | Otakudesu | `GET` | `/otakudesu/home` | Get Otakudesu Home Data |
| 40 | Otakudesu | `POST` | `/otakudesu/stream` | Get Stream via Mirror Payload |
| 41 | Otakudesu | `GET` | `/otakudesu/search` | Search Anime |
| 42 | Otakudesu | `GET` | `/otakudesu/genres` | Get All Genres |
| 43 | Otakudesu | `GET` | `/otakudesu/batch/{slug}` | Get Batch Download Links |
| 44 | Otakudesu | `GET` | `/otakudesu/complete` | Get Complete Anime List |
| 45 | Anichin | `GET` | `/anichin/episode/{slug}` | Get Episode Detail |
| 46 | Anichin | `GET` | `/anichin/ongoing` | Get Ongoing Donghua List |
| 47 | Anichin | `GET` | `/anichin/schedule` | Get Release Schedule |
| 48 | Anichin | `GET` | `/anichin/completed` | Get Completed Donghua List |
| 49 | Anichin | `GET` | `/anichin/detail/{slug}` | Get Donghua Detail |
| 50 | Anichin | `GET` | `/anichin/search` | Search Donghua |
| 51 | Anichin | `GET` | `/anichin/home` | Get Anichin Home Data |
| 52 | Tools | `GET` | `/ipinfo/{ip}` | IP Lookup (ipinfo.io) |
| 53 | Tools | `GET` | `/checkhost/info` | IP Info (Check-Host) |
| 54 | Tools | `GET` | `/nik/lookup` | NIK Lookup (Decoder) |
| 55 | Tools | `POST` | `/carbon` | Generate Carbon Image |
| 56 | AI | `GET` | `/ai/image` | Generate gambar (Direct Image) |
| 57 | AI | `GET` | `/ai/generate` | Generate gambar (JSON) |
| 58 | News | `GET` | `/news/top` | Dapatkan berita utama |
| 59 | News | `GET` | `/news/category/{category}` | Dapatkan berita per kategori |
| 60 | News | `GET` | `/news/search` | Cari berita |
| 61 | YouTube | `GET` | `/youtube2/info` | Get video metadata (Combo/Fallback) |
| 62 | YouTube | `GET` | `/youtube/download` | Download Video MP4 (Combo/Fallback) |
| 63 | YouTube | `GET` | `/youtube/download-audio` | Download Audio MP3 (Combo/Fallback) |
| 64 | YouTube | `GET` | `/youtube2/download-audio` | Download Audio MP3 (Combo/Fallback) |
| 65 | YouTube | `GET` | `/youtube/info` | Get video metadata (Combo/Fallback) |
| 66 | YouTube | `GET` | `/savetube` | Download Audio YouTube via SaveTube |
| 67 | YouTube | `GET` | `/youtube2/download` | Download Video MP4 (Combo/Fallback) |
| 68 | TV | `GET` | `/tv/schedule/{channel_id}` | Jadwal TV per Channel |
| 69 | TV | `GET` | `/tv/now` | Sedang Tayang |
| 70 | TV | `GET` | `/tv/channels` | List Channel TV |
| 71 | TV | `GET` | `/tv/bola` | Jadwal Bola |
| 72 | Government | `GET` | `/satudata/search` | Cari dataset pemerintah (Satu Data Indonesia) |
| 73 | Government | `GET` | `/satudata/dataset/{id}` | Detail dataset pemerintah |
| 74 | Komiku | `GET` | `/komiku/manga/{slug}` | Get Manga Detail |
| 75 | Komiku | `GET` | `/komiku/home` | Get Komiku Home Data |
| 76 | Komiku | `GET` | `/komiku/pustaka` | Get Komiku Pustaka (Latest/Type) |
| 77 | Komiku | `GET` | `/komiku/hot` | Get Hot Manga |
| 78 | Komiku | `GET` | `/komiku/chapter/{slug}` | Get Chapter Images |
| 79 | Komiku | `GET` | `/komiku/search` | Search Manga |
| 80 | Komiku | `GET` | `/komiku/list` | Get Manga List (Daftar Komik) |
| 81 | MovieTube | `GET` | `/movietube/stream` | Get Movie Embed Link |
| 82 | MovieTube | `GET` | `/movietube/detail/{slug}` | Get Movie Detail |
| 83 | MovieTube | `GET` | `/movietube/search` | Search Movies |
| 84 | MovieTube | `GET` | `/movietube/genre/{name}` | Get Movies by Genre |
| 85 | MovieTube | `GET` | `/movietube/home` | Get Home Section Movies |
| 86 | TempMail | `GET` | `/tempmail/messages/{email}` | Dapatkan daftar pesan |
| 87 | TempMail | `GET` | `/tempmail/read/{email}/{id}` | Baca pesan spesifik |
| 88 | TempMail | `GET` | `/tempmail/create` | Buat email sementara baru |
| 89 | KBBI | `GET` | `/kbbi` | Cari arti kata di KBBI |
| 90 | Upload | `POST` | `/upload` | Upload file |
| 91 | GoogleTranslate | `GET` | `/googletranslate` | Terjemahkan teks |

---

## Downloader

### GET `/instagram/fetch` — Fetch Instagram media

Extract download links for Photos, Videos, and Reels from Instagram using IQSaved

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/instagram/fetch` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | Instagram Media URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Instagram media data and download links |
| `400` | Invalid URL or failed to scrape |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/threads/fetch` — Fetch Threads data

Extract metadata and video URL from a Threads post

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/threads/fetch` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | Threads Post URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Threads metadata and media links |
| `400` | Invalid URL or failed to scrape |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/mediafire/fetch` — Fetch Mediafire download link

Extract direct download link from a Mediafire URL

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/mediafire/fetch` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | Mediafire URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Mediafire file data and download link |
| `400` | Invalid URL or failed to scrape |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/pinterest/fetch` — Pinterest Downloader

Mendapatkan link download gambar atau video dari Pinterest

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/pinterest/fetch` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | URL Pin Pinterest |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Media info |
| `400` | Invalid request |
| `500` | Internal Server Error |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/tiktok2` — Fetch TikTok video data (Method 2 - Fallback)

Scrape TikTok video metadata and download links using tikwm.com

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tiktok2` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | TikTok Video URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | TikTok metadata and download links |
| `400` | Invalid URL or scraping failed |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/tiktok/fetch` — Fetch TikTok video data (Method 1)

Scrape TikTok video metadata and download links using tiktokdl.app

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tiktok/fetch` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | TikTok Video URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | TikTok metadata and download links |
| `400` | Invalid URL or scraping failed |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/facebook/fetch` — Fetch Facebook video

Extract video download links from a Facebook post

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/facebook/fetch` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | Facebook Video URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Facebook video download link |
| `400` | Invalid URL or failed to scrape |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/reddit/fetch` — Fetch Reddit video

Extract video download links from a Reddit post using Rapidsave

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/reddit/fetch` |
| Category | Downloader |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | Reddit Post URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Reddit video download link |
| `400` | Invalid URL or failed to scrape |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## BMKG

### GET `/bmkg/cuaca/desa` — Prakiraan Cuaca (Desa/Kelurahan)

Dapatkan prakiraan cuaca sangat detail berdasarkan kode wilayah ADM4 (Kelurahan/Desa)

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/bmkg/cuaca/desa` |
| Category | BMKG |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `adm4` | `query` | `string` | Yes | Kode ADM4 (contoh: 31.71.03.1001) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Data cuaca desa |
| `404` | Kode wilayah tidak ditemukan |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/bmkg/gempa` — Info Gempa Bumi

Dapatkan informasi gempa bumi terbaru (M 5.0+) dan daftar gempa terkini dari BMKG

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/bmkg/gempa` |
| Category | BMKG |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Data gempa |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/bmkg/cuaca/provinces` — List Provinsi Cuaca

Dapatkan daftar kunci provinsi yang tersedia untuk pengecekan cuaca

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/bmkg/cuaca/provinces` |
| Category | BMKG |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Daftar kunci provinsi |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/bmkg/cuaca` — Prakiraan Cuaca (Provinsi)

Dapatkan prakiraan cuaca detail per kota dalam sebuah provinsi (Kabupaten/Kota)

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/bmkg/cuaca` |
| Category | BMKG |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `provinsi` | `query` | `string` | Yes | Kunci provinsi (contoh: jakarta, jawa_barat) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Data cuaca detail |
| `404` | Provinsi tidak ditemukan |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/bmkg/gempa/dirasakan` — Info Gempa Dirasakan

Dapatkan daftar gempa bumi yang dirasakan terbaru dari BMKG

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/bmkg/gempa/dirasakan` |
| Category | BMKG |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Data gempa dirasakan |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/bmkg/wilayah/search` — Cari Kode Wilayah

Cari kode ADM4 untuk Kelurahan/Desa menggunakan database pihak ketiga

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/bmkg/wilayah/search` |
| Category | BMKG |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Nama Kelurahan/Desa |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Daftar wilayah ditemukan |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Islami

### GET `/sholat/kota/{nama}` — Cari ID Kota/Kabupaten

Cari ID kota/kabupaten untuk jadwal sholat

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/sholat/kota/{nama}` |
| Category | Islami |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `nama` | `path` | `string` | Yes | Nama kota/kabupaten |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of cities |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/khutbah/detail` — Detail Khutbah

Mendapatkan isi teks khutbah lengkap berdasarkan URL

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/khutbah/detail` |
| Category | Islami |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | URL Khutbah (dari list) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Sermon detail |
| `404` | Sermon not found |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/quran/surat` — Daftar Surah Al-Qur'an

Mendapatkan daftar seluruh surah dalam Al-Qur'an

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/quran/surat` |
| Category | Islami |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of surahs |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/sholat/jadwal/{id_kota}` — Dapatkan jadwal sholat

Mengambil jadwal sholat berdasarkan ID kota

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/sholat/jadwal/{id_kota}` |
| Category | Islami |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `id_kota` | `path` | `string` | Yes | ID Kota (dari hasil cari kota) |
| `date` | `query` | `string` | No | Tanggal (YYYY-MM-DD, default hari ini) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Prayer schedule |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/hadits/{collection}/{number}` — Ambil data Hadits

Mendapatkan data hadits dari hadits.id berdasarkan koleksi dan nomor

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/hadits/{collection}/{number}` |
| Category | Islami |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `collection` | `path` | `string` | Yes | Koleksi hadits (e.g., bukhari, muslim, abudawud, tirmidzi, nasai, ibnumajah, malik) |
| `number` | `path` | `integer` | Yes | Nomor hadits |

#### Available Values

| Field | Available Values |
|---|---|
| `collection` | bukhari, muslim, abudawud, tirmidzi, nasai, ibnumajah, malik |

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Hadits data |
| `404` | Hadits not found |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/quran/surat/{nomor}` — Detail Surah (Ayat-ayat)

Mendapatkan isi surah beserta ayat, terjemahan, dan audio

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/quran/surat/{nomor}` |
| Category | Islami |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `nomor` | `path` | `integer` | Yes | Nomor surah (1-114) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Surah details |
| `404` | Surah not found |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/khutbah/list` — Daftar Khutbah

Mendapatkan daftar ribuan materi khutbah dari berbagai kategori

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/khutbah/list` |
| Category | Islami |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of sermons |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## NontonFilm

### GET `/nontonfilm/stream` — Get Streaming URL

Extract iframe streaming URL from Post ID or Movie URL

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/nontonfilm/stream` |
| Category | NontonFilm |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `id` | `query` | `string` | Yes | Post ID or Full Movie URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Streaming iframe data |
| `400` | Failed to determine Post ID or fetch stream |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/nontonfilm/search` — Search Movies

Search movie titles in NontonFilm

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/nontonfilm/search` |
| Category | NontonFilm |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Movie title keyword |
| `page` | `query` | `integer` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Search results |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/nontonfilm` — Get Latest Movies

Get latest movie list from NontonFilm

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/nontonfilm` |
| Category | NontonFilm |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `page` | `query` | `integer` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of latest movies |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Animasu

### GET `/animasu/home` — Get Animasu Home Data

Retrieve home page data from Animasu

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/home` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Ongoing and updated anime |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/animasu/search` — Search Anime

Search anime by keyword

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/search` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Search keyword |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Search results |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/animasu/genres` — Get All Genres

Retrieve all anime genres from Animasu

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/genres` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of all anime genres |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/animasu/anime/{slug}` — Get Anime Detail

Retrieve anime details and episode list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/anime/{slug}` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Anime slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Anime details and episode list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/animasu/schedule` — Get Animasu Release Schedule

Retrieve anime release schedule grouped by day

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/schedule` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Anime release schedule grouped by day |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/animasu/genres/{slug}` — Get Anime by Genre

Retrieve anime list for a specific genre

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/genres/{slug}` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Genre slug |
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Anime list for specific genre |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/animasu/movies` — Get Anime Movies

Retrieve anime movie list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/movies` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of anime movies |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/animasu/episode/{slug}` — Get Episode Embed Link

Retrieve embed URL for an anime episode

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/animasu/episode/{slug}` |
| Category | Animasu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Episode slug or full URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Embed URL |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Otakudesu

### GET `/otakudesu/episode/{slug}` — Get Episode Detail

Retrieve episode details

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/episode/{slug}` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Episode slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Episode details |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/genres/{slug}` — Get Anime by Genre

Retrieve anime list by genre

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/genres/{slug}` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Genre slug (e.g., action, romance) |
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Anime list by genre |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/schedule` — Get Release Schedule

Retrieve release schedule

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/schedule` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Release schedule |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/anime-list` — Get Full Anime List (A-Z)

Retrieve complete anime list grouped by letter

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/anime-list` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Complete list of anime grouped by letter |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/ongoing` — Get Ongoing Anime List

Retrieve ongoing anime list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/ongoing` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Ongoing anime list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/anime/{slug}` — Get Anime Detail

Retrieve anime details

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/anime/{slug}` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Anime slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Anime details |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/home` — Get Otakudesu Home Data

Retrieve home page data

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/home` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Home data |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### POST `/otakudesu/stream` — Get Stream via Mirror Payload

Send mirror payload to retrieve stream iframe URL (requires nonce internally)

| Field | Value |
|---|---|
| Method | `POST` |
| Endpoint | `/otakudesu/stream` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | application/json |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Request Body

- Content-Type: `application/json`
- Schema: `{}`

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Stream data |

#### Example Request Body

```json
{}
```

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/search` — Search Anime

Search anime by query

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/search` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Query keyword |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Search results |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/genres` — Get All Genres

Retrieve full genre list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/genres` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Full genre list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/batch/{slug}` — Get Batch Download Links

Retrieve batch download data

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/batch/{slug}` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Batch slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Batch download data |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/otakudesu/complete` — Get Complete Anime List

Retrieve complete anime list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/otakudesu/complete` |
| Category | Otakudesu |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Complete anime list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Anichin

### GET `/anichin/episode/{slug}` — Get Episode Detail

Retrieve episode details

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/anichin/episode/{slug}` |
| Category | Anichin |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Episode slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Episode details |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/anichin/ongoing` — Get Ongoing Donghua List

Retrieve ongoing list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/anichin/ongoing` |
| Category | Anichin |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Ongoing list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/anichin/schedule` — Get Release Schedule

Retrieve release schedule

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/anichin/schedule` |
| Category | Anichin |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Release schedule |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/anichin/completed` — Get Completed Donghua List

Retrieve completed list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/anichin/completed` |
| Category | Anichin |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Completed list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/anichin/detail/{slug}` — Get Donghua Detail

Retrieve donghua details

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/anichin/detail/{slug}` |
| Category | Anichin |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Donghua slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Donghua details |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/anichin/search` — Search Donghua

Search donghua by query

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/anichin/search` |
| Category | Anichin |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Query keyword |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Search results |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/anichin/home` — Get Anichin Home Data

Retrieve home page data

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/anichin/home` |
| Category | Anichin |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Home data |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Tools

### GET `/ipinfo/{ip}` — IP Lookup (ipinfo.io)

Dapatkan informasi geolocation dan ISP untuk sebuah IP address menggunakan ipinfo.io

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/ipinfo/{ip}` |
| Category | Tools |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `ip` | `path` | `string` | Yes | IP Address |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Detail informasi IP |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/checkhost/info` — IP Info (Check-Host)

Dapatkan informasi detail mengenai IP address atau hostname menggunakan Check-Host.net

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/checkhost/info` |
| Category | Tools |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `host` | `query` | `string` | Yes | IP Address atau Hostname |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Detail informasi IP |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/nik/lookup` — NIK Lookup (Decoder)

Dekode informasi dari Nomor Induk Kependudukan (Provinsi, Kota, Kecamatan, Tgl Lahir, Gender)

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/nik/lookup` |
| Category | Tools |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `nik` | `query` | `string` | Yes | 16 digit NIK |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Informasi NIK |
| `400` | NIK tidak valid |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### POST `/carbon` — Generate Carbon Image

Mengubah potongan kode menjadi gambar cantik menggunakan JSON body (POST).

| Field | Value |
|---|---|
| Method | `POST` |
| Endpoint | `/carbon` |
| Category | Tools |
| Response Type | image/png |
| Request Body | application/json |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Request Body

- Content-Type: `application/json`
- Schema: `{
  "bg": "#abb8c3",
  "code": "def hello do\n  IO.puts \"world\"\nend",
  "lang": "elixir",
  "theme": "Dracula"
}`

#### Response Codes

| Code | Description |
|---:|---|
| `200` | PNG Image |
| `400` | Bad Request |
| `500` | Internal Server Error |

#### Example Request Body

```json
{
  "bg": "#abb8c3",
  "code": "def hello do\n  IO.puts \"world\"\nend",
  "lang": "elixir",
  "theme": "Dracula"
}
```

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## AI

### GET `/ai/image` — Generate gambar (Direct Image)

Menghasilkan dan mengembalikan file gambar secara langsung

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/ai/image` |
| Category | AI |
| Response Type | Binary image (direct file; exact media type not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `prompt` | `query` | `string` | Yes | Deskripsi gambar |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | File image |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/ai/generate` — Generate gambar (JSON)

Menghasilkan URL gambar menggunakan Pollinations.ai

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/ai/generate` |
| Category | AI |
| Response Type | JSON (URL gambar; schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `prompt` | `query` | `string` | Yes | Deskripsi gambar |
| `width` | `query` | `integer` | No | Lebar gambar (default: 1024) |
| `height` | `query` | `integer` | No | Tinggi gambar (default: 1024) |
| `seed` | `query` | `integer` | No | Random seed |
| `model` | `query` | `string` | No | Model (misal: 'flux' atau 'turbo') |
| `nologo` | `query` | `boolean` | No | Hilangkan logo/watermark |

#### Available Values

| Field | Available Values |
|---|---|
| `nologo` | true, false |

#### Response Codes

| Code | Description |
|---:|---|
| `200` | URL gambar yang dihasilkan |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## News

### GET `/news/top` — Dapatkan berita utama

Dapatkan berita utama (Top Stories) Indonesia terbaru

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/news/top` |
| Category | News |
| Response Type | JSON array/object (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Array of news items |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/news/category/{category}` — Dapatkan berita per kategori

Retrieve news by category

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/news/category/{category}` |
| Category | News |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `category` | `path` | `string` | Yes | Nama kategori (nasional, internasional, dll) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Category news |
| `404` | Category not found |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/news/search` — Cari berita

Search news by keyword

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/news/search` |
| Category | News |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Kata kunci pencarian |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Search results |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## YouTube

### GET `/youtube2/info` — Get video metadata (Combo/Fallback)

Mendapatkan metadata video dengan fallback otomatis (Local yt-dlp -> SaveTube)

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/youtube2/info` |
| Category | YouTube |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | YouTube Video URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Video metadata |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/youtube/download` — Download Video MP4 (Combo/Fallback)

Download video dengan strategi fallback: YTConvert -> Local yt-dlp

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/youtube/download` |
| Category | YouTube |
| Response Type | JSON (download link; schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | YouTube Video URL |
| `quality` | `query` | `string` | No | Video quality (e.g., 720, 1080) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Download link |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/youtube/download-audio` — Download Audio MP3 (Combo/Fallback)

Download audio dengan strategi fallback: SaveTube -> YTConvert -> Local yt-dlp

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/youtube/download-audio` |
| Category | YouTube |
| Response Type | JSON (download link; schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | YouTube Video URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Download link |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/youtube2/download-audio` — Download Audio MP3 (Combo/Fallback)

Download audio dengan strategi fallback: SaveTube -> YTConvert -> Local yt-dlp

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/youtube2/download-audio` |
| Category | YouTube |
| Response Type | JSON (download link; schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | YouTube Video URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Download link |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/youtube/info` — Get video metadata (Combo/Fallback)

Mendapatkan metadata video dengan fallback otomatis (Local yt-dlp -> SaveTube)

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/youtube/info` |
| Category | YouTube |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | YouTube Video URL |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Video metadata |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/savetube` — Download Audio YouTube via SaveTube

Mengambil link download audio YouTube menggunakan layanan savetube.me

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/savetube` |
| Category | YouTube |
| Response Type | JSON (download info; schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | URL YouTube |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Download info |
| `400` | Invalid URL |
| `500` | Internal Server Error |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/youtube2/download` — Download Video MP4 (Combo/Fallback)

Download video dengan strategi fallback: YTConvert -> Local yt-dlp

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/youtube2/download` |
| Category | YouTube |
| Response Type | JSON (download link; schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `url` | `query` | `string` | Yes | YouTube Video URL |
| `quality` | `query` | `string` | No | Video quality (e.g., 720, 1080) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Download link |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## TV

### GET `/tv/schedule/{channel_id}` — Jadwal TV per Channel

Retrieve TV schedule for a specific channel

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tv/schedule/{channel_id}` |
| Category | TV |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `channel_id` | `path` | `string` | Yes | ID Channel (contoh: rcti, antv) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Jadwal channel |
| `404` | Channel tidak ditemukan |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/tv/now` — Sedang Tayang

Dapatkan daftar acara yang sedang tayang saat ini di berbagai stasiun TV

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tv/now` |
| Category | TV |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Daftar acara sedang tayang |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/tv/channels` — List Channel TV

Dapatkan daftar semua channel TV yang tersedia

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tv/channels` |
| Category | TV |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Daftar channel |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/tv/bola` — Jadwal Bola

Dapatkan jadwal siaran langsung sepakbola terbaru

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tv/bola` |
| Category | TV |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Jadwal bola |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Government

### GET `/satudata/search` — Cari dataset pemerintah (Satu Data Indonesia)

Pencarian dataset publik dari portal pemerintah data.go.id

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/satudata/search` |
| Category | Government |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Kata kunci pencarian |
| `page` | `query` | `integer` | No | Nomor halaman (default: 1) |
| `limit` | `query` | `integer` | No | Jumlah data per halaman max 50 (default: 10) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Hasil pencarian dataset |
| `500` | Gagal mengambil data dari upstream |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/satudata/dataset/{id}` — Detail dataset pemerintah

Detail 1 dataset dari data.go.id berdasarkan dataset id atau name

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/satudata/dataset/{id}` |
| Category | Government |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `id` | `path` | `string` | Yes | ID atau name dataset |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Detail dataset |
| `404` | Dataset tidak ditemukan |
| `500` | Gagal mengambil data dari upstream |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Komiku

### GET `/komiku/manga/{slug}` — Get Manga Detail

Retrieve manga details

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/komiku/manga/{slug}` |
| Category | Komiku |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Manga slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Manga detail |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/komiku/home` — Get Komiku Home Data

Retrieve Komiku home page data

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/komiku/home` |
| Category | Komiku |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Home data |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/komiku/pustaka` — Get Komiku Pustaka (Latest/Type)

Retrieve Komiku pustaka list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/komiku/pustaka` |
| Category | Komiku |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `tipe` | `query` | `string` | No | manga, manhwa, or manhua |
| `page` | `query` | `string` | No | Page number |

#### Available Values

| Field | Available Values |
|---|---|
| `tipe` | manga, manhwa, manhua |

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Pustaka list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/komiku/hot` — Get Hot Manga

Retrieve hot manga list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/komiku/hot` |
| Category | Komiku |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `page` | `query` | `string` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Hot manga list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/komiku/chapter/{slug}` — Get Chapter Images

Retrieve chapter images

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/komiku/chapter/{slug}` |
| Category | Komiku |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Chapter slug |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Chapter images |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/komiku/search` — Search Manga

Search manga by query

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/komiku/search` |
| Category | Komiku |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Search query |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Search results |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/komiku/list` — Get Manga List (Daftar Komik)

Retrieve manga list

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/komiku/list` |
| Category | Komiku |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `tipe` | `query` | `string` | No | manga, manhwa, or manhua |
| `huruf` | `query` | `string` | No | Starting letter |
| `page` | `query` | `string` | No | Page number |

#### Available Values

| Field | Available Values |
|---|---|
| `tipe` | manga, manhwa, manhua |

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Manga list |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## MovieTube

### GET `/movietube/stream` — Get Movie Embed Link

Retrieve embed stream URL

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/movietube/stream` |
| Category | MovieTube |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `id` | `query` | `string` | Yes | Movie/Episode slug |
| `type` | `query` | `string` | No | Type (movie or series) |

#### Available Values

| Field | Available Values |
|---|---|
| `type` | movie, series |

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Embed stream URL |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/movietube/detail/{slug}` — Get Movie Detail

Retrieve movie details and stream URL

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/movietube/detail/{slug}` |
| Category | MovieTube |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `slug` | `path` | `string` | Yes | Movie slug |
| `type` | `query` | `string` | No | Type (movie or series) |

#### Available Values

| Field | Available Values |
|---|---|
| `type` | movie, series |

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Movie details and stream URL |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/movietube/search` — Search Movies

Search movies by query

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/movietube/search` |
| Category | MovieTube |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Search query |
| `page` | `query` | `integer` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Search results |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/movietube/genre/{name}` — Get Movies by Genre

Retrieve movies filtered by genre

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/movietube/genre/{name}` |
| Category | MovieTube |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `name` | `path` | `string` | Yes | Genre name |
| `page` | `query` | `integer` | No | Page number |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Filtered movies |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/movietube/home` — Get Home Section Movies

Retrieve movie list by home section

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/movietube/home` |
| Category | MovieTube |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `section` | `query` | `string` | No | Section name (popular, latest, trending) |

#### Available Values

| Field | Available Values |
|---|---|
| `section` | popular, latest, trending |

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of movies |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## TempMail

### GET `/tempmail/messages/{email}` — Dapatkan daftar pesan

Retrieve message list for a temp mail inbox

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tempmail/messages/{email}` |
| Category | TempMail |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `email` | `path` | `string` | Yes | Alamat email |
| `password` | `query` | `string` | Yes | Password email |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | List of messages |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/tempmail/read/{email}/{id}` — Baca pesan spesifik

Read a specific temp mail message

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tempmail/read/{email}/{id}` |
| Category | TempMail |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `email` | `path` | `string` | Yes | Alamat email |
| `id` | `path` | `string` | Yes | ID Pesan |
| `password` | `query` | `string` | Yes | Password email |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Message content |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

### GET `/tempmail/create` — Buat email sementara baru

Menghasilkan alamat email acak baru dari mail.tm

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/tempmail/create` |
| Category | TempMail |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Email address and password generated |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## KBBI

### GET `/kbbi` — Cari arti kata di KBBI

Mencari definisi kata dari Kamus Besar Bahasa Indonesia (via kbbi.web.id)

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/kbbi` |
| Category | KBBI |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `q` | `query` | `string` | Yes | Kata yang dicari |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Definisi kata |
| `404` | Kata tidak ditemukan |
| `500` | Internal server error |

> Example response/schema was not provided in the supplied KanataAPI excerpt.


---

## Upload

### POST `/upload` — Upload file

Upload file (Gambar, Video, Dokumen) ke server. Ekstensi yang diizinkan: .jpg, .jpeg, .png, .gif, .webp, .mp4, .mov, .avi, .pdf, .txt, .zip

| Field | Value |
|---|---|
| Method | `POST` |
| Endpoint | `/upload` |
| Category | Upload |
| Response Type | application/json |
| Request Body | multipart/form-data |

#### Parameters

_No parameters._

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Request Body

- Content-Type: `multipart/form-data`
- Schema: `file: string($binary)`

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Berhasil upload |
| `400` | File tidak valid atau ekstensi dilarang |
| `500` | Gagal menyimpan file |

#### Notes

- Allowed extensions: .jpg, .jpeg, .png, .gif, .webp, .mp4, .mov, .avi, .pdf, .txt, .zip

#### Example Response

```json
{
  "content_type": "string",
  "filename": "string",
  "message": "string",
  "original_filename": "string",
  "url": "string"
}
```


---

## GoogleTranslate

### GET `/googletranslate` — Terjemahkan teks

Menerjemahkan teks menggunakan Google Translate (Unofficial)

| Field | Value |
|---|---|
| Method | `GET` |
| Endpoint | `/googletranslate` |
| Category | GoogleTranslate |
| Response Type | JSON (schema not shown in supplied excerpt) |
| Request Body | None |

#### Parameters

| Name | In | Type | Required | Description |
|---|---|---|---|---|
| `text` | `query` | `string` | Yes | Teks yang akan diterjemahkan |
| `to` | `query` | `string` | Yes | Bahasa tujuan (kode ISO, misal: id, en, ja) |
| `from` | `query` | `string` | No | Bahasa asal (default: auto) |

#### Available Values

_No enumerated available values specified in the supplied excerpt._

#### Response Codes

| Code | Description |
|---:|---|
| `200` | Translation result |
| `500` | Translation failed |

> Example response/schema was not provided in the supplied KanataAPI excerpt.

---

## Game

_No endpoints were listed under the **Game** heading in the supplied source._
