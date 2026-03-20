from __future__ import annotations

from datetime import UTC, datetime, timedelta

from quant.backfill.compute_bulk import compute_bulk_records


def _candles(count: int = 300) -> list[dict]:
    base = datetime(2026, 3, 18, tzinfo=UTC)
    rows: list[dict] = []
    for index in range(count):
        close = 100 + (index * 0.8)
        rows.append(
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
    return rows


def test_compute_bulk_records_returns_indicator_and_feature_rows() -> None:
    indicators, features = compute_bulk_records(_candles())

    assert indicators
    assert features
    assert len(indicators) == len(features)
    last_indicator = indicators[-1]
    last_feature = features[-1]
    assert last_indicator["symbol"] == "BTCUSDT"
    assert last_indicator["timeframe"] == "1m"
    assert last_indicator["ema_9"] > 0
    assert last_indicator["rsi_14"] >= 0
    assert last_feature["candle_body_pct"] >= 0
    assert "pattern_doji" in last_feature
