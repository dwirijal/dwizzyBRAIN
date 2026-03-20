from __future__ import annotations

from datetime import UTC, datetime

import pandas as pd

from quant.backfill.fetch_yfinance import YFinanceRequest, fetch_yfinance_candles, run_yfinance_backfill


class _FakeTicker:
    def __init__(self, frame: pd.DataFrame) -> None:
        self._frame = frame
        self.history_kwargs: dict | None = None

    def history(self, **kwargs):
        self.history_kwargs = kwargs
        return self._frame


class _FakeStore:
    def __init__(self) -> None:
        self.rows: list[dict] = []

    def upsert_candles(self, rows):
        self.rows = list(rows)
        return len(self.rows)


def test_fetch_yfinance_candles_normalizes_rows(monkeypatch) -> None:
    frame = pd.DataFrame(
        {
            "Open": [1.0, 2.0],
            "High": [1.5, 2.5],
            "Low": [0.5, 1.5],
            "Close": [1.2, 2.2],
            "Volume": [100.0, 200.0],
        },
        index=pd.to_datetime(
            [datetime(2026, 3, 18, tzinfo=UTC), datetime(2026, 3, 19, tzinfo=UTC)]
        ),
    )
    ticker = _FakeTicker(frame)
    monkeypatch.setattr("quant.backfill.fetch_yfinance._download_ticker", lambda symbol: ticker)

    request = YFinanceRequest(
        coin_id="spy",
        symbol="SPY",
        asset_class="stock",
        interval="1d",
        period="1mo",
    )
    rows = fetch_yfinance_candles(request)

    assert len(rows) == 2
    assert rows[0]["coin_id"] == "spy"
    assert rows[0]["exchange"] == "yfinance"
    assert rows[0]["symbol"] == "SPY"
    assert rows[0]["timeframe"] == "1d"
    assert rows[0]["quote_volume"] == 120.0
    assert ticker.history_kwargs is not None
    assert ticker.history_kwargs["interval"] == "1d"
    assert ticker.history_kwargs["period"] == "1mo"


def test_run_yfinance_backfill_writes_rows(monkeypatch) -> None:
    frame = pd.DataFrame(
        {
            "Open": [1.0],
            "High": [1.5],
            "Low": [0.5],
            "Close": [1.2],
            "Volume": [100.0],
        },
        index=pd.to_datetime([datetime(2026, 3, 18, tzinfo=UTC)]),
    )
    monkeypatch.setattr("quant.backfill.fetch_yfinance._download_ticker", lambda symbol: _FakeTicker(frame))

    store = _FakeStore()
    result = run_yfinance_backfill(
        store,
        YFinanceRequest(
            coin_id="eurusd",
            symbol="EURUSD=X",
            asset_class="forex",
            interval="1h",
            period="1mo",
        ),
    )

    assert result.candles_loaded == 1
    assert result.candles_written == 1
    assert store.rows[0]["coin_id"] == "eurusd"
    assert store.rows[0]["symbol"] == "EURUSD=X"
    assert store.rows[0]["exchange"] == "yfinance"
