from __future__ import annotations

import json
from datetime import UTC, datetime
from types import SimpleNamespace

import pytest

from quant.config import QuantConfig
from quant.prediction import OutcomeModelMetrics


def test_macro_points_converts_rows() -> None:
    from quant.backfill import label_events as mod

    rows = [
        {
            "series_id": "FEDFUNDS",
            "series_name": "Fed Funds",
            "time": datetime(2026, 3, 20, 12, 0, tzinfo=UTC),
            "value": 5.25,
            "source": "fred",
            "event_type": "rate",
            "importance": 3,
            "metadata": {"x": 1},
        }
    ]
    points = mod._macro_points(rows)

    assert len(points) == 1
    assert points[0].series_id == "FEDFUNDS"
    assert points[0].value == 5.25


def test_label_events_main_requires_postgres(monkeypatch) -> None:
    from quant.backfill import label_events as mod

    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(lambda cls: QuantConfig(valkey_url="redis://localhost:6379/0", postgres_url="")),
    )
    monkeypatch.setattr(
        "sys.argv",
        [
            "quant-label-events",
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


def test_label_events_main_json(monkeypatch, capsys) -> None:
    from quant.backfill import label_events as mod

    macro_row = {
        "series_id": "FEDFUNDS",
        "series_name": "Fed Funds",
        "time": datetime(2026, 3, 20, 12, 0, tzinfo=UTC),
        "value": 5.25,
        "source": "fred",
        "event_type": "rate",
        "importance": 3,
        "metadata": {},
    }
    candles = [{"time": "2026-03-20T12:00:00Z", "symbol": "BTCUSDT", "timeframe": "1m"}]
    labels = [{"time": "2026-03-20T12:00:00Z", "symbol": "BTCUSDT", "timeframe": "1m", "event_label": "neutral"}]

    class _Store:
        def load_candles(self, target, *, since=None, limit=None):
            return candles

        def load_macro_events(self, *, series_ids=None, since=None):
            return [macro_row]

        def upsert_candle_event_labels(self, rows):
            assert rows == labels
            return len(rows)

    captured = {}

    def _label(candle_rows, macro_points):
        captured["candle_rows"] = candle_rows
        captured["macro_points"] = macro_points
        return labels

    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(lambda cls: QuantConfig(valkey_url="redis://localhost:6379/0", postgres_url="postgres://ok")),
    )
    monkeypatch.setattr(mod, "PostgresBackfillStore", lambda dsn: _Store())
    monkeypatch.setattr(mod, "BackfillTarget", lambda coin_id, exchange, timeframe: SimpleNamespace(coin_id=coin_id, exchange=exchange, timeframe=timeframe))
    monkeypatch.setattr(mod, "label_candle_events", _label)
    monkeypatch.setattr("sys.argv", ["quant-label-events", "--coin-id", "bitcoin", "--exchange", "binance", "--timeframe", "1m", "--json"])

    mod.main()
    payload = json.loads(capsys.readouterr().out.strip())
    assert payload == {"candles": 1, "labels": 1}
    assert captured["candle_rows"] == candles
    assert len(captured["macro_points"]) == 1
    assert captured["macro_points"][0].series_id == "FEDFUNDS"


def test_train_outcome_main_requires_postgres(monkeypatch) -> None:
    from quant.backfill import train_outcome_model as mod

    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(lambda cls: QuantConfig(valkey_url="redis://localhost:6379/0", postgres_url="")),
    )
    monkeypatch.setattr(
        "sys.argv",
        [
            "quant-train-outcome",
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


def test_train_outcome_main_json_and_save(monkeypatch, capsys, tmp_path) -> None:
    from quant.backfill import train_outcome_model as mod

    class _Store:
        def load_prediction_rows(self, target, *, since=None, limit=None):
            return [{"close": 100.0, "close_1d_later": 101.0}]

    metrics = OutcomeModelMetrics(
        rows_used=50,
        train_rows=40,
        test_rows=10,
        mae_1d=0.55,
        directional_accuracy=0.66,
    )
    saved = {}

    def _save(model, output, model_metrics):
        saved["model"] = model
        saved["output"] = output
        saved["metrics"] = model_metrics

    monkeypatch.setattr(
        mod.QuantConfig,
        "from_env",
        classmethod(lambda cls: QuantConfig(valkey_url="redis://localhost:6379/0", postgres_url="postgres://ok")),
    )
    monkeypatch.setattr(mod, "train_outcome_model", lambda rows: ("model", metrics))
    monkeypatch.setattr(mod, "save_model", _save)

    class _Target:
        def __init__(self, coin_id, exchange, timeframe):
            self.coin_id = coin_id
            self.exchange = exchange
            self.timeframe = timeframe

    monkeypatch.setitem(
        __import__("sys").modules,
        "quant.backfill.db",
        SimpleNamespace(BackfillTarget=_Target, PostgresBackfillStore=lambda dsn: _Store()),
    )
    monkeypatch.setattr(
        "sys.argv",
        [
            "quant-train-outcome",
            "--coin-id",
            "bitcoin",
            "--exchange",
            "binance",
            "--timeframe",
            "1m",
            "--output",
            str(tmp_path / "model.pkl"),
            "--json",
        ],
    )

    mod.main()
    payload = json.loads(capsys.readouterr().out.strip())
    assert payload["rows_used"] == 50
    assert saved["model"] == "model"
    assert saved["metrics"] == metrics
