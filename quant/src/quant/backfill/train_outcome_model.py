from __future__ import annotations

import argparse
import json
from dataclasses import asdict

from quant.config import QuantConfig
from quant.prediction import save_model, train_outcome_model


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-train-outcome")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--exchange", required=True)
    parser.add_argument("--timeframe", required=True)
    parser.add_argument("--limit", type=int, default=0)
    parser.add_argument("--since", default="")
    parser.add_argument("--output", default="")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for backfill")

    from datetime import datetime

    from quant.backfill.db import BackfillTarget, PostgresBackfillStore

    store = PostgresBackfillStore(config.postgres_url)
    since = datetime.fromisoformat(args.since) if args.since else None
    rows = store.load_prediction_rows(
        BackfillTarget(args.coin_id, args.exchange, args.timeframe),
        since=since,
        limit=args.limit or None,
    )
    model, metrics = train_outcome_model(rows)
    if args.output:
        save_model(model, args.output, metrics)

    if args.json:
        print(json.dumps(asdict(metrics), separators=(",", ":")))
    else:
        print(
            f"rows={metrics.rows_used} train={metrics.train_rows} test={metrics.test_rows} "
            f"mae_1d={metrics.mae_1d:.4f} directional_accuracy={metrics.directional_accuracy:.3f}"
        )


if __name__ == "__main__":
    main()
