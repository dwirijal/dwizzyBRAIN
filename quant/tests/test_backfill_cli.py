from __future__ import annotations

import json
from datetime import UTC, datetime
from types import SimpleNamespace

import pytest

from quant.config import QuantConfig


def test_fetch_fred_main_json_without_postgres(monkeypatch, capsys) -> None:
    from quant.backfill import fetch_fred as mod

    point = SimpleNamespace(
        timestamp=datetime(2026, 3, 19, 12, 0, tzinfo=UTC),
        series_id="FEDFUNDS",
        series_name="Fed Funds",
        source="fred",
        event_type="rate",
        value=5.25,
        importance=3,
        metadata={},
    )

    monkeypatch.setattr(mod, "fetch_fred_series", lambda series_id: [point])
    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(
            lambda cls: QuantConfig(
                valkey_url="redis://localhost:6379/0",
                postgres_url="",
            )
        ),
    )
    monkeypatch.setattr(mod, "FRED_SERIES_CATALOG", {"FEDFUNDS": ("rate", "Fed Funds", 3)})
    monkeypatch.setattr("sys.argv", ["quant-fetch-fred", "--series", "FEDFUNDS", "--json"])

    mod.main()
    output = capsys.readouterr().out.strip()
    payload = json.loads(output)
    assert len(payload) == 1
    assert payload[0]["series_id"] == "FEDFUNDS"
    assert payload[0]["value"] == 5.25


def test_fetch_forexfactory_main_json_without_postgres(monkeypatch, capsys) -> None:
    from quant.backfill import fetch_forexfactory as mod

    point = SimpleNamespace(
        timestamp=datetime(2026, 3, 19, 12, 0, tzinfo=UTC),
        series_id="FF:USD:NFP",
        series_name="Non-Farm Payrolls",
        source="forexfactory",
        event_type="calendar",
        value=123.0,
        importance=3,
        metadata={},
    )

    monkeypatch.setattr(mod, "fetch_forex_factory_calendar", lambda week=None: [point])
    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(
            lambda cls: QuantConfig(
                valkey_url="redis://localhost:6379/0",
                postgres_url="",
            )
        ),
    )
    monkeypatch.setattr("sys.argv", ["quant-fetch-forexfactory", "--json"])

    mod.main()
    output = capsys.readouterr().out.strip()
    payload = json.loads(output)
    assert len(payload) == 1
    assert payload[0]["series_id"] == "FF:USD:NFP"
    assert payload[0]["value"] == 123.0


def test_fetch_ohlcv_main_requires_postgres(monkeypatch) -> None:
    from quant.backfill import fetch_ohlcv as mod

    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(
            lambda cls: QuantConfig(
                valkey_url="redis://localhost:6379/0",
                postgres_url="",
            )
        ),
    )
    monkeypatch.setattr(
        "sys.argv",
        [
            "quant-fetch-ohlcv",
            "--coin-id",
            "bitcoin",
            "--exchange",
            "binance",
            "--timeframe",
            "1m",
        ],
    )

    with pytest.raises(SystemExit, match="POSTGRES_URL"):
        mod.main()


def test_run_vector_backfill_writes_embeddings(monkeypatch) -> None:
    from quant.backfill import build_vectors as mod

    source_rows = [{"symbol": "BTCUSDT", "timeframe": "1m", "close": 100.0}]
    embeddings = [{"symbol": "BTCUSDT", "embedding": [0.1, 0.2, 0.3]}]

    class _Store:
        def __init__(self) -> None:
            self.calls: list[tuple[object, object, object]] = []

        def load_pattern_source_rows(self, target, since=None, limit=None):
            self.calls.append((target, since, limit))
            return source_rows

        def upsert_embeddings(self, rows):
            assert rows == embeddings
            return len(rows)

    monkeypatch.setattr(mod, "build_embedding_records", lambda rows: embeddings)

    store = _Store()
    result = mod.run_vector_backfill(
        store,
        SimpleNamespace(coin_id="bitcoin", exchange="binance", timeframe="1m"),
        since=datetime(2026, 3, 19, 0, 0, tzinfo=UTC),
        limit=10,
    )

    assert result == {"rows_loaded": 1, "embeddings_written": 1}
    assert len(store.calls) == 1


def test_build_vectors_main_requires_postgres(monkeypatch) -> None:
    from quant.backfill import build_vectors as mod

    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(
            lambda cls: QuantConfig(
                valkey_url="redis://localhost:6379/0",
                postgres_url="",
            )
        ),
    )
    monkeypatch.setattr(
        "sys.argv",
        [
            "quant-build-vectors",
            "--coin-id",
            "bitcoin",
            "--exchange",
            "binance",
            "--timeframe",
            "1m",
        ],
    )

    with pytest.raises(SystemExit, match="POSTGRES_URL"):
        mod.main()


def test_build_vectors_main_json(monkeypatch, capsys) -> None:
    from quant.backfill import build_vectors as mod

    class _Store:
        pass

    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(
            lambda cls: QuantConfig(
                valkey_url="redis://localhost:6379/0",
                postgres_url="postgres://ok",
            )
        ),
    )
    monkeypatch.setattr(mod, "PostgresBackfillStore", lambda dsn: _Store())
    monkeypatch.setattr(mod, "BackfillTarget", lambda coin_id, exchange, timeframe: SimpleNamespace(coin_id=coin_id, exchange=exchange, timeframe=timeframe))
    monkeypatch.setattr(mod, "run_vector_backfill", lambda store, target, since=None, limit=None: {"rows_loaded": 2, "embeddings_written": 2})
    monkeypatch.setattr(
        "sys.argv",
        [
            "quant-build-vectors",
            "--coin-id",
            "bitcoin",
            "--exchange",
            "binance",
            "--timeframe",
            "1m",
            "--since",
            "2026-03-20T00:00:00+00:00",
            "--limit",
            "25",
            "--json",
        ],
    )

    mod.main()
    payload = json.loads(capsys.readouterr().out.strip())
    assert payload == {"rows_loaded": 2, "embeddings_written": 2}
