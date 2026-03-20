from __future__ import annotations

import json
from dataclasses import dataclass
from datetime import UTC, datetime
from typing import Iterable

import psycopg
from psycopg.rows import dict_row


@dataclass(slots=True)
class BackfillTarget:
    coin_id: str
    exchange: str
    timeframe: str


class PostgresBackfillStore:
    def __init__(self, dsn: str) -> None:
        self.dsn = dsn

    def load_candles(
        self,
        target: BackfillTarget,
        *,
        since: datetime | None = None,
        limit: int | None = None,
    ) -> list[dict]:
        query = [
            "SELECT time, coin_id, exchange, symbol, timeframe, open, high, low, close, volume",
            "FROM ohlcv",
            "WHERE coin_id = %(coin_id)s AND exchange = %(exchange)s AND timeframe = %(timeframe)s",
        ]
        params: dict[str, object] = {
            "coin_id": target.coin_id.strip(),
            "exchange": target.exchange.strip().lower(),
            "timeframe": target.timeframe.strip().lower(),
        }
        if since is not None:
            query.append("AND time >= %(since)s")
            params["since"] = since.astimezone(UTC)
        query.append("ORDER BY time ASC")
        if limit is not None:
            query.append("LIMIT %(limit)s")
            params["limit"] = limit

        sql = "\n".join(query)
        with psycopg.connect(self.dsn, row_factory=dict_row) as conn:
            with conn.cursor() as cur:
                cur.execute(sql, params)
                rows = cur.fetchall()
        return [
            {
                "timestamp": row["time"].astimezone(UTC).isoformat().replace("+00:00", "Z"),
                "symbol": row["symbol"],
                "exchange": row["exchange"],
                "timeframe": row["timeframe"],
                "open": float(row["open"]),
                "high": float(row["high"]),
                "low": float(row["low"]),
                "close": float(row["close"]),
                "volume": float(row["volume"]),
            }
            for row in rows
        ]

    def upsert_candles(self, rows: Iterable[dict]) -> int:
        rows = list(rows)
        if not rows:
            return 0

        sql = """
INSERT INTO ohlcv (
    time, symbol_id, interval, coin_id, exchange, symbol, timeframe, open, high, low, close, volume, quote_volume, trades, is_closed
)
VALUES (
    %(time)s, %(symbol_id)s, %(interval)s, %(coin_id)s, %(exchange)s, %(symbol)s, %(timeframe)s, %(open)s, %(high)s,
    %(low)s, %(close)s, %(volume)s, %(quote_volume)s, %(trades)s, %(is_closed)s
)
ON CONFLICT (time, coin_id, exchange, timeframe)
DO UPDATE SET
    symbol_id = EXCLUDED.symbol_id,
    interval = EXCLUDED.interval,
    symbol = EXCLUDED.symbol,
    open = EXCLUDED.open,
    high = EXCLUDED.high,
    low = EXCLUDED.low,
    close = EXCLUDED.close,
    volume = EXCLUDED.volume,
    quote_volume = EXCLUDED.quote_volume,
    trades = EXCLUDED.trades,
    is_closed = EXCLUDED.is_closed
"""

        with psycopg.connect(self.dsn) as conn:
            payloads = []
            for row in rows:
                payloads.append(
                    {
                        "time": row.get("time") or row.get("timestamp"),
                        "symbol_id": self._ensure_symbol_id(conn, row),
                        "interval": str(row.get("timeframe") or "").strip().lower(),
                        "coin_id": str(row.get("coin_id") or row.get("symbol") or "").strip(),
                        "exchange": str(row.get("exchange") or "").strip().lower(),
                        "symbol": str(row.get("symbol") or row.get("coin_id") or "").strip(),
                        "timeframe": str(row.get("timeframe") or "").strip().lower(),
                        "open": row.get("open"),
                        "high": row.get("high"),
                        "low": row.get("low"),
                        "close": row.get("close"),
                        "volume": row.get("volume"),
                        "quote_volume": row.get("quote_volume"),
                        "trades": row.get("trades"),
                        "is_closed": bool(row.get("is_closed", True)),
                    }
                )

            with conn.cursor() as cur:
                cur.executemany(sql, payloads)
            conn.commit()
        return len(rows)

    def _ensure_symbol_id(self, conn: psycopg.Connection, row: dict) -> str:
        query = """
INSERT INTO symbols (exchange, symbol, base_currency, quote_currency, active, metadata, updated_at)
VALUES (%s, %s, %s, %s, TRUE, '{}'::jsonb, NOW())
ON CONFLICT (exchange, symbol)
DO UPDATE SET
    base_currency = EXCLUDED.base_currency,
    quote_currency = EXCLUDED.quote_currency,
    active = TRUE,
    updated_at = NOW()
RETURNING id
"""
        exchange = str(row.get("exchange") or "").strip().lower()
        symbol = str(row.get("symbol") or row.get("coin_id") or "").strip()
        base_currency, quote_currency = _split_symbol(symbol)

        with conn.cursor() as cur:
            cur.execute(query, (exchange, symbol, base_currency, quote_currency))
            result = cur.fetchone()
        if result is None:
            raise RuntimeError("failed to ensure symbol id")
        return str(result[0])

    def upsert_indicators(self, rows: Iterable[dict]) -> int:
        rows = list(rows)
        if not rows:
            return 0

        columns = [
            "time",
            "symbol",
            "timeframe",
            "ema_9",
            "ema_21",
            "ema_50",
            "ema_200",
            "sma_50",
            "sma_200",
            "vwap",
            "supertrend",
            "supertrend_dir",
            "adx",
            "ichimoku_tenkan",
            "ichimoku_kijun",
            "ichimoku_senkou_a",
            "ichimoku_senkou_b",
            "rsi_14",
            "rsi_2",
            "macd",
            "macd_signal",
            "macd_hist",
            "stoch_k",
            "stoch_d",
            "cci_20",
            "roc_10",
            "mfi_14",
            "atr_14",
            "bb_upper",
            "bb_mid",
            "bb_lower",
            "bb_pct_b",
            "bb_width",
            "kc_upper",
            "kc_lower",
            "hist_vol_20",
            "obv",
            "cmf_20",
            "volume_sma20",
            "volume_ratio",
            "volume_trend",
            "pivot_classic",
            "pivot_r1",
            "pivot_s1",
            "fib_382",
            "fib_500",
            "fib_618",
        ]
        values = []
        for row in rows:
            values.append(tuple(row.get(column) for column in columns))

        placeholders = ", ".join(f"%({column})s" for column in columns)
        sql = f"""
INSERT INTO candle_indicators ({", ".join(columns)})
VALUES ({placeholders})
ON CONFLICT (time, symbol, timeframe)
DO UPDATE SET {", ".join(f"{column} = EXCLUDED.{column}" for column in columns[3:])}
"""

        with psycopg.connect(self.dsn) as conn:
            with conn.cursor() as cur:
                cur.executemany(sql, [dict(zip(columns, value, strict=True)) for value in values])
            conn.commit()
        return len(rows)

    def upsert_features(self, rows: Iterable[dict]) -> int:
        rows = list(rows)
        if not rows:
            return 0

        columns = [
            "time",
            "symbol",
            "timeframe",
            "candle_body_pct",
            "upper_wick_pct",
            "lower_wick_pct",
            "dist_from_ema9",
            "dist_from_ema21",
            "dist_from_ema50",
            "dist_from_ema200",
            "dist_from_vwap",
            "bb_position",
            "atr_ratio",
            "kc_position",
            "rsi_slope",
            "macd_hist_slope",
            "obv_slope",
            "change_1h",
            "change_4h",
            "change_1d",
            "change_1w",
            "pattern_doji",
            "pattern_hammer",
            "pattern_shooting_star",
            "pattern_engulfing",
            "pattern_morning_star",
            "pattern_evening_star",
            "pattern_marubozu",
            "pattern_inside_bar",
            "pattern_pinbar",
            "smc_order_block",
            "smc_fvg",
            "smc_bos",
            "smc_choch",
            "smc_liquidity_sweep",
            "smc_premium_zone",
            "smc_discount_zone",
        ]
        placeholders = ", ".join(f"%({column})s" for column in columns)
        sql = f"""
INSERT INTO candle_features ({", ".join(columns)})
VALUES ({placeholders})
ON CONFLICT (time, symbol, timeframe)
DO UPDATE SET {", ".join(f"{column} = EXCLUDED.{column}" for column in columns[3:])}
"""

        with psycopg.connect(self.dsn) as conn:
            with conn.cursor() as cur:
                cur.executemany(sql, [{column: row.get(column) for column in columns} for row in rows])
            conn.commit()
        return len(rows)

    def upsert_macro_events(self, rows: Iterable[dict]) -> int:
        rows = list(rows)
        if not rows:
            return 0

        columns = [
            "time",
            "series_id",
            "series_name",
            "source",
            "event_type",
            "value",
            "importance",
            "metadata",
        ]
        sql = f"""
INSERT INTO macro_events ({", ".join(columns)})
VALUES ({", ".join(f"%({column})s" for column in columns)})
ON CONFLICT (time, series_id)
DO UPDATE SET
    series_name = EXCLUDED.series_name,
    source = EXCLUDED.source,
    event_type = EXCLUDED.event_type,
    value = EXCLUDED.value,
    importance = EXCLUDED.importance,
    metadata = EXCLUDED.metadata
"""

        payloads = []
        for row in rows:
            metadata = row.get("metadata") or {}
            event_time = row.get("time") or row.get("timestamp")
            payloads.append(
                {
                    "time": event_time,
                    "series_id": row.get("series_id"),
                    "series_name": row.get("series_name"),
                    "source": row.get("source", "fred"),
                    "event_type": row.get("event_type", "macro"),
                    "value": row.get("value"),
                    "importance": row.get("importance", 1),
                    "metadata": json.dumps(metadata, separators=(",", ":")),
                }
            )

        with psycopg.connect(self.dsn) as conn:
            with conn.cursor() as cur:
                cur.executemany(sql, payloads)
            conn.commit()
        return len(rows)

    def load_macro_events(
        self,
        *,
        series_ids: Iterable[str] | None = None,
        since: datetime | None = None,
        limit: int | None = None,
    ) -> list[dict]:
        query = [
            "SELECT time, series_id, series_name, source, event_type, value, importance, metadata",
            "FROM macro_events",
            "WHERE 1 = 1",
        ]
        params: dict[str, object] = {}
        series_list = [item.strip().upper() for item in (series_ids or []) if item.strip()]
        if series_list:
            query.append("AND series_id = ANY(%(series_ids)s)")
            params["series_ids"] = series_list
        if since is not None:
            query.append("AND time >= %(since)s")
            params["since"] = since.astimezone(UTC)
        query.append("ORDER BY time ASC")
        if limit is not None:
            query.append("LIMIT %(limit)s")
            params["limit"] = limit

        sql = "\n".join(query)
        with psycopg.connect(self.dsn, row_factory=dict_row) as conn:
            with conn.cursor() as cur:
                cur.execute(sql, params)
                rows = cur.fetchall()

        return [
            {
                "time": row["time"].astimezone(UTC),
                "series_id": row["series_id"],
                "series_name": row["series_name"],
                "source": row["source"],
                "event_type": row["event_type"],
                "value": float(row["value"]),
                "importance": int(row["importance"]),
                "metadata": row["metadata"] or {},
            }
            for row in rows
        ]

    def upsert_candle_event_labels(self, rows: Iterable[dict]) -> int:
        rows = list(rows)
        if not rows:
            return 0

        columns = [
            "time",
            "symbol",
            "timeframe",
            "macro_environment",
            "proximity_label",
            "rate_direction",
            "rate_regime",
            "cpi_trend",
            "last_surprise_label",
            "last_surprise_value",
            "hours_to_event",
            "hours_from_event",
            "vol_context",
            "nearest_event_series_id",
            "nearest_event_time",
        ]
        sql = f"""
INSERT INTO candle_event_labels ({", ".join(columns)})
VALUES ({", ".join(f"%({column})s" for column in columns)})
ON CONFLICT (time, symbol, timeframe)
DO UPDATE SET
    macro_environment = EXCLUDED.macro_environment,
    proximity_label = EXCLUDED.proximity_label,
    rate_direction = EXCLUDED.rate_direction,
    rate_regime = EXCLUDED.rate_regime,
    cpi_trend = EXCLUDED.cpi_trend,
    last_surprise_label = EXCLUDED.last_surprise_label,
    last_surprise_value = EXCLUDED.last_surprise_value,
    hours_to_event = EXCLUDED.hours_to_event,
    hours_from_event = EXCLUDED.hours_from_event,
    vol_context = EXCLUDED.vol_context,
    nearest_event_series_id = EXCLUDED.nearest_event_series_id,
    nearest_event_time = EXCLUDED.nearest_event_time
"""

        with psycopg.connect(self.dsn) as conn:
            with conn.cursor() as cur:
                cur.executemany(
                    sql,
                    [
                        {
                            **row,
                            "time": row.get("time") or row.get("timestamp"),
                        }
                        for row in rows
                    ],
                )
            conn.commit()
        return len(rows)

    def load_pattern_source_rows(
        self,
        target: BackfillTarget,
        *,
        since: datetime | None = None,
        limit: int | None = None,
    ) -> list[dict]:
        query = [
            "SELECT",
            "  o.time, o.symbol, o.timeframe, o.close,",
            "  i.ema_9, i.ema_21, i.ema_50, i.ema_200, i.sma_50, i.sma_200, i.vwap,",
            "  i.supertrend, i.supertrend_dir, i.adx, i.ichimoku_tenkan, i.ichimoku_kijun,",
            "  i.ichimoku_senkou_a, i.ichimoku_senkou_b, i.rsi_14, i.rsi_2, i.macd,",
            "  i.macd_signal, i.macd_hist, i.stoch_k, i.stoch_d, i.cci_20, i.roc_10, i.mfi_14,",
            "  i.atr_14, i.bb_upper, i.bb_mid, i.bb_lower, i.bb_pct_b, i.bb_width, i.kc_upper,",
            "  i.kc_lower, i.hist_vol_20, i.obv, i.cmf_20, i.volume_sma20, i.volume_ratio,",
            "  i.volume_trend, i.pivot_classic, i.pivot_r1, i.pivot_s1, i.fib_382, i.fib_500, i.fib_618,",
            "  f.candle_body_pct, f.upper_wick_pct, f.lower_wick_pct, f.dist_from_ema9, f.dist_from_ema21,",
            "  f.dist_from_ema50, f.dist_from_ema200, f.dist_from_vwap, f.bb_position, f.atr_ratio,",
            "  f.kc_position, f.rsi_slope, f.macd_hist_slope, f.obv_slope, f.change_1h, f.change_4h,",
            "  f.change_1d, f.change_1w, f.pattern_doji, f.pattern_hammer, f.pattern_shooting_star,",
            "  f.pattern_engulfing, f.pattern_morning_star, f.pattern_evening_star, f.pattern_marubozu,",
            "  f.pattern_inside_bar, f.pattern_pinbar, f.smc_order_block, f.smc_fvg, f.smc_bos,",
            "  f.smc_choch, f.smc_liquidity_sweep, f.smc_premium_zone, f.smc_discount_zone,",
            "  e.macro_environment, e.proximity_label, e.rate_direction, e.rate_regime, e.cpi_trend,",
            "  e.last_surprise_label, e.last_surprise_value, e.hours_to_event, e.hours_from_event, e.vol_context",
            "FROM ohlcv o",
            "JOIN candle_indicators i USING (time, symbol, timeframe)",
            "JOIN candle_features f USING (time, symbol, timeframe)",
            "LEFT JOIN candle_event_labels e USING (time, symbol, timeframe)",
            "WHERE o.coin_id = %(coin_id)s AND o.exchange = %(exchange)s AND o.timeframe = %(timeframe)s",
        ]
        params: dict[str, object] = {
            "coin_id": target.coin_id.strip(),
            "exchange": target.exchange.strip().lower(),
            "timeframe": target.timeframe.strip().lower(),
        }
        if since is not None:
            query.append("AND o.time >= %(since)s")
            params["since"] = since.astimezone(UTC)
        query.append("ORDER BY o.time ASC")
        if limit is not None:
            query.append("LIMIT %(limit)s")
            params["limit"] = limit

        sql = "\n".join(query)
        with psycopg.connect(self.dsn, row_factory=dict_row) as conn:
            with conn.cursor() as cur:
                cur.execute(sql, params)
                rows = cur.fetchall()

        return [
            {
                **row,
                "time": row["time"].astimezone(UTC),
                "close": float(row["close"]) if row["close"] is not None else None,
            }
            for row in rows
        ]

    def load_prediction_rows(
        self,
        target: BackfillTarget,
        *,
        since: datetime | None = None,
        limit: int | None = None,
    ) -> list[dict]:
        query = [
            "SELECT",
            "  o.time, o.symbol, o.timeframe, o.close,",
            "  i.ema_9, i.ema_21, i.ema_50, i.ema_200, i.sma_50, i.sma_200, i.vwap,",
            "  i.supertrend, i.supertrend_dir, i.adx, i.ichimoku_tenkan, i.ichimoku_kijun,",
            "  i.ichimoku_senkou_a, i.ichimoku_senkou_b, i.rsi_14, i.rsi_2, i.macd,",
            "  i.macd_signal, i.macd_hist, i.stoch_k, i.stoch_d, i.cci_20, i.roc_10, i.mfi_14,",
            "  i.atr_14, i.bb_upper, i.bb_mid, i.bb_lower, i.bb_pct_b, i.bb_width, i.kc_upper,",
            "  i.kc_lower, i.hist_vol_20, i.obv, i.cmf_20, i.volume_sma20, i.volume_ratio,",
            "  i.volume_trend, i.pivot_classic, i.pivot_r1, i.pivot_s1, i.fib_382, i.fib_500, i.fib_618,",
            "  f.candle_body_pct, f.upper_wick_pct, f.lower_wick_pct, f.dist_from_ema9, f.dist_from_ema21,",
            "  f.dist_from_ema50, f.dist_from_ema200, f.dist_from_vwap, f.bb_position, f.atr_ratio,",
            "  f.kc_position, f.rsi_slope, f.macd_hist_slope, f.obv_slope, f.change_1h, f.change_4h,",
            "  f.change_1d, f.change_1w, f.pattern_doji, f.pattern_hammer, f.pattern_shooting_star,",
            "  f.pattern_engulfing, f.pattern_morning_star, f.pattern_evening_star, f.pattern_marubozu,",
            "  f.pattern_inside_bar, f.pattern_pinbar, f.smc_order_block, f.smc_fvg, f.smc_bos,",
            "  f.smc_choch, f.smc_liquidity_sweep, f.smc_premium_zone, f.smc_discount_zone,",
            "  o1.close AS close_1h_later,",
            "  o4.close AS close_4h_later,",
            "  o1d.close AS close_1d_later,",
            "  o1w.close AS close_1w_later",
            "FROM ohlcv o",
            "JOIN candle_indicators i USING (time, symbol, timeframe)",
            "JOIN candle_features f USING (time, symbol, timeframe)",
            "LEFT JOIN ohlcv o1",
            "  ON o1.symbol = o.symbol AND o1.timeframe = o.timeframe AND o1.time = o.time + INTERVAL '1 hour'",
            "LEFT JOIN ohlcv o4",
            "  ON o4.symbol = o.symbol AND o4.timeframe = o.timeframe AND o4.time = o.time + INTERVAL '4 hours'",
            "LEFT JOIN ohlcv o1d",
            "  ON o1d.symbol = o.symbol AND o1d.timeframe = o.timeframe AND o1d.time = o.time + INTERVAL '1 day'",
            "LEFT JOIN ohlcv o1w",
            "  ON o1w.symbol = o.symbol AND o1w.timeframe = o.timeframe AND o1w.time = o.time + INTERVAL '1 week'",
            "WHERE o.coin_id = %(coin_id)s AND o.exchange = %(exchange)s AND o.timeframe = %(timeframe)s",
        ]
        params: dict[str, object] = {
            "coin_id": target.coin_id.strip(),
            "exchange": target.exchange.strip().lower(),
            "timeframe": target.timeframe.strip().lower(),
        }
        if since is not None:
            query.append("AND o.time >= %(since)s")
            params["since"] = since.astimezone(UTC)
        query.append("ORDER BY o.time ASC")
        if limit is not None:
            query.append("LIMIT %(limit)s")
            params["limit"] = limit

        sql = "\n".join(query)
        with psycopg.connect(self.dsn, row_factory=dict_row) as conn:
            with conn.cursor() as cur:
                cur.execute(sql, params)
                rows = cur.fetchall()

        normalized = []
        for row in rows:
            normalized.append(
                {
                    **row,
                    "time": row["time"].astimezone(UTC),
                    "close": float(row["close"]) if row["close"] is not None else None,
                    "close_1h_later": float(row["close_1h_later"]) if row["close_1h_later"] is not None else None,
                    "close_4h_later": float(row["close_4h_later"]) if row["close_4h_later"] is not None else None,
                    "close_1d_later": float(row["close_1d_later"]) if row["close_1d_later"] is not None else None,
                    "close_1w_later": float(row["close_1w_later"]) if row["close_1w_later"] is not None else None,
                }
            )
        return normalized

    def upsert_embeddings(self, rows: Iterable[dict]) -> int:
        rows = list(rows)
        if not rows:
            return 0

        sql = """
INSERT INTO candle_embeddings (time, symbol, timeframe, embedding)
VALUES (%(time)s, %(symbol)s, %(timeframe)s, %(embedding)s::vector)
ON CONFLICT (time, symbol, timeframe)
DO UPDATE SET embedding = EXCLUDED.embedding
"""
        payloads = []
        for row in rows:
            payloads.append(
                {
                    "time": row.get("time"),
                    "symbol": row.get("symbol"),
                    "timeframe": row.get("timeframe"),
                    "embedding": _vector_literal(row.get("embedding")),
                }
            )

        with psycopg.connect(self.dsn) as conn:
            with conn.cursor() as cur:
                cur.executemany(sql, payloads)
            conn.commit()
        return len(rows)

    def find_similar_pattern_matches(
        self,
        symbol: str,
        timeframe: str,
        embedding: Iterable[object],
        *,
        limit: int = 20,
    ) -> list[dict]:
        sql = """
WITH ranked AS (
    SELECT
        emb.time,
        emb.symbol,
        emb.timeframe,
        ohlcv.close,
        cel.macro_environment,
        cel.proximity_label,
        cel.rate_direction,
        cel.cpi_trend,
        cel.last_surprise_label,
        cel.last_surprise_value,
        1 - (emb.embedding <=> %(embedding)s::vector) AS similarity_score
    FROM candle_embeddings emb
    JOIN ohlcv USING (time, symbol, timeframe)
    LEFT JOIN candle_event_labels cel USING (time, symbol, timeframe)
    WHERE emb.symbol = %(symbol)s AND emb.timeframe = %(timeframe)s
    ORDER BY emb.embedding <=> %(embedding)s::vector
    LIMIT %(limit)s
)
SELECT
    ranked.*,
    o1.close AS close_1h_later,
    o4.close AS close_4h_later,
    o1d.close AS close_1d_later,
    o1w.close AS close_1w_later
FROM ranked
LEFT JOIN ohlcv o1
    ON o1.symbol = ranked.symbol
   AND o1.timeframe = ranked.timeframe
   AND o1.time = ranked.time + INTERVAL '1 hour'
LEFT JOIN ohlcv o4
    ON o4.symbol = ranked.symbol
   AND o4.timeframe = ranked.timeframe
   AND o4.time = ranked.time + INTERVAL '4 hours'
LEFT JOIN ohlcv o1d
    ON o1d.symbol = ranked.symbol
   AND o1d.timeframe = ranked.timeframe
   AND o1d.time = ranked.time + INTERVAL '1 day'
LEFT JOIN ohlcv o1w
    ON o1w.symbol = ranked.symbol
   AND o1w.timeframe = ranked.timeframe
   AND o1w.time = ranked.time + INTERVAL '1 week'
ORDER BY ranked.similarity_score DESC, ranked.time DESC
"""

        params = {
            "symbol": symbol.strip(),
            "timeframe": timeframe.strip().lower(),
            "embedding": _vector_literal(list(embedding)),
            "limit": limit,
        }

        with psycopg.connect(self.dsn, row_factory=dict_row) as conn:
            with conn.cursor() as cur:
                cur.execute(sql, params)
                rows = cur.fetchall()

        return [
            {
                **row,
                "close": float(row["close"]) if row["close"] is not None else None,
                "close_1h_later": float(row["close_1h_later"]) if row["close_1h_later"] is not None else None,
                "close_4h_later": float(row["close_4h_later"]) if row["close_4h_later"] is not None else None,
                "close_1d_later": float(row["close_1d_later"]) if row["close_1d_later"] is not None else None,
                "close_1w_later": float(row["close_1w_later"]) if row["close_1w_later"] is not None else None,
                "similarity_score": float(row["similarity_score"]),
            }
            for row in rows
        ]


def _vector_literal(value: object) -> str:
    if isinstance(value, str):
        text = value.strip()
        if text.startswith("[") and text.endswith("]"):
            return text
        return f"[{text}]"

    if isinstance(value, Iterable):
        parts = []
        for item in value:
            if item is None:
                parts.append("0")
            else:
                parts.append(f"{float(item):.10f}")
        return f"[{','.join(parts)}]"

    return "[]"


def _split_symbol(symbol: str) -> tuple[str, str]:
    normalized = symbol.strip().upper()
    if not normalized:
        return "", ""
    if "/" in normalized:
        base, quote = normalized.split("/", 1)
        return base.strip(), quote.strip()
    for quote in ["USDT", "USDC", "BUSD", "BTC", "ETH", "EUR", "USD", "GBP", "JPY"]:
        if normalized.endswith(quote) and len(normalized) > len(quote):
            return normalized[: -len(quote)], quote
    return normalized, ""
