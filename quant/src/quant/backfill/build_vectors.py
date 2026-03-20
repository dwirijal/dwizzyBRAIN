from __future__ import annotations

import argparse
import json
from datetime import datetime

from quant.backfill.db import BackfillTarget, PostgresBackfillStore
from quant.config import QuantConfig
from quant.patterns import build_embedding_records


def run_vector_backfill(
    store: PostgresBackfillStore,
    target: BackfillTarget,
    *,
    since: datetime | None = None,
    limit: int | None = None,
) -> dict[str, int]:
    rows = store.load_pattern_source_rows(target, since=since, limit=limit)
    embeddings = build_embedding_records(rows)
    written = store.upsert_embeddings(embeddings)
    return {"rows_loaded": len(rows), "embeddings_written": written}


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-build-vectors")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--exchange", required=True)
    parser.add_argument("--timeframe", required=True)
    parser.add_argument("--since", default="")
    parser.add_argument("--limit", type=int, default=0)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for pattern backfill")

    store = PostgresBackfillStore(config.postgres_url)
    since = datetime.fromisoformat(args.since) if args.since else None
    limit = args.limit or None
    result = run_vector_backfill(
        store,
        BackfillTarget(coin_id=args.coin_id, exchange=args.exchange, timeframe=args.timeframe),
        since=since,
        limit=limit,
    )

    if args.json:
        print(json.dumps(result, separators=(",", ":")))
    else:
        print(f"rows={result['rows_loaded']} embeddings={result['embeddings_written']}")


if __name__ == "__main__":
    main()
