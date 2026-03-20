# dwizzyOS Market Unified Architecture

March 2026

Dokumen ini menjadi source of truth untuk arsitektur `/v1/market/*` dwizzyOS. Dokumen ini menggantikan:

- `docs/dwizzyOS_market_architecture.docx.md`
- `docs/dwizzyOS_market_expansion.docx.md`
- `docs/dwizzyOS_market_expansion_v2.docx.md`

Fokus utamanya adalah arsitektur market yang konsisten dengan implementasi `dwizzyBRAIN` saat ini: hybrid transport, symbol resolution, multi-exchange aggregation, OHLCV pipeline, dan arbitrage signal engine.

## 1. Prinsip Dasar

Sistem market data dibangun dengan lima prinsip:

1. `coin_id` menjadi kunci universal lintas seluruh sistem.
2. Frontend menerima response yang sudah di-merge server-side, bukan merakit sendiri dari banyak source.
3. Availability dan freshness harus eksplisit, bukan implicit.
4. Jalur realtime tidak boleh terlalu tergantung pada abstraction yang belum cukup stabil.
5. Transport layer boleh berbeda, tetapi normalisasi schema di atasnya harus tetap satu.

## 2. Keputusan Transport

Keputusan utama untuk transport market data:

- Native WebSocket adalah jalur utama untuk exchange hot-path dan latency-sensitive.
- CCXT REST adalah jalur standar untuk fallback, backfill, dan exchange minor.
- CCXT WebSocket / Pro hanya dipakai selektif sebagai adapter tambahan, bukan pengganti default native WS.

Implementasi awal:

| Exchange | Mode utama | Peran |
|---|---|---|
| Binance | Native WS + REST | Hot ticker, order book, canonical OHLCV |
| Bybit | Native WS + REST | Hot backup dan arbitrage coverage |
| OKX | Native WS + REST | Hot coverage tambahan |
| Gate.io | CCXT REST | Minor exchange coverage |
| KuCoin | CCXT REST | Minor exchange coverage |
| Kraken | CCXT REST | Regional spread coverage |
| MEXC | CCXT REST | Long-tail listing coverage |
| HTX | CCXT REST | Additional Asian liquidity coverage |

Alasan keputusan ini:

- Native WS lebih predictable untuk Binance, Bybit, dan OKX.
- CCXT REST menyederhanakan integrasi exchange minor tanpa menambah banyak maintenance cost.
- CCXT Go websocket/pro masih diperlakukan sebagai opsi bertahap per exchange, bukan fondasi realtime utama.

## 3. Three-Tier Market Pipeline

### 3.1 Cold Tier

Dipakai untuk metadata dan data yang tidak perlu realtime.

- Source utama: CoinGecko
- Refresh: 24 jam
- Storage: PostgreSQL
- Data: metadata coin, image, links, categories, ATH/ATL, rank, cold enrichment fields

### 3.2 Warm Tier

Dipakai untuk data yang cukup segar tetapi tidak perlu push realtime.

- Source utama: Binance REST dan CCXT REST
- Source fallback: CoinPaprika, DexScreener bila relevan
- Refresh: 5-10 menit untuk snapshot umum, lebih cepat untuk subset strategis
- Storage: Valkey dengan TTL
- Data: ticker summary, market cap updates, 24h volume, intermediate snapshots

### 3.3 Hot Tier

Dipakai untuk ticker dan order book yang sensitif terhadap latency.

- Source utama: native WS major exchanges
- Storage cepat: Valkey
- Storage historis: TimescaleDB untuk tabel tertentu
- Reconnect policy mengikuti batas exchange, misalnya Binance direfresh sebelum batas 24 jam

## 4. Symbol Resolution

Semua source exchange wajib di-resolve ke `coin_id` sebelum masuk ke layer agregasi dan API.

Canonical flow:

1. Raw payload datang dari WS atau REST.
2. Resolver mencari `exchange + exchange_symbol -> coin_id`.
3. Jika ketemu, sistem menghasilkan ticker/candle yang sudah ternormalisasi.
4. Jika tidak ketemu, simbol dicatat ke `unknown_symbols` dan tidak dibuang diam-diam.

Komponen:

- `coin_exchange_mappings`
- `unknown_symbols`
- Valkey cache untuk forward dan reverse resolution
- auto-build dan validator job untuk menjaga mapping tetap sehat

Aturan penting:

- `coin_id -> exchange_symbol` dipakai saat outbound query dan backfill.
- `exchange + raw_symbol -> coin_id` dipakai saat inbound ingestion.
- Quote priority default: `USDT -> USDC -> BUSD -> BTC`
- Edge case seperti rebrand, fork, delisting, dan DEX-only token ditangani eksplisit lewat mapping state

## 5. Coverage Tiers

Tidak semua coin memiliki coverage yang sama. Sistem membagi coin berdasarkan rank dan availability:

| Tier | Coverage | Mode |
|---|---|---|
| A | Coin besar, tersedia di major CEX | Hot + Warm + Cold |
| B | Coin menengah, tersedia parsial | Warm + Cold |
| C | Coin kecil / long tail | Cold + selective fallback |
| D | Sangat rendah prioritas | Metadata only / on-demand |

Tier ini mengendalikan:

- apakah coin ikut scheduler hot path
- apakah OHLCV di-backfill
- apakah arbitrage discan
- kapan enrichment dijalankan on-demand

## 6. Core Services

### 6.1 Ticker Ingestion

- Native WS client menerima raw ticker major exchange
- CCXT manager mengambil snapshot ticker exchange minor
- Ingestion service melakukan symbol resolution
- Publisher menulis:
  - `price:{symbol}:{exchange}`
  - `price:{coin_id}:{exchange}`
- Aggregator membentuk tampilan ticker lintas exchange per coin

### 6.2 OHLCV Service

OHLCV mengikuti prinsip berikut:

- Binance menjadi canonical exchange default untuk chart utama
- Exchange lain tetap tersedia sebagai opsi
- Backfill dan incremental sync dilakukan via REST
- Candle disimpan ke TimescaleDB
- Candle baru dipublish ke Valkey agar quant worker bisa mengonsumsi stream yang sama

### 6.3 Order Book Service

- Major exchange: native WS
- Minor exchange: polling terukur via CCXT REST
- Snapshot ditulis ke Valkey dengan TTL berbeda sesuai mode transport

### 6.4 Ticker Aggregator

Aggregator membangun view unified per coin:

- best bid
- best ask
- last price per exchange
- spread percentage
- volume comparison
- availability map

### 6.5 Arbitrage Engine

Arbitrage engine adalah signal layer, bukan execution bot.

- Scan interval: 5 detik
- Scope awal: top coin likuid
- Rule minimum: spread, net spread, depth, cooldown, availability
- Output:
  - simpan ke PostgreSQL
  - alert ke Discord

## 7. Storage Strategy

### 7.1 PostgreSQL

Dipakai untuk:

- metadata coin
- coin exchange mappings
- unknown symbols
- arbitrage configuration
- arbitrage signals

### 7.2 TimescaleDB

Dipakai untuk data historis time-series:

- `ohlcv`
- `exchange_spread_history`
- tabel time-series lain yang bernilai untuk analisa

Retention dan compression diterapkan per use case agar cost tetap terkendali.

### 7.3 Valkey

Dipakai untuk hot cache dan PubSub:

- ticker latest cache
- order book snapshots
- short-lived OHLCV buffers
- pub/sub raw dan processed channels
- lock ringan dan dedup tertentu

## 8. API Surface

Target minimal untuk `/v1/market/*`:

- `/v1/market`
- `/v1/market/{id}`
- `/v1/market/{id}/ohlcv`
- `/v1/market/{id}/tickers`
- `/v1/market/{id}/orderbook`
- `/v1/market/{id}/arbitrage`

Response API harus:

- memakai `coin_id` sebagai identity utama
- menyertakan availability/freshness yang jelas
- tidak membocorkan detail source-specific yang tidak perlu ke frontend

## 9. Kebijakan CCXT

CCXT tetap bagian penting dari sistem, tetapi bukan satu-satunya fondasi market transport.

Kebijakan operasional:

1. Native WS diprioritaskan untuk exchange besar yang masuk hot path.
2. CCXT REST diprioritaskan untuk exchange minor, backfill, dan fallback.
3. CCXT WS / Pro hanya diaktifkan per exchange setelah lolos uji stabilitas, memory profile, reconnect behavior, dan error handling.
4. Tidak ada keputusan full migration ke CCXT websocket selama jalur native masih lebih stabil untuk kebutuhan realtime utama.

## 10. Implementasi Repo

Struktur implementasi yang mengikuti dokumen ini:

- `engine/market/ws/` untuk native major exchange clients
- `engine/market/ccxt.go` untuk REST fallback/minor exchanges
- `engine/market/adapters/ccxt_watch.go` untuk adapter websocket optional di masa depan
- `engine/market/mapping/` untuk symbol resolution dan mapping lifecycle
- `engine/market/ohlcv/` untuk candle backfill dan sync
- `engine/market/ticker/` untuk aggregation dan spread tracking
- `engine/market/arbitrage/` untuk detection dan alerting

## 11. Status Dokumen Lama

Dokumen lama tidak lagi menjadi referensi utama karena isinya saling overlap dan sebagian memuat asumsi yang sudah direvisi, terutama soal peran CCXT dan jalur realtime.

Gunakan dokumen ini sebagai referensi utama untuk:

- pengembangan Phase 2 market
- keputusan transport layer
- integrasi CCXT
- penulisan PRD turunan dan task breakdown
