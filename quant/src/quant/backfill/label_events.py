from __future__ import annotations

import argparse
import json
from datetime import datetime

from quant.backfill.db import BackfillTarget, PostgresBackfillStore
from quant.config import QuantConfig
from quant.events import FRED_SERIES_CATALOG, label_candle_events


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-label-events")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--exchange", required=True)
    parser.add_argument("--timeframe", required=True)
    parser.add_argument("--since", default="")
    parser.add_argument("--limit", type=int, default=0)
    parser.add_argument("--series", nargs="*", default=list(FRED_SERIES_CATALOG.keys()))
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for backfill")

    store = PostgresBackfillStore(config.postgres_url)
    since = datetime.fromisoformat(args.since) if args.since else None
    candles = store.load_candles(
        BackfillTarget(coin_id=args.coin_id, exchange=args.exchange, timeframe=args.timeframe),
        since=since,
        limit=args.limit or None,
    )
    macro_points = store.load_macro_events(series_ids=args.series, since=since)
    labels = label_candle_events(candles, _macro_points(macro_points))
    written = store.upsert_candle_event_labels(labels)

    if args.json:
        print(json.dumps({"candles": len(candles), "labels": written}, separators=(",", ":")))
    else:
        print(f"candles={len(candles)} labels={written}")


def _macro_points(rows):
    from quant.events import MacroPoint

    points = []
    for row in rows:
        points.append(
            MacroPoint(
                series_id=row["series_id"],
                series_name=row["series_name"],
                timestamp=row["time"],
                value=row["value"],
                source=row["source"],
                event_type=row["event_type"],
                importance=row["importance"],
                metadata=row["metadata"],
            )
        )
    return points


if __name__ == "__main__":
    main()
