from __future__ import annotations

import argparse
import json
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import TYPE_CHECKING

import duckdb

from quant.config import QuantConfig
from quant.indicators import apply_indicators, build_frame

if TYPE_CHECKING:
    from quant.backfill.db import BackfillTarget, PostgresBackfillStore


@dataclass(slots=True)
class ExportResult:
    candles_loaded: int
    parquet_path: str


def export_cold_archive(
    store,
    target,
    *,
    since: datetime | None = None,
    limit: int | None = None,
    output_dir: Path,
) -> ExportResult:
    candles = store.load_candles(target, since=since, limit=limit)
    if not candles:
        output_dir.mkdir(parents=True, exist_ok=True)
        output_path = output_dir / _parquet_name(target)
        _write_empty_parquet(output_path)
        return ExportResult(candles_loaded=0, parquet_path=str(output_path))

    frame = build_frame(candles)
    enriched = apply_indicators(frame)

    output_dir.mkdir(parents=True, exist_ok=True)
    output_path = output_dir / _parquet_name(target)
    _write_parquet(enriched, output_path)
    return ExportResult(candles_loaded=len(candles), parquet_path=str(output_path))


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-export-cold")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--exchange", required=True)
    parser.add_argument("--timeframe", required=True)
    parser.add_argument("--since", default="")
    parser.add_argument("--limit", type=int, default=0)
    parser.add_argument("--output-dir", default="quant-archive")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for backfill")

    from quant.backfill.db import BackfillTarget, PostgresBackfillStore

    store = PostgresBackfillStore(config.postgres_url)
    since = datetime.fromisoformat(args.since) if args.since else None
    result = export_cold_archive(
        store,
        BackfillTarget(coin_id=args.coin_id, exchange=args.exchange, timeframe=args.timeframe),
        since=since,
        limit=args.limit or None,
        output_dir=Path(args.output_dir),
    )

    if args.json:
        print(json.dumps({"candles_loaded": result.candles_loaded, "parquet_path": result.parquet_path}))
    else:
        print(f"candles={result.candles_loaded} parquet={result.parquet_path}")


def _parquet_name(target: BackfillTarget) -> str:
    return f"{target.coin_id}-{target.exchange}-{target.timeframe}.parquet"


def _write_parquet(frame, output_path: Path) -> None:
    connection = duckdb.connect(database=":memory:")
    try:
        connection.register("enriched", frame)
        connection.execute(
            f"COPY enriched TO '{output_path.as_posix()}' (FORMAT PARQUET, COMPRESSION ZSTD)"
        )
    finally:
        connection.close()


def _write_empty_parquet(output_path: Path) -> None:
    connection = duckdb.connect(database=":memory:")
    try:
        connection.execute(f"CREATE TABLE empty(time TIMESTAMP, symbol VARCHAR, timeframe VARCHAR)")
        connection.execute(
            f"COPY empty TO '{output_path.as_posix()}' (FORMAT PARQUET, COMPRESSION ZSTD)"
        )
    finally:
        connection.close()


if __name__ == "__main__":
    main()
