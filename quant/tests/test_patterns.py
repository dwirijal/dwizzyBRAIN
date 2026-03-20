from __future__ import annotations

from decimal import Decimal

from quant.patterns import build_embedding_record, build_fingerprint


def test_build_fingerprint_returns_30_dimensions() -> None:
    row = {
        "dist_from_ema9": 2.5,
        "dist_from_ema21": -3.0,
        "dist_from_ema50": 6.0,
        "dist_from_ema200": -9.0,
        "adx": 42.0,
        "supertrend_dir": -1,
        "rsi_14": 58.0,
        "rsi_2": 73.0,
        "macd_hist": 1.5,
        "macd_hist_slope": 0.25,
        "atr_14": Decimal("2.0"),
        "stoch_k": 61.0,
        "cci_20": 100.0,
        "bb_position": 0.55,
        "bb_width": 0.18,
        "atr_ratio": 1.2,
        "hist_vol_20": 65.0,
        "kc_position": 0.44,
        "volume_ratio": 1.7,
        "volume_trend": 0.35,
        "cmf_20": 0.2,
        "obv_slope": -0.4,
        "candle_body_pct": 0.62,
        "upper_wick_pct": 0.21,
        "lower_wick_pct": 0.17,
        "dist_from_vwap": 4.0,
        "rsi_slope": 3.5,
        "hours_to_event": -12.0,
        "last_surprise_value": 1.5,
        "rate_regime": "high",
        "vol_context": "extreme_vol",
    }

    fingerprint = build_fingerprint(row)

    assert len(fingerprint) == 30
    assert fingerprint[0] == 0.25
    assert fingerprint[1] == -0.3
    assert fingerprint[5] == -1.0
    assert fingerprint[6] == 0.58
    assert fingerprint[8] == 0.75
    assert fingerprint[12] == 0.55
    assert fingerprint[16] == 0.44
    assert fingerprint[27] == 0.5
    assert fingerprint[28] == 2.0 / 3.0
    assert fingerprint[29] == 1.0


def test_build_embedding_record_uses_timestamp_fallback() -> None:
    row = {
        "timestamp": "2026-03-18T12:00:00Z",
        "symbol": "BTCUSDT",
        "timeframe": "1m",
        "dist_from_ema9": 0.0,
        "dist_from_ema21": 0.0,
        "dist_from_ema50": 0.0,
        "dist_from_ema200": 0.0,
        "adx": 0.0,
        "supertrend_dir": 1,
        "rsi_14": 50.0,
        "rsi_2": 50.0,
        "macd_hist": 0.0,
        "macd_hist_slope": 0.0,
        "atr_14": 1.0,
        "stoch_k": 50.0,
        "cci_20": 0.0,
        "bb_position": 0.5,
        "bb_width": 0.1,
        "atr_ratio": 1.0,
        "hist_vol_20": 1.0,
        "kc_position": 0.5,
        "volume_ratio": 1.0,
        "volume_trend": 0.0,
        "cmf_20": 0.0,
        "obv_slope": 0.0,
        "candle_body_pct": 0.5,
        "upper_wick_pct": 0.25,
        "lower_wick_pct": 0.25,
        "dist_from_vwap": 0.0,
        "rsi_slope": 0.0,
        "hours_to_event": 0.0,
        "last_surprise_value": 0.0,
        "rate_regime": "low_zirp",
        "vol_context": "normal_vol",
    }

    record = build_embedding_record(row)

    assert record["time"] == "2026-03-18T12:00:00Z"
    assert record["symbol"] == "BTCUSDT"
    assert record["timeframe"] == "1m"
    assert len(record["embedding"]) == 30
