from __future__ import annotations

import pandas as pd


def detect_anomaly(frame: pd.DataFrame) -> tuple[bool, str | None]:
    if frame.empty or len(frame) < 21:
        return False, None

    latest = frame.iloc[-1]
    window = frame.iloc[-21:-1]
    if window.empty:
        return False, None

    latest_volume = float(latest.get("volume", 0.0) or 0.0)
    avg_volume = float(window["volume"].mean())
    volume_ratio = latest_volume / avg_volume if avg_volume > 0 else 0.0

    prev_close = float(window.iloc[-1].get("close", 0.0) or 0.0)
    close = float(latest.get("close", 0.0) or 0.0)
    if prev_close <= 0 or close <= 0:
        return False, None

    price_change = abs(close - prev_close) / prev_close
    ema_fast = float(latest.get("ema_fast", close) or close)
    ema_distance = abs(close - ema_fast) / close
    atr_14 = float(latest.get("atr_14", 0.0) or 0.0)
    atr_ratio = float(latest.get("atr_ratio", 0.0) or 0.0)

    if volume_ratio >= 2.5:
        return True, "volume_spike"
    if atr_14 > 0 and atr_ratio >= 1.8 and ema_distance >= 0.02:
        return True, "price_deviation"
    if price_change >= 0.03 and volume_ratio >= 1.5:
        return True, "price_deviation"

    return False, None
