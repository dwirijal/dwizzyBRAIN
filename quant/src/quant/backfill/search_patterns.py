from __future__ import annotations

import argparse
import json
from dataclasses import dataclass
from datetime import datetime

from quant.backfill.db import BackfillTarget, PostgresBackfillStore
from quant.config import QuantConfig
from quant.patterns import build_fingerprint


@dataclass(slots=True)
class PatternSearchResult:
    low_confidence: bool
    matches: list[dict]
    outcomes: dict[str, dict[str, float | int | None]]
    query: dict[str, object]


def run_similarity_search(
    store: PostgresBackfillStore,
    target: BackfillTarget,
    *,
    since: datetime | None = None,
    limit: int = 20,
    min_matches: int = 30,
) -> PatternSearchResult:
    source_rows = store.load_pattern_source_rows(target, since=since, limit=1)
    if not source_rows:
        return PatternSearchResult(
            low_confidence=True,
            matches=[],
            outcomes={},
            query={"coin_id": target.coin_id, "exchange": target.exchange, "timeframe": target.timeframe},
        )

    query_row = source_rows[-1]
    fingerprint = build_fingerprint(query_row)
    matches = store.find_similar_pattern_matches(
        str(query_row.get("symbol", "")).strip(),
        str(query_row.get("timeframe", target.timeframe)).strip().lower(),
        fingerprint,
        limit=limit,
    )
    outcomes = {
        "1h": _summarize_horizon(matches, "close", "close_1h_later"),
        "4h": _summarize_horizon(matches, "close", "close_4h_later"),
        "1d": _summarize_horizon(matches, "close", "close_1d_later"),
        "1w": _summarize_horizon(matches, "close", "close_1w_later"),
    }

    return PatternSearchResult(
        low_confidence=len(matches) < min_matches,
        matches=matches,
        outcomes=outcomes,
        query={
            "coin_id": target.coin_id,
            "exchange": target.exchange,
            "timeframe": target.timeframe,
            "time": query_row.get("time"),
            "fingerprint": fingerprint,
            "macro_environment": query_row.get("macro_environment") or "",
        },
    )


def main() -> None:
    parser = argparse.ArgumentParser(prog="quant-search-patterns")
    parser.add_argument("--coin-id", required=True)
    parser.add_argument("--exchange", required=True)
    parser.add_argument("--timeframe", required=True)
    parser.add_argument("--since", default="")
    parser.add_argument("--limit", type=int, default=20)
    parser.add_argument("--min-matches", type=int, default=30)
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    config = QuantConfig.from_env()
    if not config.postgres_url:
        raise SystemExit("POSTGRES_URL or POSTGRES_URL_FILE is required for pattern search")

    store = PostgresBackfillStore(config.postgres_url)
    since = datetime.fromisoformat(args.since) if args.since else None
    result = run_similarity_search(
        store,
        BackfillTarget(coin_id=args.coin_id, exchange=args.exchange, timeframe=args.timeframe),
        since=since,
        limit=args.limit,
        min_matches=args.min_matches,
    )

    payload = {
        "low_confidence": result.low_confidence,
        "matches": result.matches,
        "outcomes": result.outcomes,
        "query": result.query,
    }
    if args.json:
        print(json.dumps(payload, default=str, separators=(",", ":")))
    else:
        print(json.dumps(payload, default=str, indent=2))


def _summarize_horizon(matches: list[dict], close_key: str, future_key: str) -> dict[str, float | int | None]:
    returns: list[float] = []
    for row in matches:
        close = row.get(close_key)
        future = row.get(future_key)
        if close is None or future is None:
            continue
        try:
            close_value = float(close)
            future_value = float(future)
        except (TypeError, ValueError):
            continue
        if close_value == 0:
            continue
        returns.append(((future_value - close_value) / close_value) * 100.0)

    if not returns:
        return {
            "count": 0,
            "median": None,
            "win_rate": None,
            "avg_win": None,
            "avg_loss": None,
        }

    wins = [value for value in returns if value > 0]
    losses = [value for value in returns if value <= 0]
    return {
        "count": len(returns),
        "median": float(_median(returns)),
        "win_rate": float(len(wins) / len(returns)),
        "avg_win": float(sum(wins) / len(wins)) if wins else None,
        "avg_loss": float(sum(losses) / len(losses)) if losses else None,
    }


def _median(values: list[float]) -> float:
    values = sorted(values)
    mid = len(values) // 2
    if len(values) % 2 == 1:
        return values[mid]
    return (values[mid - 1] + values[mid]) / 2.0


if __name__ == "__main__":
    main()
