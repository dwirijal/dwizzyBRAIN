from __future__ import annotations

from datetime import UTC, datetime
from types import SimpleNamespace

from quant.backfill import search_patterns as mod
from quant.backfill.search_patterns import _summarize_horizon


def test_summarize_horizon_handles_empty_matches() -> None:
    summary = _summarize_horizon([], "close", "close_1h_later")

    assert summary["count"] == 0
    assert summary["median"] is None
    assert summary["win_rate"] is None


def test_summarize_horizon_computes_basic_stats() -> None:
    summary = _summarize_horizon(
        [
            {"close": 100.0, "close_1h_later": 110.0},
            {"close": 100.0, "close_1h_later": 90.0},
            {"close": 100.0, "close_1h_later": 120.0},
        ],
        "close",
        "close_1h_later",
    )

    assert summary["count"] == 3
    assert summary["median"] == 10.0
    assert summary["win_rate"] == 2 / 3
    assert summary["avg_win"] == 15.0
    assert summary["avg_loss"] == -10.0


def test_run_similarity_search_returns_low_confidence_without_source_rows() -> None:
    class _Store:
        def load_pattern_source_rows(self, target, *, since=None, limit=None):
            return []

    result = mod.run_similarity_search(
        _Store(),
        SimpleNamespace(coin_id="bitcoin", exchange="binance", timeframe="1m"),
    )

    assert result.low_confidence is True
    assert result.matches == []
    assert result.outcomes == {}
    assert result.query["coin_id"] == "bitcoin"


def test_run_similarity_search_builds_outcomes(monkeypatch) -> None:
    source_time = datetime(2026, 3, 20, 12, 0, tzinfo=UTC)
    source_row = {
        "symbol": "BTCUSDT",
        "timeframe": "1m",
        "time": source_time,
        "macro_environment": "risk_on",
    }
    matches = [
        {"close": 100.0, "close_1h_later": 110.0, "close_4h_later": 120.0, "close_1d_later": 130.0, "close_1w_later": 140.0},
        {"close": 100.0, "close_1h_later": 90.0, "close_4h_later": 95.0, "close_1d_later": 98.0, "close_1w_later": 105.0},
    ]

    class _Store:
        def load_pattern_source_rows(self, target, *, since=None, limit=None):
            return [source_row]

        def find_similar_pattern_matches(self, symbol, timeframe, fingerprint, *, limit=20):
            assert symbol == "BTCUSDT"
            assert timeframe == "1m"
            assert fingerprint == [0.1, 0.2, 0.3]
            assert limit == 2
            return matches

    monkeypatch.setattr(mod, "build_fingerprint", lambda row: [0.1, 0.2, 0.3])

    result = mod.run_similarity_search(
        _Store(),
        SimpleNamespace(coin_id="bitcoin", exchange="binance", timeframe="1m"),
        limit=2,
        min_matches=2,
    )

    assert result.low_confidence is False
    assert result.query["fingerprint"] == [0.1, 0.2, 0.3]
    assert result.query["macro_environment"] == "risk_on"
    assert result.outcomes["1h"]["count"] == 2
    assert result.outcomes["1h"]["win_rate"] == 0.5
