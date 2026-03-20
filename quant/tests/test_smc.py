from __future__ import annotations

from datetime import UTC, datetime, timedelta

from quant.indicators import build_frame
from quant.smc import apply_smc_features


def _payloads(count: int = 80) -> list[dict]:
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


def test_apply_smc_features_adds_expected_columns() -> None:
    frame = build_frame(_payloads())
    frame["ema_fast"] = frame["close"].ewm(span=9, adjust=False).mean()
    frame["ema_slow"] = frame["close"].ewm(span=21, adjust=False).mean()

    enriched = apply_smc_features(frame)

    assert "smc_bos" in enriched.columns
    assert "smc_choch" in enriched.columns
    assert "smc_liquidity_sweep" in enriched.columns
    assert "smc_premium_zone" in enriched.columns
    assert "smc_discount_zone" in enriched.columns
    assert "smc_order_block" in enriched.columns
    assert "smc_fvg" in enriched.columns
