from quant.scorer import calculate_quant_score


def test_calculate_quant_score_rewards_balanced_bullish_setup() -> None:
    score = calculate_quant_score(
        rsi=58,
        macd_histogram=1.5,
        close=105,
        bb_upper=110,
        bb_lower=95,
        ema_fast=106,
        ema_slow=101,
        bb_position=0.55,
        volume_ratio=1.8,
        atr_ratio=1.1,
    )

    assert score > 70


def test_calculate_quant_score_penalizes_overextended_setup() -> None:
    score = calculate_quant_score(
        rsi=88,
        macd_histogram=-2.0,
        close=120,
        bb_upper=121,
        bb_lower=90,
        ema_fast=95,
        ema_slow=102,
        bb_position=0.96,
        volume_ratio=0.5,
        atr_ratio=2.2,
        anomaly=True,
    )

    assert score < 40
