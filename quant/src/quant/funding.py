from __future__ import annotations


def extract_funding_rate(payload: dict) -> float | None:
    for key in ("funding_rate", "fundingRate", "funding"):
        raw = payload.get(key)
        if raw in (None, ""):
            continue
        try:
            return float(raw)
        except (TypeError, ValueError):
            continue
    return None


def classify_funding_sentiment(rate: float | None) -> str | None:
    if rate is None:
        return None
    if rate > 0.0005:
        return "bearish"
    if rate < -0.0005:
        return "bullish"
    return "neutral"
