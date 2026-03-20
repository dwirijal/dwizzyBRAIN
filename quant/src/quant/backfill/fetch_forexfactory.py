from __future__ import annotations

import argparse
import json

from quant.config import QuantConfig
from quant.events import fetch_forex_factory_calendar


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-fetch-forexfactory")
    parser.add_argument("--week", default="")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    points = fetch_forex_factory_calendar(args.week or None)

    config = QuantConfig.from_env()
    if config.postgres_url:
        from quant.backfill.db import PostgresBackfillStore

        store = PostgresBackfillStore(config.postgres_url)
        written = store.upsert_macro_events(
            {
                "time": point.timestamp,
                "series_id": point.series_id,
                "series_name": point.series_name,
                "source": point.source,
                "event_type": point.event_type,
                "value": point.value,
                "importance": point.importance,
                "metadata": point.metadata,
            }
            for point in points
        )
        if args.json:
            print(json.dumps({"points": len(points), "written": written}, separators=(",", ":")))
        else:
            print(f"points={len(points)} written={written}")
        return

    if args.json:
        print(
            json.dumps(
                [
                    {
                        "time": point.timestamp.isoformat().replace("+00:00", "Z"),
                        "series_id": point.series_id,
                        "series_name": point.series_name,
                        "value": point.value,
                    }
                    for point in points
                ],
                separators=(",", ":"),
            )
        )
    else:
        for point in points:
            print(f"{point.timestamp.isoformat().replace('+00:00', 'Z')} {point.series_id} {point.value}")


if __name__ == "__main__":
    main()
