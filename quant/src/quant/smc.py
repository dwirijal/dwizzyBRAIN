from __future__ import annotations

import pandas as pd


def apply_smc_features(frame: pd.DataFrame) -> pd.DataFrame:
    if frame.empty:
        return frame.copy()

    enriched = frame.copy()
    high = enriched["high"]
    low = enriched["low"]
    close = enriched["close"]
    open_ = enriched["open"]

    swing_high = high.rolling(window=5).max().shift(1)
    swing_low = low.rolling(window=5).min().shift(1)
    trend_up = enriched["ema_fast"] > enriched["ema_slow"]
    bullish_break = close > swing_high
    bearish_break = close < swing_low
    enriched["smc_bos"] = (bullish_break | bearish_break).fillna(False)
    enriched["smc_choch"] = ((trend_up & bearish_break) | (~trend_up & bullish_break)).fillna(False)
    bullish_sweep = (high > swing_high) & (close < swing_high)
    bearish_sweep = (low < swing_low) & (close > swing_low)
    enriched["smc_liquidity_sweep"] = (bullish_sweep | bearish_sweep).fillna(False)
    enriched["smc_premium_zone"] = (close > ((high.rolling(window=20).max() + low.rolling(window=20).min()) / 2.0)).fillna(False)
    enriched["smc_discount_zone"] = (close < ((high.rolling(window=20).max() + low.rolling(window=20).min()) / 2.0)).fillna(False)
    bearish_impulse = (close < open_) & (close.shift(1) < open_.shift(1)) & (close.shift(2) < open_.shift(2))
    bullish_impulse = (close > open_) & (close.shift(1) > open_.shift(1)) & (close.shift(2) > open_.shift(2))
    enriched["smc_order_block"] = ((trend_up & bearish_impulse) | (~trend_up & bullish_impulse)).fillna(False)
    enriched["smc_fvg"] = ((low.shift(1) > high.shift(2)) | (high.shift(1) < low.shift(2))).fillna(False)
    return enriched
