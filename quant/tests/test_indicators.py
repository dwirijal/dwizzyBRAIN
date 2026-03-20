from __future__ import annotations

from datetime import UTC, datetime, timedelta

from quant.indicators import apply_indicators, build_frame, latest_complete_row


def _payloads(count: int = 300) -> list[dict]:
    base = datetime(2026, 3, 18, tzinfo=UTC)
    payloads: list[dict] = []
    for index in range(count):
        close = 100 + (index * 0.8)
        payloads.append(
            {
                "symbol": "BTCUSDT",
                "exchange": "binance",
                "timeframe": "1m",
                "timestamp": (base + timedelta(minutes=index)).isoformat().replace("+00:00", "Z"),
                "open": close - 0.5,
                "high": close + 1.0,
                "low": close - 1.0,
                "close": close,
                "volume": 1000 + index,
            }
        )
    return payloads


def test_build_frame_sorts_and_normalizes() -> None:
    payloads = list(reversed(_payloads(3)))
    frame = build_frame(payloads)

    assert list(frame["close"]) == [100.0, 100.8, 101.6]
    assert str(frame["timestamp"].dtype).startswith("datetime64[ns, UTC]")


def test_apply_indicators_produces_complete_latest_row() -> None:
    frame = build_frame(_payloads())
    enriched = apply_indicators(frame)
    latest = latest_complete_row(enriched)

    assert latest is not None
    assert latest["ema_fast"] > 0
    assert latest["bb_upper"] > latest["bb_lower"]
    assert 0 <= latest["rsi"] <= 100
    assert latest["sma_50"] > 0
    assert latest["sma_200"] > 0
    assert latest["dema_21"] > 0
    assert latest["tema_21"] > 0
    assert latest["hma_21"] > 0
    assert latest["adx_14"] >= 0
    assert latest["stoch_k"] >= 0
    assert latest["stoch_d"] >= 0
    assert latest["williams_r_14"] <= 0
    assert latest["aroon_up_25"] >= 0
    assert latest["aroon_down_25"] >= 0
    assert latest["donchian_upper_20"] >= latest["donchian_lower_20"]
    assert latest["cci_20"] == latest["cci_20"]
    assert latest["roc_10"] == latest["roc_10"]
    assert latest["mfi_14"] >= 0
    assert latest["vwap"] > 0
    assert latest["bb_pct_b"] == latest["bb_pct_b"]
    assert latest["bb_width"] == latest["bb_width"]
    assert latest["kc_upper"] > latest["kc_lower"]
    assert latest["hist_vol_20"] >= 0
    assert latest["change_1h"] == latest["change_1h"]
    assert latest["change_4h"] == latest["change_4h"]
    assert latest["supertrend"] == latest["supertrend"]
    assert latest["supertrend_dir"] in (-1, 1)
    assert latest["ichimoku_tenkan"] == latest["ichimoku_tenkan"]
    assert latest["ichimoku_kijun"] == latest["ichimoku_kijun"]
    assert latest["ichimoku_senkou_a"] == latest["ichimoku_senkou_a"]
    assert latest["ichimoku_senkou_b"] == latest["ichimoku_senkou_b"]
    assert latest["pivot_classic"] == latest["pivot_classic"]
    assert latest["pivot_r1"] == latest["pivot_r1"]
    assert latest["pivot_s1"] == latest["pivot_s1"]
    assert latest["fib_382"] == latest["fib_382"]
    assert latest["fib_500"] == latest["fib_500"]
    assert latest["fib_618"] == latest["fib_618"]
    assert latest["fib_ext_1272"] == latest["fib_ext_1272"]
    assert latest["fib_ext_1618"] == latest["fib_ext_1618"]
    assert latest["swing_high_10"] == latest["swing_high_10"]
    assert latest["swing_low_10"] == latest["swing_low_10"]
    assert "pattern_doji" in enriched.columns
    assert "pattern_pinbar" in enriched.columns
    assert "smc_bos" in enriched.columns
    assert "smc_choch" in enriched.columns
    assert "smc_liquidity_sweep" in enriched.columns
    assert "smc_premium_zone" in enriched.columns
    assert "smc_discount_zone" in enriched.columns
    assert "smc_order_block" in enriched.columns
    assert "smc_fvg" in enriched.columns
