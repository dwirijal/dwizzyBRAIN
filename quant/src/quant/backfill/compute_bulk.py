from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from datetime import datetime
from typing import Iterable

from quant.config import QuantConfig
from quant.indicators import apply_indicators, build_frame


@dataclass(slots=True)
class BulkResult:
    candles_loaded: int
    indicators_written: int
    features_written: int


def compute_bulk_records(candles: list[dict]) -> tuple[list[dict], list[dict]]:
    frame = build_frame(candles)
    enriched = apply_indicators(frame)

    indicator_rows: list[dict] = []
    feature_rows: list[dict] = []
    for _, row in enriched.iterrows():
        indicator_row = _indicator_row(row)
        feature_row = _feature_row(row)
        if indicator_row is None or feature_row is None:
            continue
        indicator_rows.append(indicator_row)
        feature_rows.append(feature_row)

    return indicator_rows, feature_rows


def run_bulk_backfill(
    store,
    target,
    *,
    since: datetime | None = None,
    limit: int | None = None,
) -> BulkResult:
    candles = store.load_candles(target, since=since, limit=limit)
    indicators, features = compute_bulk_records(candles)
    written_indicators = store.upsert_indicators(indicators)
    written_features = store.upsert_features(features)
    return BulkResult(
        candles_loaded=len(candles),
        indicators_written=written_indicators,
        features_written=written_features,
    )


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-backfill")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--exchange", required=True)
    parser.add_argument("--timeframe", required=True)
    parser.add_argument("--since", default="")
    parser.add_argument("--limit", type=int, default=0)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    from quant.backfill.db import BackfillTarget, PostgresBackfillStore

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for backfill")

    store = PostgresBackfillStore(config.postgres_url)
    since = datetime.fromisoformat(args.since) if args.since else None
    limit = args.limit or None
    result = run_bulk_backfill(
        store,
        BackfillTarget(coin_id=args.coin_id, exchange=args.exchange, timeframe=args.timeframe),
        since=since,
        limit=limit,
    )

    if args.json:
        print(json.dumps(asdict(result), separators=(",", ":")))
    else:
        print(
            f"candles={result.candles_loaded} indicators={result.indicators_written} features={result.features_written}"
        )


def _indicator_row(row) -> dict | None:
    required_columns = [
        "symbol",
        "timeframe",
        "ema_fast",
        "ema_slow",
        "ema_50",
        "ema_200",
        "sma_50",
        "sma_200",
        "vwap",
        "supertrend",
        "supertrend_dir",
        "adx_14",
        "ichimoku_tenkan",
        "ichimoku_kijun",
        "ichimoku_senkou_a",
        "ichimoku_senkou_b",
        "rsi",
        "macd",
        "macd_signal",
        "macd_histogram",
        "stoch_k",
        "stoch_d",
        "cci_20",
        "roc_10",
        "mfi_14",
        "atr_14",
        "bb_upper",
        "bb_middle",
        "bb_lower",
        "bb_pct_b",
        "bb_width",
        "kc_upper",
        "kc_lower",
        "hist_vol_20",
        "obv",
        "cmf_20",
        "volume_sma20",
        "volume_ratio",
        "volume_trend",
        "pivot_classic",
        "pivot_r1",
        "pivot_s1",
        "fib_382",
        "fib_500",
        "fib_618",
    ]
    if not _is_complete(row, required_columns):
        return None

    return {
        "time": row["timestamp"].to_pydatetime(),
        "symbol": row["symbol"],
        "timeframe": row["timeframe"],
        "ema_9": _float(row["ema_fast"]),
        "ema_21": _float(row["ema_slow"]),
        "ema_50": _float(row["ema_50"]),
        "ema_200": _float(row["ema_200"]),
        "sma_50": _float(row["sma_50"]),
        "sma_200": _float(row["sma_200"]),
        "vwap": _float(row["vwap"]),
        "supertrend": _float(row["supertrend"]),
        "supertrend_dir": int(row["supertrend_dir"]),
        "adx": _float(row["adx_14"]),
        "ichimoku_tenkan": _float(row["ichimoku_tenkan"]),
        "ichimoku_kijun": _float(row["ichimoku_kijun"]),
        "ichimoku_senkou_a": _float(row["ichimoku_senkou_a"]),
        "ichimoku_senkou_b": _float(row["ichimoku_senkou_b"]),
        "rsi_14": _float(row["rsi"]),
        "rsi_2": _float(row["rsi_2"]) if "rsi_2" in row and not _is_na(row["rsi_2"]) else None,
        "macd": _float(row["macd"]),
        "macd_signal": _float(row["macd_signal"]),
        "macd_hist": _float(row["macd_histogram"]),
        "stoch_k": _float(row["stoch_k"]),
        "stoch_d": _float(row["stoch_d"]),
        "cci_20": _float(row["cci_20"]),
        "roc_10": _float(row["roc_10"]),
        "mfi_14": _float(row["mfi_14"]),
        "atr_14": _float(row["atr_14"]),
        "bb_upper": _float(row["bb_upper"]),
        "bb_mid": _float(row["bb_middle"]),
        "bb_lower": _float(row["bb_lower"]),
        "bb_pct_b": _float(row["bb_pct_b"]),
        "bb_width": _float(row["bb_width"]),
        "kc_upper": _float(row["kc_upper"]),
        "kc_lower": _float(row["kc_lower"]),
        "hist_vol_20": _float(row["hist_vol_20"]),
        "obv": _float(row["obv"]),
        "cmf_20": _float(row["cmf_20"]) if "cmf_20" in row and not _is_na(row["cmf_20"]) else None,
        "volume_sma20": _float(row["volume_sma20"]),
        "volume_ratio": _float(row["volume_ratio"]),
        "volume_trend": _float(row["volume_trend"]),
        "pivot_classic": _float(row["pivot_classic"]),
        "pivot_r1": _float(row["pivot_r1"]),
        "pivot_s1": _float(row["pivot_s1"]),
        "fib_382": _float(row["fib_382"]),
        "fib_500": _float(row["fib_500"]),
        "fib_618": _float(row["fib_618"]),
    }


def _feature_row(row) -> dict | None:
    required_columns = [
        "symbol",
        "timeframe",
        "candle_body_pct",
        "upper_wick_pct",
        "lower_wick_pct",
        "dist_from_ema9",
        "dist_from_ema21",
        "dist_from_ema50",
        "dist_from_ema200",
        "dist_from_vwap",
        "bb_position",
        "atr_ratio",
        "kc_position",
        "rsi_slope",
        "macd_hist_slope",
        "obv_slope",
        "change_1h",
        "change_4h",
        "change_1d",
        "change_1w",
        "pattern_doji",
        "pattern_hammer",
        "pattern_shooting_star",
        "pattern_engulfing",
        "pattern_morning_star",
        "pattern_evening_star",
        "pattern_marubozu",
        "pattern_inside_bar",
        "pattern_pinbar",
        "smc_order_block",
        "smc_fvg",
        "smc_bos",
        "smc_choch",
        "smc_liquidity_sweep",
        "smc_premium_zone",
        "smc_discount_zone",
    ]
    if not _is_complete(row, required_columns):
        return None

    return {
        "time": row["timestamp"].to_pydatetime(),
        "symbol": row["symbol"],
        "timeframe": row["timeframe"],
        "candle_body_pct": _float(row["candle_body_pct"]),
        "upper_wick_pct": _float(row["upper_wick_pct"]),
        "lower_wick_pct": _float(row["lower_wick_pct"]),
        "dist_from_ema9": _float(row["dist_from_ema9"]),
        "dist_from_ema21": _float(row["dist_from_ema21"]),
        "dist_from_ema50": _float(row["dist_from_ema50"]),
        "dist_from_ema200": _float(row["dist_from_ema200"]),
        "dist_from_vwap": _float(row["dist_from_vwap"]),
        "bb_position": _float(row["bb_position"]),
        "atr_ratio": _float(row["atr_ratio"]),
        "kc_position": _float(row["kc_position"]),
        "rsi_slope": _float(row["rsi_slope"]),
        "macd_hist_slope": _float(row["macd_hist_slope"]),
        "obv_slope": _float(row["obv_slope"]),
        "change_1h": _float(row["change_1h"]),
        "change_4h": _float(row["change_4h"]),
        "change_1d": _float(row["change_1d"]),
        "change_1w": _float(row["change_1w"]),
        "pattern_doji": bool(row["pattern_doji"]),
        "pattern_hammer": bool(row["pattern_hammer"]),
        "pattern_shooting_star": bool(row["pattern_shooting_star"]),
        "pattern_engulfing": bool(row["pattern_engulfing"]),
        "pattern_morning_star": bool(row["pattern_morning_star"]),
        "pattern_evening_star": bool(row["pattern_evening_star"]),
        "pattern_marubozu": bool(row["pattern_marubozu"]),
        "pattern_inside_bar": bool(row["pattern_inside_bar"]),
        "pattern_pinbar": bool(row["pattern_pinbar"]),
        "smc_order_block": bool(row["smc_order_block"]),
        "smc_fvg": bool(row["smc_fvg"]),
        "smc_bos": bool(row["smc_bos"]),
        "smc_choch": bool(row["smc_choch"]),
        "smc_liquidity_sweep": bool(row["smc_liquidity_sweep"]),
        "smc_premium_zone": bool(row["smc_premium_zone"]),
        "smc_discount_zone": bool(row["smc_discount_zone"]),
    }


def _is_complete(row, columns: list[str]) -> bool:
    return all(column in row and not _is_na(row[column]) for column in columns)


def _is_na(value: object) -> bool:
    try:
        import pandas as pd

        return bool(pd.isna(value))
    except Exception:
        return value is None


def _float(value: object) -> float:
    return float(value)


if __name__ == "__main__":
    main()
