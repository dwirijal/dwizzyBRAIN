from __future__ import annotations

import argparse
import json
from datetime import datetime

from quant.config import QuantConfig


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-fetch-ohlcv")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--exchange", required=True)
    parser.add_argument("--timeframe", required=True)
    parser.add_argument("--since", default="")
    parser.add_argument("--limit", type=int, default=0)
    args = parser.parse_args()

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for backfill")

    from quant.backfill.db import BackfillTarget, PostgresBackfillStore

    store = PostgresBackfillStore(config.postgres_url)
    since = datetime.fromisoformat(args.since) if args.since else None
    target = BackfillTarget(args.coin_id, args.exchange, args.timeframe)
    rows = store.load_candles(target, since=since, limit=args.limit or None)
    for row in rows:
        print(json.dumps(row, separators=(",", ":")))


if __name__ == "__main__":
    main()
