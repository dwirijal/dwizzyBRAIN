from __future__ import annotations

import argparse
import json
from dataclasses import asdict, dataclass
from datetime import UTC, datetime
from typing import Iterable, Sequence

import pandas as pd

from quant.config import QuantConfig


@dataclass(slots=True)
class YFinanceRequest:
    coin_id: str
    symbol: str
    asset_class: str
    interval: str
    period: str | None = None
    start: datetime | None = None
    end: datetime | None = None


@dataclass(slots=True)
class YFinanceResult:
    candles_loaded: int
    candles_written: int
    asset_class: str
    symbol: str
    coin_id: str
    interval: str


def fetch_yfinance_candles(request: YFinanceRequest) -> list[dict]:
    ticker = _download_ticker(request.symbol)
    kwargs: dict[str, object] = {
        "interval": _normalize_interval(request.interval),
        "auto_adjust": False,
        "actions": False,
        "prepost": False,
    }
    if request.start is not None:
        kwargs["start"] = request.start.astimezone(UTC)
    if request.end is not None:
        kwargs["end"] = request.end.astimezone(UTC)
    elif request.period:
        kwargs["period"] = request.period

    history = ticker.history(**kwargs)
    if history is None or history.empty:
        return []

    rows: list[dict] = []
    for timestamp, row in history.iterrows():
        if pd.isna(row.get("Open")) or pd.isna(row.get("High")) or pd.isna(row.get("Low")) or pd.isna(row.get("Close")):
            continue

        volume = _safe_float(row.get("Volume"), default=0.0)
        close = _safe_float(row.get("Close"), default=0.0)

        rows.append(
            {
                "time": _to_timestamp(timestamp),
                "coin_id": request.coin_id,
                "exchange": "yfinance",
                "symbol": request.symbol,
                "timeframe": _normalize_interval(request.interval),
                "open": _safe_float(row.get("Open")),
                "high": _safe_float(row.get("High")),
                "low": _safe_float(row.get("Low")),
                "close": close,
                "volume": volume,
                "quote_volume": volume * close,
                "trades": None,
                "is_closed": True,
                "asset_class": request.asset_class,
            }
        )

    return rows


def run_yfinance_backfill(store, request: YFinanceRequest) -> YFinanceResult:
    rows = fetch_yfinance_candles(request)
    written = store.upsert_candles(rows)
    return YFinanceResult(
        candles_loaded=len(rows),
        candles_written=written,
        asset_class=request.asset_class,
        symbol=request.symbol,
        coin_id=request.coin_id,
        interval=_normalize_interval(request.interval),
    )


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-fetch-yfinance")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--symbol", required=True)
    parser.add_argument("--asset-class", required=True, choices=["stock", "forex", "commodity"])
    parser.add_argument("--interval", required=True)
    parser.add_argument("--period", default="")
    parser.add_argument("--start", default="")
    parser.add_argument("--end", default="")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for backfill")

    from quant.backfill.db import PostgresBackfillStore

    request = YFinanceRequest(
        coin_id=args.coin_id,
        symbol=args.symbol,
        asset_class=args.asset_class,
        interval=args.interval,
        period=args.period or None,
        start=_parse_datetime(args.start),
        end=_parse_datetime(args.end),
    )
    result = run_yfinance_backfill(PostgresBackfillStore(config.postgres_url), request)

    if args.json:
        print(json.dumps(asdict(result), separators=(",", ":")))
    else:
        print(
            f"candles={result.candles_loaded} written={result.candles_written} "
            f"asset_class={result.asset_class} symbol={result.symbol} interval={result.interval}"
        )


def _download_ticker(symbol: str):
    try:
        import yfinance as yf
    except ImportError as exc:  # pragma: no cover - exercised in runtime smoke
        raise SystemExit("yfinance is required for multi-asset backfill") from exc

    return yf.Ticker(symbol)


def _normalize_interval(interval: str) -> str:
    value = interval.strip().lower()
    aliases = {
        "60m": "1h",
        "90m": "90m",
        "1d": "1d",
        "1wk": "1wk",
        "1mo": "1mo",
    }
    return aliases.get(value, value)


def _parse_datetime(value: str) -> datetime | None:
    normalized = value.strip()
    if not normalized:
        return None
    if normalized.endswith("Z"):
        normalized = normalized[:-1] + "+00:00"
    parsed = datetime.fromisoformat(normalized)
    if parsed.tzinfo is None:
        return parsed.replace(tzinfo=UTC)
    return parsed.astimezone(UTC)


def _to_timestamp(value: object) -> str:
    if isinstance(value, pd.Timestamp):
        ts = value.to_pydatetime()
    elif isinstance(value, datetime):
        ts = value
    else:
        ts = pd.Timestamp(value).to_pydatetime()

    if ts.tzinfo is None:
        ts = ts.replace(tzinfo=UTC)
    return ts.astimezone(UTC).isoformat().replace("+00:00", "Z")


def _safe_float(value: object, *, default: float = 0.0) -> float:
    try:
        parsed = float(value)
    except (TypeError, ValueError):
        return default
    if pd.isna(parsed):
        return default
    return parsed


if __name__ == "__main__":
    main()
