from __future__ import annotations

from datetime import UTC, datetime, timedelta

from quant.anomaly import detect_anomaly
from quant.indicators import apply_indicators, build_frame


def _payloads(volume_spike: bool = False) -> list[dict]:
    base = datetime(2026, 3, 18, tzinfo=UTC)
    payloads: list[dict] = []
    for index in range(40):
        close = 100 + (index * 0.6)
        volume = 1000 + index
        if volume_spike and index == 39:
            volume = 5000
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
                "volume": volume,
            }
        )
    return payloads


def test_detect_anomaly_flags_volume_spike() -> None:
    frame = apply_indicators(build_frame(_payloads(volume_spike=True)))
    anomaly, anomaly_type = detect_anomaly(frame)

    assert anomaly is True
    assert anomaly_type == "volume_spike"
