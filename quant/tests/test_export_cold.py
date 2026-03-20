from __future__ import annotations

from datetime import UTC, datetime, timedelta
from types import SimpleNamespace
from pathlib import Path

import duckdb

from quant.backfill.export_cold import export_cold_archive


class _Store:
    def load_candles(self, target, *, since=None, limit=None):
        base = datetime(2026, 3, 18, tzinfo=UTC)
        candles: list[dict] = []
        for index in range(120):
            close = 100 + index
            candles.append(
                {
                    "timestamp": (base + timedelta(minutes=index)).isoformat().replace("+00:00", "Z"),
                    "symbol": "BTCUSDT",
                    "exchange": "binance",
                    "timeframe": "1m",
                    "open": close - 0.5,
                    "high": close + 1.0,
                    "low": close - 1.0,
                    "close": close,
                    "volume": 1000 + index,
                }
            )
        return candles


def test_export_cold_archive_writes_parquet(tmp_path: Path) -> None:
    result = export_cold_archive(
        _Store(),
        SimpleNamespace(coin_id="bitcoin", exchange="binance", timeframe="1m"),
        output_dir=tmp_path,
    )

    parquet_path = Path(result.parquet_path)
    assert parquet_path.exists()
    assert parquet_path.suffix == ".parquet"

    connection = duckdb.connect(database=":memory:")
    try:
        rows = connection.execute(
            "SELECT count(*) AS count, max(symbol) AS symbol, max(timeframe) AS timeframe FROM read_parquet(?)",
            [str(parquet_path)],
        ).fetchone()
    finally:
        connection.close()

    assert rows is not None
    assert rows[0] == 120
    assert rows[1] == "BTCUSDT"
    assert rows[2] == "1m"
