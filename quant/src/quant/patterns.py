from __future__ import annotations

from collections.abc import Iterable, Mapping


RATE_REGIME_ENCODING = {
    "low_zirp": 0.0,
    "low": 0.0,
    "medium": 1.0,
    "med": 1.0,
    "high": 2.0,
    "very_high": 3.0,
    "very-high": 3.0,
}

VOL_CONTEXT_ENCODING = {
    "low_vol_compression": 0.0,
    "compression": 0.0,
    "normal_vol": 1.0,
    "normal": 1.0,
    "high_vol": 2.0,
    "high": 2.0,
    "extreme_vol": 3.0,
    "extreme": 3.0,
}


def build_fingerprint(row: Mapping[str, object]) -> list[float]:
    """Build a 30-dimensional normalized embedding for a candle row."""
    return [
        # Trend
        clamp(_safe_float(row.get("dist_from_ema9")) / 10.0),
        clamp(_safe_float(row.get("dist_from_ema21")) / 10.0),
        clamp(_safe_float(row.get("dist_from_ema50")) / 15.0),
        clamp(_safe_float(row.get("dist_from_ema200")) / 30.0),
        clamp(_safe_float(row.get("adx")) / 100.0),
        _direction_value(row.get("supertrend_dir")),
        # Momentum
        clamp(_safe_float(row.get("rsi_14")) / 100.0),
        clamp(_safe_float(row.get("rsi_2")) / 100.0),
        clamp(_ratio(row.get("macd_hist"), row.get("atr_14"))),
        clamp(_ratio(row.get("macd_hist_slope"), row.get("atr_14"))),
        clamp(_safe_float(row.get("stoch_k")) / 100.0),
        clamp(_safe_float(row.get("cci_20")) / 200.0),
        # Volatility
        clamp(_safe_float(row.get("bb_position"))),
        clamp(_safe_float(row.get("bb_width")) * 10.0),
        clamp(_ratio(row.get("atr_ratio"), 3.0)),
        clamp(_safe_float(row.get("hist_vol_20")) / 100.0),
        clamp(_safe_float(row.get("kc_position"))),
        # Volume
        clamp(_safe_float(row.get("volume_ratio")) / 5.0),
        clamp(_safe_float(row.get("volume_trend")) * 2.0),
        clamp(_safe_float(row.get("cmf_20"))),
        clamp(_safe_float(row.get("obv_slope"))),
        # Price action
        clamp(_safe_float(row.get("candle_body_pct"))),
        clamp(_safe_float(row.get("upper_wick_pct"))),
        clamp(_safe_float(row.get("lower_wick_pct"))),
        clamp(_ratio(row.get("dist_from_vwap"), 5.0)),
        clamp(_ratio(row.get("rsi_slope"), 20.0)),
        # Event context
        clamp(_ratio(row.get("hours_to_event"), 72.0)),
        clamp(_ratio(row.get("last_surprise_value"), 3.0)),
        clamp(_encode_rate_regime(row.get("rate_regime")) / 3.0),
        clamp(_encode_vol_context(row.get("vol_context")) / 3.0),
    ]


def build_embedding_record(row: Mapping[str, object]) -> dict[str, object]:
    """Build a pgvector-ready payload for a row."""
    embedding = build_fingerprint(row)
    return {
        "time": row.get("time") or row.get("timestamp"),
        "symbol": str(row.get("symbol", "")).strip(),
        "timeframe": str(row.get("timeframe", "")).strip().lower(),
        "embedding": embedding,
    }


def build_embedding_records(rows: Iterable[Mapping[str, object]]) -> list[dict[str, object]]:
    return [build_embedding_record(row) for row in rows]


def clamp(value: float, lo: float = -1.0, hi: float = 1.0) -> float:
    return max(lo, min(hi, value))


def _safe_float(value: object, default: float = 0.0) -> float:
    try:
        if value is None:
            return default
        result = float(value)
    except (TypeError, ValueError):
        return default
    if result != result:
        return default
    return result


def _ratio(value: object, denominator: float) -> float:
    base = _safe_float(value)
    denom = _safe_float(denominator)
    if denom == 0:
        return 0.0
    return base / denom


def _direction_value(value: object) -> float:
    base = _safe_float(value)
    if base > 0:
        return 1.0
    if base < 0:
        return -1.0
    return 0.0


def _encode_rate_regime(value: object) -> float:
    key = str(value or "").strip().lower()
    return RATE_REGIME_ENCODING.get(key, 0.0)


def _encode_vol_context(value: object) -> float:
    key = str(value or "").strip().lower()
    return VOL_CONTEXT_ENCODING.get(key, 1.0 if key else 0.0)
