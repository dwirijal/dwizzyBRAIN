from __future__ import annotations


def calculate_quant_score(
    *,
    rsi: float,
    macd_histogram: float,
    close: float,
    bb_upper: float,
    bb_lower: float,
    ema_fast: float,
    ema_slow: float,
    bb_position: float,
    volume_ratio: float,
    atr_ratio: float,
    anomaly: bool = False,
    funding_rate: float | None = None,
) -> float:
    score = 50.0

    if 45 <= rsi <= 55:
        score += 5
    elif 55 < rsi <= 70:
        score += 12
    elif 30 <= rsi < 45:
        score += 8
    elif rsi > 80 or rsi < 20:
        score -= 15

    if macd_histogram > 0:
        score += 15
    elif macd_histogram < 0:
        score -= 15

    if ema_fast > ema_slow:
        score += 12
    elif ema_fast < ema_slow:
        score -= 12

    if 0.35 <= bb_position <= 0.65:
        score += 8
    elif bb_position > 0.9 or bb_position < 0.1:
        score -= 10

    if volume_ratio >= 2.5:
        score += 10
    elif volume_ratio >= 1.5:
        score += 5
    elif volume_ratio < 0.7:
        score -= 4

    if atr_ratio >= 2.0:
        score -= 8
    elif 0.7 <= atr_ratio <= 1.4:
        score += 4

    if anomaly:
        score -= 12

    if funding_rate is not None:
        if funding_rate > 0.0005:
            score -= 3
        elif funding_rate < -0.0005:
            score += 3

    return max(0.0, min(100.0, round(score, 2)))
