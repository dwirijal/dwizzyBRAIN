from __future__ import annotations

import numpy as np
import pandas as pd

from quant.smc import apply_smc_features


def build_frame(payloads: list[dict]) -> pd.DataFrame:
    frame = pd.DataFrame(payloads)
    if frame.empty:
        return frame

    frame["timestamp"] = pd.to_datetime(frame["timestamp"], utc=True)
    numeric_columns = ["open", "high", "low", "close", "volume"]
    for column in numeric_columns:
        frame[column] = pd.to_numeric(frame[column], errors="coerce")

    return frame.sort_values("timestamp").reset_index(drop=True)


def apply_indicators(frame: pd.DataFrame) -> pd.DataFrame:
    if frame.empty:
        return frame.copy()

    enriched = frame.copy()

    open_ = enriched["open"]
    high = enriched["high"]
    low = enriched["low"]
    close = enriched["close"]
    volume = enriched["volume"]

    enriched["ema_fast"] = close.ewm(span=9, adjust=False).mean()
    enriched["ema_slow"] = close.ewm(span=21, adjust=False).mean()
    enriched["ema_50"] = close.ewm(span=50, adjust=False).mean()
    enriched["ema_200"] = close.ewm(span=200, adjust=False).mean()
    enriched["sma_50"] = close.rolling(window=50).mean()
    enriched["sma_200"] = close.rolling(window=200).mean()
    enriched["dema_21"] = _dema(close, 21)
    enriched["tema_21"] = _tema(close, 21)
    enriched["hma_21"] = _hma(close, 21)

    delta = close.diff()
    gain = delta.clip(lower=0)
    loss = -delta.clip(upper=0)
    average_gain = gain.ewm(alpha=1 / 14, min_periods=14, adjust=False).mean()
    average_loss = loss.ewm(alpha=1 / 14, min_periods=14, adjust=False).mean()
    rs = average_gain / average_loss.replace(0, pd.NA)
    enriched["rsi"] = 100 - (100 / (1 + rs))
    enriched.loc[(average_loss == 0) & (average_gain > 0), "rsi"] = 100.0
    enriched.loc[(average_gain == 0) & (average_loss > 0), "rsi"] = 0.0
    enriched.loc[(average_gain == 0) & (average_loss == 0), "rsi"] = 50.0
    enriched["rsi_2"] = _rsi(close, 2)

    ema12 = close.ewm(span=12, adjust=False).mean()
    ema26 = close.ewm(span=26, adjust=False).mean()
    enriched["macd"] = ema12 - ema26
    enriched["macd_signal"] = enriched["macd"].ewm(span=9, adjust=False).mean()
    enriched["macd_histogram"] = enriched["macd"] - enriched["macd_signal"]

    rolling_mean = close.rolling(window=20).mean()
    rolling_std = close.rolling(window=20).std(ddof=0)
    enriched["bb_middle"] = rolling_mean
    enriched["bb_upper"] = rolling_mean + (rolling_std * 2.0)
    enriched["bb_lower"] = rolling_mean - (rolling_std * 2.0)
    enriched["bb_position"] = (close - enriched["bb_lower"]) / (enriched["bb_upper"] - enriched["bb_lower"])
    enriched["bb_pct_b"] = enriched["bb_position"]
    enriched["bb_width"] = (enriched["bb_upper"] - enriched["bb_lower"]) / enriched["bb_middle"]

    typical_price = (high + low + close) / 3.0
    enriched["vwap"] = (typical_price * volume).cumsum() / volume.cumsum().replace(0, pd.NA)

    true_range = pd.concat(
        [
            high - low,
            (high - close.shift(1)).abs(),
            (low - close.shift(1)).abs(),
        ],
        axis=1,
    ).max(axis=1)
    enriched["atr_14"] = true_range.ewm(alpha=1 / 14, min_periods=14, adjust=False).mean()
    atr_sma20 = enriched["atr_14"].rolling(window=20).mean()
    enriched["atr_ratio"] = enriched["atr_14"] / atr_sma20.replace(0, pd.NA)
    enriched["hist_vol_20"] = close.pct_change().rolling(window=20).std(ddof=0) * (365 ** 0.5) * 100

    hl2 = (high + low) / 2.0
    supertrend_multiplier = 3.0
    upper_basic = hl2 + (supertrend_multiplier * enriched["atr_14"])
    lower_basic = hl2 - (supertrend_multiplier * enriched["atr_14"])
    final_upper = upper_basic.copy()
    final_lower = lower_basic.copy()
    for idx in range(1, len(enriched)):
        prev_close = close.iloc[idx - 1]
        final_upper.iloc[idx] = (
            upper_basic.iloc[idx]
            if (upper_basic.iloc[idx] < final_upper.iloc[idx - 1]) or (prev_close > final_upper.iloc[idx - 1])
            else final_upper.iloc[idx - 1]
        )
        final_lower.iloc[idx] = (
            lower_basic.iloc[idx]
            if (lower_basic.iloc[idx] > final_lower.iloc[idx - 1]) or (prev_close < final_lower.iloc[idx - 1])
            else final_lower.iloc[idx - 1]
        )
    supertrend = pd.Series(index=enriched.index, dtype="float64")
    supertrend_dir = pd.Series(index=enriched.index, dtype="float64")
    for idx in range(len(enriched)):
        if idx == 0 or pd.isna(enriched["atr_14"].iloc[idx]):
            continue
        if idx == 1:
            supertrend.iloc[idx] = final_lower.iloc[idx]
            supertrend_dir.iloc[idx] = 1.0 if close.iloc[idx] >= final_lower.iloc[idx] else -1.0
            continue
        prev_supertrend = supertrend.iloc[idx - 1]
        if close.iloc[idx] <= final_upper.iloc[idx]:
            supertrend.iloc[idx] = final_upper.iloc[idx]
            supertrend_dir.iloc[idx] = -1.0
        elif close.iloc[idx] >= final_lower.iloc[idx]:
            supertrend.iloc[idx] = final_lower.iloc[idx]
            supertrend_dir.iloc[idx] = 1.0
        else:
            supertrend.iloc[idx] = prev_supertrend
            supertrend_dir.iloc[idx] = supertrend_dir.iloc[idx - 1]
    enriched["supertrend"] = supertrend.ffill().fillna(hl2)
    enriched["supertrend_dir"] = supertrend_dir.ffill().fillna(1.0).astype("Int64")

    enriched["ichimoku_tenkan"] = (high.rolling(window=9).max() + low.rolling(window=9).min()) / 2.0
    enriched["ichimoku_kijun"] = (high.rolling(window=26).max() + low.rolling(window=26).min()) / 2.0
    enriched["ichimoku_senkou_a"] = (enriched["ichimoku_tenkan"] + enriched["ichimoku_kijun"]) / 2.0
    enriched["ichimoku_senkou_b"] = (high.rolling(window=52).max() + low.rolling(window=52).min()) / 2.0

    enriched["volume_sma20"] = volume.rolling(window=20).mean()
    enriched["volume_ratio"] = volume / enriched["volume_sma20"].replace(0, pd.NA)
    enriched["volume_trend"] = volume.rolling(window=5).mean() / enriched["volume_sma20"].replace(0, pd.NA)

    plus_dm = (high.diff()).where((high.diff() > (low.shift(1) - low)) & (high.diff() > 0), 0.0)
    minus_dm = ((low.shift(1) - low)).where(((low.shift(1) - low) > high.diff()) & ((low.shift(1) - low) > 0), 0.0)
    plus_dm = plus_dm.fillna(0.0)
    minus_dm = minus_dm.fillna(0.0)
    plus_dm_smoothed = plus_dm.ewm(alpha=1 / 14, min_periods=14, adjust=False).mean()
    minus_dm_smoothed = minus_dm.ewm(alpha=1 / 14, min_periods=14, adjust=False).mean()
    plus_di = 100 * plus_dm_smoothed / enriched["atr_14"].replace(0, pd.NA)
    minus_di = 100 * minus_dm_smoothed / enriched["atr_14"].replace(0, pd.NA)
    dx = (100 * (plus_di - minus_di).abs() / (plus_di + minus_di).replace(0, pd.NA)).replace([pd.NA], pd.NA)
    enriched["adx_14"] = dx.ewm(alpha=1 / 14, min_periods=14, adjust=False).mean()

    lowest_low = low.rolling(window=14).min()
    highest_high = high.rolling(window=14).max()
    stoch_range = (highest_high - lowest_low).replace(0, pd.NA)
    enriched["stoch_k"] = 100 * (close - lowest_low) / stoch_range
    enriched["stoch_d"] = enriched["stoch_k"].rolling(window=3).mean()
    enriched["williams_r_14"] = -100 * (highest_high - close) / stoch_range
    enriched["aroon_up_25"] = _aroon(high, 25, direction="up")
    enriched["aroon_down_25"] = _aroon(low, 25, direction="down")
    donchian_high = high.rolling(window=20).max()
    donchian_low = low.rolling(window=20).min()
    enriched["donchian_upper_20"] = donchian_high
    enriched["donchian_lower_20"] = donchian_low

    cci_mean = typical_price.rolling(window=20).mean()
    cci_mad = typical_price.rolling(window=20).apply(lambda values: (abs(values - values.mean())).mean(), raw=True)
    enriched["cci_20"] = (typical_price - cci_mean) / (0.015 * cci_mad.replace(0, pd.NA))

    enriched["roc_10"] = close.pct_change(periods=10) * 100

    mfi_positive = typical_price.where(typical_price > typical_price.shift(1), 0.0) * volume
    mfi_negative = typical_price.where(typical_price < typical_price.shift(1), 0.0) * volume
    pos_mf = mfi_positive.rolling(window=14).sum()
    neg_mf = mfi_negative.rolling(window=14).sum()
    mfi = pd.Series(index=enriched.index, dtype="float64")
    positive_only = (pos_mf > 0) & (neg_mf <= 0)
    negative_only = (neg_mf > 0) & (pos_mf <= 0)
    balanced = (pos_mf > 0) & (neg_mf > 0)
    mfi.loc[positive_only] = 100.0
    mfi.loc[negative_only] = 0.0
    mfi.loc[balanced] = 100 - (100 / (1 + (pos_mf[balanced] / neg_mf[balanced])))
    enriched["mfi_14"] = mfi
    money_flow_multiplier = ((close - low) - (high - close)) / (high - low + 1e-9)
    money_flow_volume = money_flow_multiplier * volume
    cmf_sum = money_flow_volume.rolling(window=20).sum()
    volume_sum = volume.rolling(window=20).sum().replace(0, pd.NA)
    enriched["cmf_20"] = cmf_sum / volume_sum

    ema20 = close.ewm(span=20, adjust=False).mean()
    enriched["kc_upper"] = ema20 + (enriched["atr_14"] * 2.0)
    enriched["kc_lower"] = ema20 - (enriched["atr_14"] * 2.0)
    enriched["kc_position"] = (close - enriched["kc_lower"]) / (enriched["kc_upper"] - enriched["kc_lower"])

    enriched["candle_body_pct"] = (close - open_).abs() / (high - low + 1e-9)
    enriched["upper_wick_pct"] = (high - close.combine(open_, max)) / (high - low + 1e-9)
    enriched["lower_wick_pct"] = (open_.combine(close, min) - low) / (high - low + 1e-9)
    enriched["dist_from_ema9"] = (close - enriched["ema_fast"]) / close * 100
    enriched["dist_from_ema21"] = (close - enriched["ema_slow"]) / close * 100
    enriched["dist_from_ema50"] = (close - enriched["ema_50"]) / close * 100
    enriched["dist_from_ema200"] = (close - enriched["ema_200"]) / close * 100
    enriched["dist_from_vwap"] = (close - enriched["vwap"]) / close * 100
    enriched["rsi_slope"] = enriched["rsi"].diff(periods=3)
    enriched["macd_hist_slope"] = enriched["macd_histogram"].diff(periods=3)
    enriched["obv"] = (volume.where(close > close.shift(1), 0.0) - volume.where(close < close.shift(1), 0.0)).cumsum()
    enriched["obv_slope"] = enriched["obv"].diff(periods=3) / enriched["obv"].shift(3).abs().replace(0, pd.NA)

    enriched["change_1h"] = _timeframe_change(enriched, 1)
    enriched["change_4h"] = _timeframe_change(enriched, 4)
    enriched["change_1d"] = _timeframe_change(enriched, 24)
    enriched["change_1w"] = _timeframe_change(enriched, 168)

    enriched["pattern_doji"] = enriched["candle_body_pct"] <= 0.1
    enriched["pattern_hammer"] = (enriched["lower_wick_pct"] >= 0.5) & (enriched["upper_wick_pct"] <= 0.2)
    enriched["pattern_shooting_star"] = (enriched["upper_wick_pct"] >= 0.5) & (enriched["lower_wick_pct"] <= 0.2)
    prev_open = open_.shift(1)
    prev_close = close.shift(1)
    prev_high = high.shift(1)
    prev_low = low.shift(1)
    bullish_engulf = (close > open_) & (prev_close < prev_open) & (open_ <= prev_close) & (close >= prev_open)
    bearish_engulf = (close < open_) & (prev_close > prev_open) & (open_ >= prev_close) & (close <= prev_open)
    enriched["pattern_engulfing"] = bullish_engulf | bearish_engulf
    enriched["pattern_morning_star"] = (
        (close > open_)
        & (prev_close.shift(1) < prev_open.shift(1))
        & (prev_close < prev_open)
        & (close > ((prev_open + prev_close) / 2.0))
    )
    enriched["pattern_evening_star"] = (
        (close < open_)
        & (prev_close.shift(1) > prev_open.shift(1))
        & (prev_close > prev_open)
        & (close < ((prev_open + prev_close) / 2.0))
    )
    enriched["pattern_marubozu"] = enriched["candle_body_pct"] >= 0.9
    enriched["pattern_inside_bar"] = (high < prev_high) & (low > prev_low)
    enriched["pattern_pinbar"] = (
        ((enriched["upper_wick_pct"] >= 0.4) | (enriched["lower_wick_pct"] >= 0.4))
        & (enriched["candle_body_pct"] <= 0.3)
    )

    enriched = apply_smc_features(enriched)

    pivot_high = high.rolling(window=24).max().shift(1)
    pivot_low = low.rolling(window=24).min().shift(1)
    pivot_close = close.shift(1)
    enriched["pivot_classic"] = (pivot_high + pivot_low + pivot_close) / 3.0
    enriched["pivot_r1"] = (2 * enriched["pivot_classic"]) - pivot_low
    enriched["pivot_s1"] = (2 * enriched["pivot_classic"]) - pivot_high

    fib_high = high.rolling(window=52).max()
    fib_low = low.rolling(window=52).min()
    fib_range = (fib_high - fib_low).replace(0, pd.NA)
    enriched["fib_382"] = fib_high - (fib_range * 0.382)
    enriched["fib_500"] = fib_high - (fib_range * 0.500)
    enriched["fib_618"] = fib_high - (fib_range * 0.618)
    enriched["fib_ext_1272"] = fib_high + (fib_range * 0.272)
    enriched["fib_ext_1618"] = fib_high + (fib_range * 0.618)
    enriched["swing_high_10"] = high.rolling(window=10).max()
    enriched["swing_low_10"] = low.rolling(window=10).min()

    return enriched


def latest_complete_row(frame: pd.DataFrame) -> pd.Series | None:
    if frame.empty:
        return None

    for _, row in frame.iloc[::-1].iterrows():
        values = [
            row.get("rsi"),
            row.get("macd"),
            row.get("macd_signal"),
            row.get("macd_histogram"),
            row.get("bb_upper"),
            row.get("bb_middle"),
            row.get("bb_lower"),
            row.get("ema_fast"),
            row.get("ema_slow"),
            row.get("bb_position"),
            row.get("atr_14"),
            row.get("atr_ratio"),
            row.get("volume_ratio"),
            row.get("vwap"),
            row.get("adx_14"),
            row.get("stoch_k"),
            row.get("stoch_d"),
            row.get("cci_20"),
            row.get("roc_10"),
            row.get("mfi_14"),
        ]
        if all(not _is_nan(value) for value in values):
            return row

    return None


def _is_nan(value: object) -> bool:
    return bool(pd.isna(value))


def _timeframe_change(frame: pd.DataFrame, hours: int) -> pd.Series:
    timeframe = ""
    if "timeframe" in frame.columns and not frame["timeframe"].empty:
        timeframe = str(frame["timeframe"].iloc[-1]).strip().lower()
    bars = _timeframe_bars(timeframe, hours)
    if bars <= 0:
        return pd.Series([pd.NA] * len(frame), index=frame.index)
    bars = min(bars, max(1, len(frame) - 1))
    return frame["close"].pct_change(periods=bars) * 100


def _timeframe_bars(timeframe: str, hours: int) -> int:
    minutes_per_bar = _timeframe_minutes(timeframe)
    if minutes_per_bar <= 0:
        return 0
    target_minutes = hours * 60
    bars = round(target_minutes / minutes_per_bar)
    return max(1, bars)


def _timeframe_minutes(timeframe: str) -> int:
    match timeframe:
        case "1m":
            return 1
        case "5m":
            return 5
        case "15m":
            return 15
        case "1h":
            return 60
        case "4h":
            return 240
        case "1d":
            return 1440
        case "1w":
            return 10080
        case _:
            return 0


def _rsi(close: pd.Series, period: int) -> pd.Series:
    delta = close.diff()
    gain = delta.clip(lower=0)
    loss = -delta.clip(upper=0)
    average_gain = gain.ewm(alpha=1 / period, min_periods=period, adjust=False).mean()
    average_loss = loss.ewm(alpha=1 / period, min_periods=period, adjust=False).mean()
    rs = average_gain / average_loss.replace(0, pd.NA)
    rsi = 100 - (100 / (1 + rs))
    rsi.loc[(average_loss == 0) & (average_gain > 0)] = 100.0
    rsi.loc[(average_gain == 0) & (average_loss > 0)] = 0.0
    rsi.loc[(average_gain == 0) & (average_loss == 0)] = 50.0
    return rsi


def _wma(series: pd.Series, period: int) -> pd.Series:
    if period <= 1:
        return series.copy()
    weights = np.arange(1, period + 1, dtype="float64")
    divisor = weights.sum()
    return series.rolling(window=period).apply(lambda values: float(np.dot(values, weights) / divisor), raw=True)


def _dema(series: pd.Series, period: int) -> pd.Series:
    ema = series.ewm(span=period, adjust=False).mean()
    return 2 * ema - ema.ewm(span=period, adjust=False).mean()


def _tema(series: pd.Series, period: int) -> pd.Series:
    ema1 = series.ewm(span=period, adjust=False).mean()
    ema2 = ema1.ewm(span=period, adjust=False).mean()
    ema3 = ema2.ewm(span=period, adjust=False).mean()
    return (3 * ema1) - (3 * ema2) + ema3


def _hma(series: pd.Series, period: int) -> pd.Series:
    if period <= 1:
        return series.copy()
    half = max(1, period // 2)
    sqrt_period = max(1, int(period ** 0.5))
    wma_half = _wma(series, half)
    wma_full = _wma(series, period)
    raw = (2 * wma_half) - wma_full
    return _wma(raw, sqrt_period)


def _aroon(series: pd.Series, period: int, *, direction: str) -> pd.Series:
    if period <= 1:
        return pd.Series([pd.NA] * len(series), index=series.index)

    def _calc(values: np.ndarray) -> float:
        if direction == "up":
            idx = int(np.argmax(values))
        else:
            idx = int(np.argmin(values))
        periods_since = (len(values) - 1) - idx
        return ((period - periods_since) / period) * 100.0

    return series.rolling(window=period).apply(_calc, raw=True)
