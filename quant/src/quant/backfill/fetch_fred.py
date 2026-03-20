from __future__ import annotations

import argparse
import json
from datetime import UTC, datetime

from quant.config import QuantConfig
from quant.events import FRED_SERIES_CATALOG, fetch_fred_series


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-fetch-fred")
    parser.add_argument("--series", nargs="*", default=list(FRED_SERIES_CATALOG.keys()))
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    points = []
    for series_id in args.series:
        points.extend(fetch_fred_series(series_id))

    config = QuantConfig.from_env()
    if config.postgres_url:
        from quant.backfill.db import PostgresBackfillStore

        store = PostgresBackfillStore(config.postgres_url)
        written = store.upsert_macro_events(
            {
                "time": point.timestamp.astimezone(UTC),
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
                        "time": point.timestamp.astimezone(UTC).isoformat().replace("+00:00", "Z"),
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
            print(
                f"{point.timestamp.astimezone(UTC).isoformat().replace('+00:00', 'Z')} {point.series_id} {point.value}"
            )


if __name__ == "__main__":
    main()
